package core

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/juanrubio/boardwalk/internal/exiterr"
	"github.com/juanrubio/boardwalk/internal/githubx"
)

// Change is one planned field/title/body change with display values
// for the change summary shared by CLI and TUI (cli.md/tui.md).
type Change struct {
	Field string // display name ("Title", "Body", or a project field)
	Old   string
	New   string
	// Exactly one payload kind is set:
	Assign   *githubx.FieldAssign // project field change
	IsTitle  bool
	IsBody   bool
	NewTitle string
	NewBody  string
}

// Result is the per-change outcome of a save (never atomic).
type Result struct {
	Change Change
	Err    error
}

// ParseFieldAssignment validates a "Name=Value" assignment against the
// live schema and builds a Change. Unknown fields/options and invalid
// date/number values fail with exit 2 and the valid choices (cli.md).
func ParseFieldAssignment(b *githubx.Board, it githubx.Item, assignment string) (Change, error) {
	name, raw, found := strings.Cut(assignment, "=")
	if !found {
		return Change{}, exiterr.Usagef("field assignment %q must look like \"Name=Value\"", assignment)
	}
	name = strings.TrimSpace(name)
	raw = strings.TrimSpace(raw)
	if strings.EqualFold(name, "Title") {
		return Change{Field: "Title", Old: it.Title, New: raw, IsTitle: true, NewTitle: raw}, nil
	}
	field, ok := githubx.FieldByName(b.Fields, name)
	if !ok {
		return Change{}, exiterr.Usagef("unknown field %q on project %s (fields: %s)",
			name, b.Project.Ref(), fieldNames(b.Fields))
	}
	old := ""
	if v, ok2 := it.Value(field.Name); ok2 {
		old = DisplayValue(v)
	}
	assign := githubx.FieldAssign{FieldID: field.ID, FieldName: field.Name}
	if raw == "" {
		assign.Value = githubx.MutationValue{Clear: true}
		return Change{Field: field.Name, Old: old, New: "(cleared)", Assign: &assign}, nil
	}
	switch field.DataType {
	case "SINGLE_SELECT":
		opt, ok := field.OptionByName(raw)
		if !ok {
			return Change{}, exiterr.Usagef("unknown option %q for field %q (valid: %s)",
				raw, field.Name, optionNames(field))
		}
		assign.Value = githubx.MutationValue{OptionID: &opt.ID}
		return Change{Field: field.Name, Old: old, New: raw, Assign: &assign}, nil
	case "ITERATION":
		for _, it2 := range field.Iterations {
			if it2.Title == raw || it2.ID == raw {
				id := it2.ID
				assign.Value = githubx.MutationValue{IterationID: &id}
				return Change{Field: field.Name, Old: old, New: it2.Title, Assign: &assign}, nil
			}
		}
		return Change{}, exiterr.Usagef("unknown iteration %q for field %q", raw, field.Name)
	case "DATE":
		if _, err := time.Parse("2006-01-02", raw); err != nil {
			return Change{}, exiterr.Usagef("invalid date %q for field %q (want YYYY-MM-DD)", raw, field.Name)
		}
		assign.Value = githubx.MutationValue{Date: &raw}
		return Change{Field: field.Name, Old: old, New: raw, Assign: &assign}, nil
	case "NUMBER":
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return Change{}, exiterr.Usagef("invalid number %q for field %q", raw, field.Name)
		}
		assign.Value = githubx.MutationValue{Number: &f}
		return Change{Field: field.Name, Old: old, New: raw, Assign: &assign}, nil
	case "TEXT":
		assign.Value = githubx.MutationValue{Text: &raw}
		return Change{Field: field.Name, Old: old, New: raw, Assign: &assign}, nil
	default:
		return Change{}, exiterr.Usagef("field %q of type %s cannot be set with --field in the MVP", field.Name, field.DataType)
	}
}

