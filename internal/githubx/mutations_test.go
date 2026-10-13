package githubx

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

// fakeDoer records GraphQL calls and optionally fills responses.
type fakeDoer struct {
	queries []string
	vars    []map[string]interface{}
	fill    func(query string, resp interface{})
}

func (f *fakeDoer) DoWithContext(_ context.Context, query string, variables map[string]interface{}, response interface{}) error {
	f.queries = append(f.queries, query)
	f.vars = append(f.vars, variables)
	if f.fill != nil {
		f.fill(query, response)
	}
	return nil
}

func draftItem() Item {
	return Item{
		ID:         "PVTI_item",
		DatabaseID: 157092128,
		Kind:       KindDraft,
		Title:      "Server maintenance",
	}
}

// Regression: updateProjectV2DraftIssue needs the DraftIssue node ID
// (PVTDI_…), not the project item ID (PVTI_…). Passing the item ID
// made GitHub refuse every draft title/body edit with "Could not
// resolve to DraftIssue node".
func TestUpdateTitleBodyDraftUsesContentID(t *testing.T) {
	t.Setenv("GH_TOKEN", "test-token")
	f := &fakeDoer{}
	c := &gqlClient{doer: f}
	it := draftItem()
	it.ContentID = "PVTDI_draft"
	body := "Revisar Linode por ultima vez"
	if _, err := c.UpdateTitleBody(context.Background(), it, nil, &body); err != nil {
		t.Fatal(err)
	}
	if len(f.queries) != 1 {
		t.Fatalf("expected 1 GraphQL call, got %d: %v", len(f.queries), f.queries)
	}
	if got := f.vars[0]["id"]; got != "PVTDI_draft" {
		t.Fatalf("draftIssueId = %v, want PVTDI_draft (content ID, not item ID)", got)
	}
	if !strings.Contains(f.queries[0], "updateProjectV2DraftIssue") {
		t.Fatalf("unexpected mutation: %s", f.queries[0])
	}
}

// Without a content ID (older snapshots), the draft ID is resolved
// from the item first — still never the item ID itself.
func TestUpdateTitleBodyDraftResolvesContentID(t *testing.T) {
	t.Setenv("GH_TOKEN", "test-token")
	f := &fakeDoer{fill: func(query string, resp interface{}) {
		if !strings.Contains(query, "ProjectV2Item") {
			return
		}
		// Fill node.content.id on the anonymous response struct.
		rv := reflect.ValueOf(resp).Elem()
		node := rv.FieldByName("Node")
		nv := reflect.New(node.Type().Elem())
		nv.Elem().FieldByName("Content").FieldByName("ID").SetString("PVTDI_resolved")
		node.Set(nv)
	}}
	c := &gqlClient{doer: f}
	body := "x"
	if _, err := c.UpdateTitleBody(context.Background(), draftItem(), nil, &body); err != nil {
		t.Fatal(err)
	}
	if len(f.queries) != 2 {
		t.Fatalf("expected resolve + mutation, got %d calls", len(f.queries))
	}
	if got := f.vars[1]["id"]; got != "PVTDI_resolved" {
		t.Fatalf("draftIssueId = %v, want PVTDI_resolved", got)
	}
}
