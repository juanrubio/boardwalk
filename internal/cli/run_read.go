package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/juanrubio/gh-boardwalk/internal/config"
	"github.com/juanrubio/gh-boardwalk/internal/core"
	"github.com/juanrubio/gh-boardwalk/internal/exiterr"
	"github.com/juanrubio/gh-boardwalk/internal/githubx"
)

// itemJSON is the stable --json shape for items (ADR 0009: bare array,
// fields may be added, never renamed/removed).
type itemJSON struct {
	Ref        string            `json:"ref"`
	ID         string            `json:"id"`
	DatabaseID int64             `json:"databaseId"`
	Kind       string            `json:"kind"`
	Title      string            `json:"title"`
	Repository string            `json:"repository,omitempty"`
	Number     int               `json:"number,omitempty"`
	URL        string            `json:"url,omitempty"`
	Fields     map[string]string `json:"fields"`
}

func toItemJSON(it githubx.Item) itemJSON {
	fields := make(map[string]string, len(it.Values))
	for _, v := range it.Values {
		fields[v.FieldName] = displayValue(v)
	}
	return itemJSON{
		Ref: it.Ref(), ID: it.ID, DatabaseID: it.DatabaseID, Kind: string(it.Kind),
		Title: it.Title, Repository: it.Repo, Number: it.Number, URL: it.URL, Fields: fields,
	}
}

// displayValue renders a field value as humans and JSON both show it.
func displayValue(v githubx.FieldValue) string {
	switch {
	case v.OptionName != "":
		return v.OptionName
	case v.IterationTitle != "":
		return v.IterationTitle
	case v.Number != nil:
		return formatNumber(*v.Number)
	case v.Date != nil:
		return v.Date.Format("2006-01-02")
	case len(v.Users) > 0:
		var parts []string
		for _, u := range v.Users {
			parts = append(parts, "@"+u)
		}
		return strings.Join(parts, ", ")
	default:
		return v.Text
	}
}

func formatNumber(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%g", f)
}

func fieldValue(it githubx.Item, name string) string {
	if v, ok := it.Value(name); ok {
		return displayValue(v)
	}
	return "-"
}

func assigneeDisplay(it githubx.Item) string {
	for _, v := range it.Values {
		if len(v.Users) > 0 {
			var parts []string
			for _, u := range v.Users {
				parts = append(parts, "@"+u)
			}
			return strings.Join(parts, ", ")
		}
	}
	return "-"
}

func runProjectList(ctx context.Context, opts *Options, owner string) error {
	svc, err := opts.service()
	if err != nil {
		return err
	}
	projects, err := svc.ListProjects(ctx, owner)
	if err != nil {
		return err
	}
	if opts.JSON {
		type projectJSON struct {
			Ref        string `json:"ref"`
			ID         string `json:"id"`
			Number     int    `json:"number"`
			Title      string `json:"title"`
			Owner      string `json:"owner"`
			OwnerKind  string `json:"ownerKind"`
			URL        string `json:"url"`
			ItemsTotal int    `json:"itemsTotal"`
			Alias      string `json:"alias,omitempty"`
			Default    bool   `json:"default,omitempty"`
		}
		out := make([]projectJSON, 0, len(projects))
		defaultRef := ""
		if svc.Config != nil {
			defaultRef = svc.Config.ResolveProject("")
		}
		aliasByRef := map[string]string{}
		if svc.Config != nil {
			for name, target := range svc.Config.Aliases {
				aliasByRef[target] = name
			}
		}
		for _, p := range projects {
			out = append(out, projectJSON{
				Ref: p.Ref(), ID: p.ID, Number: p.Number, Title: p.Title, Owner: p.Owner,
				OwnerKind: p.OwnerKind, URL: p.URL, ItemsTotal: p.ItemsTotal,
				Alias: aliasByRef[p.Ref()], Default: p.Ref() == defaultRef,
			})
		}
		enc := json.NewEncoder(opts.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	if len(projects) == 0 {
		_, _ = fmt.Fprintln(opts.Out, "No accessible Projects v2 found.")
		return nil
	}
	w := tabwriter.NewWriter(opts.Out, 0, 2, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "REF\tTITLE\tOWNER\tITEMS")
	for _, p := range projects {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", p.Ref(), p.Title, p.Owner, p.ItemsTotal)
	}
	return w.Flush()
}

func runProjectUse(ctx context.Context, opts *Options, ref, as string) error {
	svc, err := opts.service()
	if err != nil {
		return err
	}
	if svc.Config == nil {
		return exiterr.Failuref("config not loaded")
	}
	if as != "" && !config.ValidAlias(as) {
		return exiterr.Usagef("alias %q must be lowercase letters, digits, and hyphens", as)
	}
	project, err := svc.Client.ResolveProject(ctx, svc.Config.ResolveProject(ref))
	if err != nil {
		// ref may be an alias name not yet saved; resolve the raw ref.
		project, err = svc.Client.ResolveProject(ctx, ref)
		if err != nil {
			return err
		}
	}
	canonical := project.Ref()
	if as != "" {
		svc.Config.SetAlias(as, canonical)
		svc.Config.Default = as
	} else {
		svc.Config.Default = canonical
	}
	if err := svc.Config.Save(); err != nil {
		return err
	}
	if opts.JSON {
		enc := json.NewEncoder(opts.Out)
		return enc.Encode(map[string]any{"ref": canonical, "title": project.Title, "default": svc.Config.Default, "config": svc.Config.FilePath()})
	}
	_, _ = fmt.Fprintf(opts.Out, "✓ Saved default project %s (%s) in %s\n", canonical, project.Title, svc.Config.FilePath())
	return nil
}

func runItemList(ctx context.Context, opts *Options, f listFilters) error {
	svc, err := opts.service()
	if err != nil {
		return err
	}
	project, err := svc.ResolveSelectedProject(ctx, opts.Project)
	if err != nil {
		return err
	}
	var board *githubx.Board
	err = withSpinner(opts, "Loading board…", func() error {
		var e error
		board, e = svc.LoadBoard(ctx, project, f.Search, nil)
		return e
	})
	if err != nil {
		return err
	}
	items := core.ApplyFilters(board, core.Filters{
		Status: f.Status, Assignee: f.Assignee, Repo: f.Repo, Type: f.Type, Search: f.Search,
	})
	if opts.JSON {
		out := make([]itemJSON, 0, len(items))
		for _, it := range items {
			out = append(out, toItemJSON(it))
		}
		enc := json.NewEncoder(opts.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	renderItemList(opts, board, project, items, len(board.Items))
	return nil
}

// Ensure exiterr import is used even before mutation runners land.
var _ = exiterr.CodeUsage
