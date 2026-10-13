package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/juanrubio/gh-boardwalk/internal/config"
	"github.com/juanrubio/gh-boardwalk/internal/core"
	"github.com/juanrubio/gh-boardwalk/internal/fixture"
)

func fixtureOptions(t *testing.T, jsonOut bool) (*Options, *bytes.Buffer) {
	t.Helper()
	fc, err := fixture.LoadDir(filepath.Join("..", "..", "testdata", "fixtures"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Default: "fixture-user/1", Aliases: map[string]string{}}
	buf := &bytes.Buffer{}
	opts := &Options{Out: buf, Err: &bytes.Buffer{}, JSON: jsonOut}
	opts.NewService = func(_ *Options) (*core.Service, error) {
		return core.New(fc, cfg, nil), nil
	}
	return opts, buf
}

func TestItemListOfflineJSON(t *testing.T) {
	opts, buf := fixtureOptions(t, true)
	root := NewRootCmd(opts)
	root.SetArgs([]string{"item", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var items []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, buf.String())
	}
	if len(items) != 14 {
		t.Fatalf("items = %d, want 14", len(items))
	}
	if items[0]["ref"] == "" || items[0]["kind"] == "" {
		t.Fatalf("missing ref/kind in %+v", items[0])
	}
}

func TestItemListOfflineHumanAndFilter(t *testing.T) {
	opts, buf := fixtureOptions(t, false)
	root := NewRootCmd(opts)
	root.SetArgs([]string{"item", "list", "--type", "draft"})
	// SnazzyEmail fixture is all issues; expect a clean empty result.
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Showing 0 of 14") {
		t.Fatalf("unexpected output:\n%s", buf.String())
	}
}

func TestProjectListOffline(t *testing.T) {
	opts, buf := fixtureOptions(t, true)
	root := NewRootCmd(opts)
	root.SetArgs([]string{"project", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var projects []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &projects); err != nil {
		t.Fatal(err)
	}
	if len(projects) != 3 {
		t.Fatalf("projects = %d, want 3", len(projects))
	}
}
