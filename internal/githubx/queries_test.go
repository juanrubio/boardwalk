package githubx

import "testing"

// Regression: detail fetches decode with no board schema (fields=nil).
// Field values must survive via the name/data type carried on each
// value node; dropping them made items lose their Status grouping
// after the detail pane opened.
func TestToItemWithoutSchemaKeepsFieldValues(t *testing.T) {
	var g gqlItem
	g.ID = "PVTI_1"
	g.Content.Typename = "Issue"
	g.Content.ID = "I_1"
	g.Content.Title = "Thing"
	g.FieldValues.Nodes = []gqlFieldValue{
		{Typename: "ProjectV2ItemFieldSingleSelectValue", Name: "Ready", OptionID: "o1"},
		{Typename: "ProjectV2ItemFieldTextValue", Text: "hello"},
	}
	g.FieldValues.Nodes[0].Field.ID = "f-status"
	g.FieldValues.Nodes[0].Field.Name = "Status"
	g.FieldValues.Nodes[0].Field.DataType = "SINGLE_SELECT"
	g.FieldValues.Nodes[1].Field.ID = "f-notes"
	g.FieldValues.Nodes[1].Field.Name = "Notes"
	g.FieldValues.Nodes[1].Field.DataType = "TEXT"

	it := g.toItem(nil)
	if it.ContentID != "I_1" {
		t.Fatalf("ContentID = %q", it.ContentID)
	}
	v, ok := it.Value("Status")
	if !ok || v.OptionName != "Ready" {
		t.Fatalf("Status value lost without schema: %+v ok=%v", v, ok)
	}
	if v, ok := it.Value("Notes"); !ok || v.Text != "hello" {
		t.Fatalf("Notes value lost without schema: %+v ok=%v", v, ok)
	}
}
