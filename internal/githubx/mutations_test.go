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

// (A7) Conversion resolves the repository node ID, then sends the
// ITEM id (not the draft issue id) plus that repository ID, and
// returns the converted issue from the follow-up detail fetch.
func TestConvertDraftToIssue(t *testing.T) {
	t.Setenv("GH_TOKEN", "test-token")
	f := &fakeDoer{fill: func(query string, resp interface{}) {
		rv := reflect.ValueOf(resp).Elem()
		switch {
		case strings.Contains(query, "repository(owner"):
			repo := rv.FieldByName("Repository")
			nv := reflect.New(repo.Type().Elem())
			nv.Elem().FieldByName("ID").SetString("R_repo")
			repo.Set(nv)
		case strings.Contains(query, "BoardwalkItem"):
			var g gqlItem
			g.ID = "PVTI_1"
			g.Content.Typename = "Issue"
			g.Content.ID = "I_new"
			g.Content.Title = "Converted"
			rv.FieldByName("Node").Set(reflect.ValueOf(&g))
		}
	}}
	c := &gqlClient{doer: f}
	it, err := c.ConvertDraftToIssue(context.Background(), "PVT_p", "PVTI_1", "acme/website")
	if err != nil {
		t.Fatal(err)
	}
	if it.Kind != KindIssue || it.ContentID != "I_new" {
		t.Fatalf("converted item = %+v", it)
	}
	if len(f.queries) != 3 {
		t.Fatalf("expected repo resolve + mutation + detail, got %d", len(f.queries))
	}
	if f.vars[1]["item"] != "PVTI_1" || f.vars[1]["repo"] != "R_repo" {
		t.Fatalf("mutation vars = %v", f.vars[1])
	}
}

// (A13) Reorder sends updateProjectV2ItemPosition with project/item
// vars; after is JSON null (Go nil) for "move to top" and the ID
// string otherwise.
func TestUpdateItemPosition(t *testing.T) {
	t.Setenv("GH_TOKEN", "test-token")
	f := &fakeDoer{}
	c := &gqlClient{doer: f}
	ctx := context.Background()
	if err := c.UpdateItemPosition(ctx, "PVT_p", "PVTI_1", ""); err != nil {
		t.Fatal(err)
	}
	if err := c.UpdateItemPosition(ctx, "PVT_p", "PVTI_1", "PVTI_9"); err != nil {
		t.Fatal(err)
	}
	if len(f.queries) != 2 {
		t.Fatalf("expected 2 GraphQL calls, got %d: %v", len(f.queries), f.queries)
	}
	for i, q := range f.queries {
		if !strings.Contains(q, "updateProjectV2ItemPosition") {
			t.Fatalf("query %d: unexpected mutation: %s", i, q)
		}
		if f.vars[i]["project"] != "PVT_p" || f.vars[i]["item"] != "PVTI_1" {
			t.Fatalf("query %d: vars = %v, want project=PVT_p item=PVTI_1", i, f.vars[i])
		}
	}
	if v, present := f.vars[0]["after"]; !present || v != nil {
		t.Fatalf("move-to-top after = %v (present=%v), want present nil (JSON null)", v, present)
	}
	if got := f.vars[1]["after"]; got != "PVTI_9" {
		t.Fatalf("after = %v, want PVTI_9", got)
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
