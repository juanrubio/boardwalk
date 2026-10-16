package core

import (
	"testing"

	"github.com/juanrubio/boardwalk/internal/githubx"
)

func boardFixture() *githubx.Board {
	status := githubx.Field{ID: "F1", Name: "Status", DataType: "SINGLE_SELECT", Options: []githubx.Option{
		{ID: "o1", Name: "Todo"}, {ID: "o2", Name: "In Progress"}, {ID: "o3", Name: "Done"},
	}}
	mk := func(id, title, statusName, kind string, repo string, users ...string) githubx.Item {
		it := githubx.Item{ID: id, Title: title, Kind: githubx.Kind(kind), Repo: repo}
		if statusName != "" {
			it.Values = append(it.Values, githubx.FieldValue{FieldID: "F1", FieldName: "Status", DataType: "SINGLE_SELECT", OptionName: statusName})
		}
		if len(users) > 0 {
			it.Values = append(it.Values, githubx.FieldValue{FieldName: "Assignees", DataType: "ASSIGNEES", Users: users})
		}
		return it
	}
	return &githubx.Board{
		Fields: []githubx.Field{status},
		Items: []githubx.Item{
			mk("1", "Alpha task", "Done", "issue", "o/r", "alice"),
			mk("2", "Beta task", "Todo", "draft", ""),
			mk("3", "Gamma PR", "In Progress", "pr", "o/r", "bob"),
			mk("4", "No status", "", "issue", "o/x"),
		},
	}
}

func titles(items []githubx.Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Title
	}
	return out
}

func TestApplyFilters(t *testing.T) {
	b := boardFixture()
	tests := []struct {
		name string
		f    Filters
		want []string
	}{
		{name: "none", f: Filters{}, want: []string{"Alpha task", "Beta task", "Gamma PR", "No status"}},
		{name: "status", f: Filters{Status: "Todo"}, want: []string{"Beta task"}},
		{name: "type", f: Filters{Type: "pr"}, want: []string{"Gamma PR"}},
		{name: "repo", f: Filters{Repo: "o/x"}, want: []string{"No status"}},
		{name: "assignee me", f: Filters{Assignee: "@me", Me: "alice"}, want: []string{"Alpha task"}},
		{name: "search", f: Filters{Search: "gamma"}, want: []string{"Gamma PR"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := titles(ApplyFilters(b, tt.f))
			if len(got) != len(tt.want) {
				t.Fatalf("got %v want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSortByGrouping(t *testing.T) {
	b := boardFixture()
	field, _ := githubx.DefaultBoardField(b.Fields)
	got := titles(SortByGrouping(b, b.Items, field))
	want := []string{"No status", "Beta task", "Gamma PR", "Alpha task"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}
