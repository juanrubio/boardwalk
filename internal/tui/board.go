package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/juanrubio/gh-boardwalk/internal/core"
	"github.com/juanrubio/gh-boardwalk/internal/githubx"
)

// column is one kanban column: a grouping option (or the no-value group).
type column struct {
	name      string // option name; "" for no-value
	label     string
	optionID  string
	color     string
	items     []githubx.Item
	collapsed bool
}

func groupFieldFor(board *githubx.Board, remembered string) (githubx.Field, bool) {
	if remembered != "" {
		if f, ok := githubx.FieldByName(board.Fields, remembered); ok && f.DataType == "SINGLE_SELECT" {
			return f, true
		}
	}
	return githubx.DefaultBoardField(board.Fields)
}

func singleSelectFields(fields []githubx.Field) []githubx.Field {
	var out []githubx.Field
	for _, f := range fields {
		if f.DataType == "SINGLE_SELECT" {
			out = append(out, f)
		}
	}
	return out
}

// buildColumns groups visible items by field, option order, with the
// no-value column first and hidden when empty (tui.md §1 A1).
func buildColumns(items []githubx.Item, field githubx.Field, hasField bool, collapsed map[string]bool) []column {
	if !hasField {
		return nil
	}
	var cols []column
	noVal := column{label: "No " + field.Name}
	byOpt := make(map[string]int)
	for _, o := range field.Options {
		byOpt[o.Name] = len(cols)
		cols = append(cols, column{name: o.Name, label: o.Name, optionID: o.ID, color: o.Color, collapsed: collapsed[o.Name]})
	}
	for _, it := range items {
		v, ok := it.Value(field.Name)
		if !ok || v.OptionName == "" {
			noVal.items = append(noVal.items, it)
			continue
		}
		if idx, ok := byOpt[v.OptionName]; ok {
			cols[idx].items = append(cols[idx].items, it)
		} else {
			noVal.items = append(noVal.items, it)
		}
	}
	if len(noVal.items) > 0 {
		noVal.collapsed = collapsed[noVal.label]
		return append([]column{noVal}, cols...)
	}
	return cols
}

// filterItems applies the incremental `/` filter locally. Plain text
// matches title/reference/repo/assignees/field values; status:X and
// assignee:Y qualifiers are honoured too (server does the same, §2).
func filterItems(items []githubx.Item, query string) []githubx.Item {
	q := strings.TrimSpace(query)
	if q == "" {
		return items
	}
	out := make([]githubx.Item, 0, len(items))
	for _, it := range items {
		if matchesFilter(it, q) {
			out = append(out, it)
		}
	}
	return out
}

func matchesFilter(it githubx.Item, q string) bool {
	rest := q
	for _, tok := range strings.Fields(q) {
		if k, v, ok := strings.Cut(tok, ":"); ok {
			v = strings.Trim(v, "\"")
			switch strings.ToLower(k) {
			case "status":
				matched := false
				for _, fv := range it.Values {
					if strings.EqualFold(fv.OptionName, v) {
						matched = true
					}
				}
				if !matched {
					return false
				}
				rest = strings.Replace(rest, tok, "", 1)
			case "assignee":
				matched := false
				for _, fv := range it.Values {
					for _, u := range fv.Users {
						if strings.EqualFold(u, strings.TrimPrefix(v, "@")) {
							matched = true
						}
					}
				}
				if !matched {
					return false
				}
				rest = strings.Replace(rest, tok, "", 1)
			case "repo":
				if !strings.EqualFold(it.Repo, v) {
					return false
				}
				rest = strings.Replace(rest, tok, "", 1)
			case "is":
				if !strings.EqualFold(string(it.Kind), strings.TrimPrefix(v, "draft")) && v != string(it.Kind) {
					// is:draft handled below via kind compare
					if v != "draft" || it.Kind != githubx.KindDraft {
						return false
					}
				}
				rest = strings.Replace(rest, tok, "", 1)
			}
		}
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return true
	}
	var hay strings.Builder
	hay.WriteString(it.Title + " " + it.Ref() + " " + it.Repo + " " + string(it.Kind))
	for _, fv := range it.Values {
		hay.WriteString(" " + core.DisplayValue(fv))
		for _, u := range fv.Users {
			hay.WriteString(" @" + u)
		}
	}
	return strings.Contains(strings.ToLower(hay.String()), strings.ToLower(rest))
}

func assigneeOf(it githubx.Item) string {
	for _, v := range it.Values {
		if len(v.Users) > 0 {
			return "@" + v.Users[0]
		}
	}
	return ""
}

// metaPairs returns short label/value pairs after assignee (card line 3).
func metaPairs(it githubx.Item, groupName string) []string {
	var out []string
	for _, v := range it.Values {
		if v.FieldName == groupName || len(v.Users) > 0 {
			continue
		}
		d := core.DisplayValue(v)
		if d == "-" || d == "" {
			continue
		}
		if v.DataType == "DATE" && v.Date != nil {
			d = fmt.Sprintf("%s (%s)", d, relDate(*v.Date))
		}
		out = append(out, shortLabel(v.FieldName)+" "+d)
		if len(out) >= 3 {
			break
		}
	}
	return out
}

func shortLabel(name string) string {
	switch strings.ToLower(name) {
	case "priority":
		return "▲"
	case "target", "due", "due date":
		return ""
	default:
		return name
	}
}

func relDate(t time.Time) string {
	days := int(time.Until(t).Hours() / 24)
	switch {
	case days == 0:
		return "today"
	case days > 0:
		return fmt.Sprintf("in %dd", days)
	default:
		return fmt.Sprintf("overdue %dd", -days)
	}
}

func relAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

func padRight(s string, w int) string {
	n := len([]rune(s))
	if n >= w {
		return truncateRunes(s, w)
	}
	return s + strings.Repeat(" ", w-n)
}