// DisplayValue renders a field value for summaries (shared wording).
func DisplayValue(v githubx.FieldValue) string {
	switch {
	case v.OptionName != "":
		return v.OptionName
	case v.IterationTitle != "":
		return v.IterationTitle
	case v.Number != nil:
		if *v.Number == float64(int64(*v.Number)) {
			return fmt.Sprintf("%d", int64(*v.Number))
		}
		return fmt.Sprintf("%g", *v.Number)
	case v.Date != nil:
		return v.Date.Format("2006-01-02")
	case len(v.Users) > 0:
		var parts []string
		for _, u := range v.Users {
			parts = append(parts, "@"+u)
		}
		return strings.Join(parts, ", ")
	case v.Text != "":
		return v.Text
	default:
		return "-"
	}
}

func fieldNames(fields []githubx.Field) string {
	var names []string
	for _, f := range fields {
		names = append(names, f.Name)
	}
	return strings.Join(names, ", ")
}

func optionNames(f githubx.Field) string {
	var names []string
	for _, o := range f.Options {
		names = append(names, o.Name)
	}
	return strings.Join(names, ", ")
}

// SingleRepo returns the one repository used by the board's linked
// items, or "" when linked items span several repositories (or there
// are none). Used to default a draft conversion's target repo (A7).
func SingleRepo(b *githubx.Board) string {
	repo := ""
	for _, it := range b.Items {
		if it.Kind == githubx.KindDraft || it.Repo == "" {
			continue
		}
		if repo == "" {
			repo = it.Repo
		} else if repo != it.Repo {
			return ""
		}
	}
	return repo
}

// ConvertDraft converts a draft item into a real issue in repo
// (owner/name), keeping its board position and field values (A7).
func (s *Service) ConvertDraft(ctx context.Context, project githubx.Project, it githubx.Item, repo string) (githubx.Item, error) {
	if it.Kind != githubx.KindDraft {
		return githubx.Item{}, exiterr.Usagef("%s is a %s, not a draft; only drafts can be converted", it.Ref(), it.Kind)
	}
	if s == nil || s.Client == nil {
		return githubx.Item{}, exiterr.Failuref("no GitHub client configured")
	}
	return s.Client.ConvertDraftToIssue(ctx, project.ID, it.ID, repo)
}

// ApplyChanges sends title/body first (separate GitHub mutations),
// then all project-field changes as one batched request. Per-change
// results are returned; applied changes are never rolled back.
func (s *Service) ApplyChanges(ctx context.Context, project githubx.Project, it githubx.Item, changes []Change) []Result {
	results := make([]Result, len(changes))
	var assigns []githubx.FieldAssign
	var assignIdx []int
	for i, ch := range changes {
		switch {
		case ch.IsTitle || ch.IsBody:
			// handled below, once
		case ch.Assign != nil:
			assigns = append(assigns, *ch.Assign)
			assignIdx = append(assignIdx, i)
		}
	}
	// Title/body as one client call when either is present.
	var titlePtr, bodyPtr *string
	needsTB := false
	for _, ch := range changes {
		if ch.IsTitle {
			titlePtr = &ch.NewTitle
			needsTB = true
		}
		if ch.IsBody {
			bodyPtr = &ch.NewBody
			needsTB = true
		}
	}
	var tbErr error
	if needsTB {
		_, tbErr = s.Client.UpdateTitleBody(ctx, it, titlePtr, bodyPtr)
	}
	fieldResults := s.Client.SetFieldValues(ctx, project.ID, it.ID, assigns)
	frByIdx := make(map[int]error, len(assignIdx))
	for j, idx := range assignIdx {
		if j < len(fieldResults) {
			frByIdx[idx] = fieldResults[j].Err
		}
	}
	for i, ch := range changes {
		results[i] = Result{Change: ch}
		switch {
		case ch.IsTitle || ch.IsBody:
			results[i].Err = tbErr
		case ch.Assign != nil:
			results[i].Err = frByIdx[i]
		}
	}
	return results
}

// SummaryLines renders "Field: old → new" lines for previews.
func SummaryLines(changes []Change) []string {
	lines := make([]string, len(changes))
	for i, ch := range changes {
		old := ch.Old
		if old == "" {
			old = "-"
		}
		lines[i] = fmt.Sprintf("%s: %s → %s", ch.Field, truncateSummary(old), truncateSummary(ch.New))
	}
	return lines
}

func truncateSummary(s string) string {
	r := []rune(s)
	if len(r) > 40 {
		return string(r[:39]) + "…"
	}
	return s
}
