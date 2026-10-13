package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveOrderFlagEnvDefault(t *testing.T) {
	cfg := &Config{Default: "life-board", Aliases: map[string]string{
		"life-board": "juanrubio/3", "snazzy": "juanrubio/1",
	}}
	t.Setenv("BOARDWALK_PROJECT", "snazzy")
	if got := cfg.ResolveProject("juanrubio/1"); got != "juanrubio/1" {
		t.Fatalf("flag should win, got %q", got)
	}
	if got := cfg.ResolveProject(""); got != "juanrubio/1" {
		t.Fatalf("env should beat default, got %q", got)
	}
	if err := os.Unsetenv("BOARDWALK_PROJECT"); err != nil {
		t.Fatal(err)
	}
	if got := cfg.ResolveProject(""); got != "juanrubio/3" {
		t.Fatalf("default alias should expand, got %q", got)
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.path = path
	cfg.SetAlias("life-board", "juanrubio/3")
	cfg.Default = "life-board"
	cfg.SetUIPrefs("juanrubio/3", UIPrefs{GroupBy: "Status", Density: "compact", Collapsed: []string{"Done"}})
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	back, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if back.Aliases["life-board"] != "juanrubio/3" || back.Default != "life-board" {
		t.Fatalf("round trip mismatch: %+v", back)
	}
	p, ok := back.UIPrefsFor("juanrubio/3")
	if !ok || p.GroupBy != "Status" || p.Density != "compact" {
		t.Fatalf("ui prefs mismatch: %+v ok=%v", p, ok)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config perms = %o, want 600", info.Mode().Perm())
	}
}

func TestValidAlias(t *testing.T) {
	for _, ok := range []string{"life-board", "a", "x1-y2"} {
		if !ValidAlias(ok) {
			t.Errorf("ValidAlias(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"", "Life", "-x", "x-", "x_y", "x y"} {
		if ValidAlias(bad) {
			t.Errorf("ValidAlias(%q) = true, want false", bad)
		}
	}
}
