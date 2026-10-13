package core

import (
	"context"
	"strings"

	"github.com/juanrubio/gh-boardwalk/internal/exiterr"
	"github.com/juanrubio/gh-boardwalk/internal/githubx"
)

// ListProjects discovers Projects v2 (optionally restricted to an owner).
func (s *Service) ListProjects(ctx context.Context, owner string) ([]githubx.Project, error) {
	if err := s.CheckAuth(ctx); err != nil {
		return nil, err
	}
	return s.Client.ListProjects(ctx, owner)
}

// ResolveSelectedProject resolves the selected project (flag/env/default)
// against GitHub. A missing selection is a usage error naming the escapes.
func (s *Service) ResolveSelectedProject(ctx context.Context, flag string) (githubx.Project, error) {
	if err := s.CheckAuth(ctx); err != nil {
		return githubx.Project{}, err
	}
	ref := s.ResolveProjectRef(flag)
	if ref == "" {
		return githubx.Project{}, exiterr.Usagef("no project selected").
			WithRemediation("Pass --project <alias|owner/number|URL>, set BOARDWALK_PROJECT, or save a default with `boardwalk project use <ref>`.")
	}
	return s.Client.ResolveProject(ctx, ref)
}

// Filters narrows a board for item list / TUI list agreement.
type Filters struct {
	Status   string // value of the default board field
	Assignee string // login or @me
	Repo     string
	Type     string // issue|pr|draft
	Search   string
	Me       string // viewer login for @me expansion (may be empty)
}

// ApplyFilters returns the visible items, preserving board order.
// Status filtering uses the default board field and grouping order is
// applied by callers that need it.
func ApplyFilters(b *githubx.Board, f Filters) []githubx.Item {
	statusField, hasStatus := githubx.DefaultBoardField(b.Fields)
	out := make([]githubx.Item, 0, len(b.Items))
	for _, it := range b.Items {
		if it.Archived {
			continue
		}
		if f.Type != "" && string(it.Kind) != f.Type {
			continue
		}
		if f.Repo != "" && it.Repo != f.Repo {
			continue
		}
		if f.Status != "" && hasStatus {
			v, ok := it.Value(statusField.Name)
			if !ok || v.OptionName != f.Status {
				continue
			}
		}
		if f.Assignee != "" {
			want := f.Assignee
			if want == "@me" {
				want = f.Me
			}
			if !hasAssignee(it, want) {
				continue
			}
		}
		if f.Search != "" && !MatchesQuery(it, f.Search) {
			continue
		}
		out = append(out, it)
	}
	return out
}

func hasAssignee(it githubx.Item, login string) bool {
	if login == "" {
		return false
	}
	for _, v := range it.Values {
		for _, u := range v.Users {
			if strings.EqualFold(u, login) {
				return true
			}
		}
	}
	return false
}

// SortByGrouping orders items by grouping-field option order, then by
// board order within each group (tui.md list/kanban agreement).
func SortByGrouping(b *githubx.Board, items []githubx.Item, groupField githubx.Field) []githubx.Item {
	rank := make(map[string]int, len(groupField.Options))
	for i, o := range groupField.Options {
		rank[o.Name] = i
	}
	noValue := len(groupField.Options) + 1
	type scored struct {
		it  githubx.Item
		key int
		idx int
	}
	ss := make([]scored, 0, len(items))
	for i, it := range items {
		key := noValue
		if v, ok := it.Value(groupField.Name); ok {
			if r, ok2 := rank[v.OptionName]; ok2 {
				key = r
			}
		}
		// No-value group goes first (tui.md: first column, hidden when empty).
		if key == noValue {
			key = -1
		}
		ss = append(ss, scored{it, key, i})
	}
	// Stable insertion sort keeps board order within groups.
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && (ss[j].key < ss[j-1].key); j-- {
			ss[j], ss[j-1] = ss[j-1], ss[j]
		}
	}
	out := make([]githubx.Item, len(ss))
	for i, s := range ss {
		out[i] = s.it
	}
	return out
}
