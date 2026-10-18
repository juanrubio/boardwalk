package githubx

import (
	"reflect"
	"testing"
)

func reorderItem(id, status string) Item {
	return Item{ID: id, Values: []FieldValue{{FieldName: "Status", OptionName: status}}}
}

// Global order interleaves two columns: Todo = a1,a2,a3; Doing = b1,b2.
func reorderItems() []Item {
	return []Item{
		reorderItem("a1", "Todo"),
		reorderItem("b1", "Doing"),
		reorderItem("a2", "Todo"),
		reorderItem("b2", "Doing"),
		reorderItem("a3", "Todo"),
	}
}

func itemIDs(items []Item) []string {
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	return ids
}

func TestPositionAfterID(t *testing.T) {
	items := reorderItems()
	tests := []struct {
		name      string
		groupName string
		itemID    string
		dir       int
		wantAfter string
		wantOK    bool
	}{
		{name: "down mid-column", groupName: "Status", itemID: "a2", dir: 1, wantAfter: "a3", wantOK: true},
		{name: "down mid-column other column", groupName: "Status", itemID: "b1", dir: 1, wantAfter: "b2", wantOK: true},
		// a3's column neighbour above is a2; globally a2 follows b1, so
		// a3 must sit after b1 to land immediately before a2.
		{name: "up mid-column interleaved", groupName: "Status", itemID: "a3", dir: -1, wantAfter: "b1", wantOK: true},
		{name: "up mid-column interleaved other column", groupName: "Status", itemID: "b2", dir: -1, wantAfter: "a1", wantOK: true},
		// a2's neighbour above is a1, first in the global order.
		{name: "up to top", groupName: "Status", itemID: "a2", dir: -1, wantAfter: "", wantOK: true},
		{name: "already top of column", groupName: "Status", itemID: "a1", dir: -1, wantOK: false},
		{name: "already top of later column", groupName: "Status", itemID: "b1", dir: -1, wantOK: false},
		{name: "already bottom of column", groupName: "Status", itemID: "a3", dir: 1, wantOK: false},
		{name: "already bottom of earlier column", groupName: "Status", itemID: "b2", dir: 1, wantOK: false},
		{name: "unknown item", groupName: "Status", itemID: "nope", dir: 1, wantOK: false},
		{name: "whole list down", groupName: "", itemID: "b1", dir: 1, wantAfter: "a2", wantOK: true},
		{name: "whole list up", groupName: "", itemID: "a2", dir: -1, wantAfter: "a1", wantOK: true},
		{name: "whole list up to top", groupName: "", itemID: "b1", dir: -1, wantAfter: "", wantOK: true},
		{name: "whole list already top", groupName: "", itemID: "a1", dir: -1, wantOK: false},
		{name: "whole list already bottom", groupName: "", itemID: "a3", dir: 1, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			after, ok := PositionAfterID(items, tt.groupName, tt.itemID, tt.dir)
			if ok != tt.wantOK || after != tt.wantAfter {
				t.Fatalf("PositionAfterID(%q, %q, %d) = (%q, %v), want (%q, %v)",
					tt.groupName, tt.itemID, tt.dir, after, ok, tt.wantAfter, tt.wantOK)
			}
		})
	}
}

func TestApplyReorder(t *testing.T) {
	tests := []struct {
		name    string
		itemID  string
		afterID string
		want    []string
	}{
		{name: "insert after", itemID: "a1", afterID: "a2", want: []string{"b1", "a2", "a1", "b2", "a3"}},
		{name: "insert after last", itemID: "b1", afterID: "a3", want: []string{"a1", "a2", "b2", "a3", "b1"}},
		{name: "move up one", itemID: "a3", afterID: "b1", want: []string{"a1", "b1", "a3", "a2", "b2"}},
		{name: "to top", itemID: "a3", afterID: "", want: []string{"a3", "a1", "b1", "a2", "b2"}},
		{name: "unknown item unchanged", itemID: "nope", afterID: "a1", want: []string{"a1", "b1", "a2", "b2", "a3"}},
		{name: "unknown after unchanged", itemID: "a1", afterID: "nope", want: []string{"a1", "b1", "a2", "b2", "a3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := reorderItems()
			before := itemIDs(items)
			got := ApplyReorder(items, tt.itemID, tt.afterID)
			if ids := itemIDs(got); !reflect.DeepEqual(ids, tt.want) {
				t.Fatalf("ApplyReorder(%q, %q) = %v, want %v", tt.itemID, tt.afterID, ids, tt.want)
			}
			if ids := itemIDs(items); !reflect.DeepEqual(ids, before) {
				t.Fatalf("input slice mutated: %v, was %v", ids, before)
			}
		})
	}
}
