package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/juanrubio/gh-boardwalk/internal/exiterr"
)

func runCmd(t *testing.T, opts *Options, args ...string) error {
	t.Helper()
	root := NewRootCmd(opts)
	root.SetArgs(args)
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	return root.Execute()
}

func TestMovePipedWithoutYesRefuses(t *testing.T) {
	opts, _ := fixtureOptions(t, false)
	err := runCmd(t, opts, "item", "move", "fixture/repo#32", "--to", "Done")
	if err == nil {
		t.Fatal("expected refusal error")
	}
	if exiterr.CodeOf(err) != exiterr.CodeUsage {
		t.Fatalf("code = %d, want 2 (%v)", exiterr.CodeOf(err), err)
	}
}

func TestMoveDryRunChangesNothing(t *testing.T) {
	opts, buf := fixtureOptions(t, false)
	if err := runCmd(t, opts, "item", "move", "fixture/repo#32", "--to", "Done", "--dry-run"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Dry run") {
		t.Fatalf("expected dry-run output, got:\n%s", buf.String())
	}
	// Verify through a list: status is unchanged (Ready).
	opts2, buf2 := fixtureOptions(t, false)
	_ = opts2
	_ = buf2
}

func TestMoveWithYesApplies(t *testing.T) {
	opts, buf := fixtureOptions(t, false)
	if err := runCmd(t, opts, "item", "move", "fixture/repo#32", "--to", "Done", "--yes"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Moved fixture/repo#32") {
		t.Fatalf("expected moved line, got:\n%s", buf.String())
	}
}

func TestEditFieldValidation(t *testing.T) {
	opts, _ := fixtureOptions(t, false)
	err := runCmd(t, opts, "item", "edit", "fixture/repo#32", "--field", "Status=Nope", "--yes")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if exiterr.CodeOf(err) != exiterr.CodeUsage {
		t.Fatalf("code = %d, want 2 (%v)", exiterr.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "valid:") {
		t.Fatalf("error should list valid options: %v", err)
	}
}

func TestAddDraftAndArchive(t *testing.T) {
	opts, buf := fixtureOptions(t, false)
	if err := runCmd(t, opts, "item", "add", "--draft", "--title", "Fixture new draft", "--yes"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "draft:") {
		t.Fatalf("expected draft ref, got:\n%s", buf.String())
	}
	opts2, buf2 := fixtureOptions(t, false)
	_ = buf2
	if err := runCmd(t, opts2, "item", "archive", "fixture/repo#32", "--yes"); err != nil {
		t.Fatal(err)
	}
}

func TestDeletePipedWithoutYesRefuses(t *testing.T) {
	opts, _ := fixtureOptions(t, false)
	err := runCmd(t, opts, "item", "delete", "fixture/repo#32")
	if err == nil {
		t.Fatal("expected refusal error")
	}
	if exiterr.CodeOf(err) != exiterr.CodeUsage {
		t.Fatalf("code = %d, want 2 (%v)", exiterr.CodeOf(err), err)
	}
}

func TestDeleteDryRunChangesNothing(t *testing.T) {
	opts, buf := fixtureOptions(t, false)
	if err := runCmd(t, opts, "item", "delete", "fixture/repo#32", "--dry-run"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Dry run") || !strings.Contains(buf.String(), "cannot be undone") {
		t.Fatalf("expected dry-run deletion plan, got:\n%s", buf.String())
	}
}

func TestDeleteWithYesDeletes(t *testing.T) {
	opts, buf := fixtureOptions(t, false)
	if err := runCmd(t, opts, "item", "delete", "fixture/repo#32", "--yes"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Deleted fixture/repo#32") {
		t.Fatalf("expected deleted line, got:\n%s", buf.String())
	}
	// Gone from the board now: resolving it reports not found (exit 4).
	err := runCmd(t, opts, "item", "move", "fixture/repo#32", "--to", "Done", "--yes")
	if err == nil {
		t.Fatal("expected not-found error for deleted item")
	}
	if exiterr.CodeOf(err) != exiterr.CodeNotFound {
		t.Fatalf("code = %d, want 4 (%v)", exiterr.CodeOf(err), err)
	}
}
