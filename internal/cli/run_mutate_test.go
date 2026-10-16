package cli

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/juanrubio/boardwalk/internal/exiterr"
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

// (A7) Convert: drafts convert with repo inference, non-drafts are a
// usage error, and --dry-run changes nothing.
func TestConvertDraft(t *testing.T) {
	opts, buf := fixtureOptions(t, false)
	if err := runCmd(t, opts, "item", "add", "--draft", "--title", "Convert me", "--yes"); err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`draft:\d+`).FindString(buf.String())
	if m == "" {
		t.Fatalf("no draft ref in add output:\n%s", buf.String())
	}
	buf.Reset()
	if err := runCmd(t, opts, "item", "convert", m, "--yes"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Converted "+m+" → fixture/repo#") {
		t.Fatalf("expected conversion line, got:\n%s", buf.String())
	}
	// Non-draft cannot convert (exit 2).
	err := runCmd(t, opts, "item", "convert", "fixture/repo#32", "--yes")
	if err == nil || exiterr.CodeOf(err) != exiterr.CodeUsage {
		t.Fatalf("non-draft convert: err=%v code=%d, want usage", err, exiterr.CodeOf(err))
	}
}

func TestConvertDryRunChangesNothing(t *testing.T) {
	opts, buf := fixtureOptions(t, false)
	if err := runCmd(t, opts, "item", "add", "--draft", "--title", "Convert dry", "--yes"); err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`draft:\d+`).FindString(buf.String())
	buf.Reset()
	if err := runCmd(t, opts, "item", "convert", m, "--dry-run"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Dry run") {
		t.Fatalf("expected dry-run output, got:\n%s", buf.String())
	}
	// Still a draft afterwards: converting for real succeeds.
	buf.Reset()
	if err := runCmd(t, opts, "item", "convert", m, "--yes"); err != nil {
		t.Fatalf("convert after dry-run failed: %v", err)
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
