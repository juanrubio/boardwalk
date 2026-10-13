package fixture

import (
	"context"
	"path/filepath"
	"testing"
)

func load(t *testing.T) *Client {
	t.Helper()
	c, err := LoadDir(filepath.Join("..", "..", "testdata", "fixtures"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestFixtureBoardsLoad(t *testing.T) {
	c := load(t)
	if len(c.Boards) != 3 {
		t.Fatalf("boards = %d, want 3", len(c.Boards))
	}
	tests := []struct {
		ref   string
		items int
	}{
		{"fixture-user/1", 14},
		{"fixture-user/3", 16},
		{"fixture-org/7", 150},
	}
	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			b := c.Board(tt.ref)
			if b == nil {
				t.Fatalf("missing board %s", tt.ref)
			}
			if len(b.Items) != tt.items {
				t.Fatalf("items = %d, want %d", len(b.Items), tt.items)
			}
		})
	}
	large := c.Board("fixture-org/7")
	kinds := map[string]bool{}
	types := map[string]bool{}
	for _, it := range large.Items {
		kinds[string(it.Kind)] = true
	}
	for _, f := range large.Fields {
		types[f.DataType] = true
	}
	for _, want := range []string{"issue", "pr", "draft"} {
		if !kinds[want] {
			t.Errorf("large board missing kind %s", want)
		}
	}
	for _, want := range []string{"SINGLE_SELECT", "ITERATION", "TEXT"} {
		if !types[want] {
			t.Errorf("large board missing field type %s", want)
		}
	}
}

func TestFixturePagination(t *testing.T) {
	c := load(t)
	b := c.Board("fixture-org/7")
	ctx := context.Background()
	_, page1, next, err := c.LoadBoardPage(ctx, b.Project.ID, "", 50, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 50 || next == "" {
		t.Fatalf("page1 = %d next=%q", len(page1), next)
	}
	_, page3, next3, err := c.LoadBoardPage(ctx, b.Project.ID, "100", 50, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page3) != 50 || next3 != "" {
		t.Fatalf("page3 = %d next=%q", len(page3), next3)
	}
}
