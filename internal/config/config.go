// Package config loads and saves Boardwalk's user configuration
// (ADR 0005 user-only scope, ADR 0008 TOML format) and resolves which
// project a command targets.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/juanrubio/boardwalk/internal/exiterr"
)

// UIPrefs holds per-project TUI view preferences (tui.md Section 1/3).
type UIPrefs struct {
	GroupBy   string   `toml:"group_by"`
	Density   string   `toml:"density"`
	Collapsed []string `toml:"collapsed"`
}

// UI holds the [ui] section of the config file.
type UI struct {
	Projects map[string]UIPrefs `toml:"projects"`
}

// Config is the parsed config.toml. It never contains credentials
// (ADR 0002/0005): only aliases, a default project, and UI preferences.
type Config struct {
	Default string            `toml:"default"`
	Aliases map[string]string `toml:"aliases"`
	UI      UI                `toml:"ui"`

	path string // file this config was loaded from ("" = defaults)
}

// Path returns the config file location for the current OS
// (~/.config/boardwalk/config.toml on Linux; ADR 0008).
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", exiterr.Wrap(exiterr.CodeFailure, "locate user config directory", err)
	}
	return filepath.Join(dir, "boardwalk", "config.toml"), nil
}

// Load reads the config file. A missing file is normal and yields an
// empty Config. A malformed file is an actionable error naming the
// file and line (ADR 0008).
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

// LoadFrom is Load with an explicit path (used in tests).
func LoadFrom(path string) (*Config, error) {
	cfg := &Config{Aliases: map[string]string{}, path: path}
	cfg.UI.Projects = map[string]UIPrefs{}
	data, err := os.ReadFile(path) //nolint:gosec // path is the user's own config file
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, exiterr.Wrap(exiterr.CodeFailure, "read config "+path, err)
	}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, exiterr.New(exiterr.CodeUsage,
			fmt.Sprintf("malformed config file %s: %v", path, err)).
			WithRemediation("Fix the TOML in that file, or move it aside and run `boardwalk project use` to recreate it.")
	}
	if cfg.Aliases == nil {
		cfg.Aliases = map[string]string{}
	}
	if cfg.UI.Projects == nil {
		cfg.UI.Projects = map[string]UIPrefs{}
	}
	for key := range cfg.Aliases {
		if !ValidAlias(key) {
			fmt.Fprintf(os.Stderr, "boardwalk: warning: ignoring alias %q in %s: names must be lowercase letters, digits, and hyphens\n", key, path)
			delete(cfg.Aliases, key)
		}
	}
	return cfg, nil
}

// FilePath returns where this config is persisted.
func (c *Config) FilePath() string { return c.path }

// Save writes the config back, creating the directory (0700) and file
// (0600). Comments in the original file are not preserved (ADR 0008
// accepts this); the content is regenerated from the struct.
func (c *Config) Save() error {
	if c.path == "" {
		path, err := Path()
		if err != nil {
			return err
		}
		c.path = path
	}
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return exiterr.Wrap(exiterr.CodeFailure, "create config directory "+dir, err)
	}
	data, err := toml.Marshal(c)
	if err != nil {
		return exiterr.Wrap(exiterr.CodeFailure, "encode config", err)
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return exiterr.Wrap(exiterr.CodeFailure, "write config "+c.path, err)
	}
	if err := os.Rename(tmp, c.path); err != nil {
		return exiterr.Wrap(exiterr.CodeFailure, "replace config "+c.path, err)
	}
	return nil
}

// SetAlias records name -> projectRef (owner/number or URL).
func (c *Config) SetAlias(name, projectRef string) {
	if c.Aliases == nil {
		c.Aliases = map[string]string{}
	}
	c.Aliases[name] = projectRef
}

// SetUIPrefs stores per-project TUI preferences.
func (c *Config) SetUIPrefs(projectRef string, p UIPrefs) {
	if c.UI.Projects == nil {
		c.UI.Projects = map[string]UIPrefs{}
	}
	c.UI.Projects[projectRef] = p
}

// UIPrefsFor returns the stored TUI preferences for a project, if any.
func (c *Config) UIPrefsFor(projectRef string) (UIPrefs, bool) {
	p, ok := c.UI.Projects[projectRef]
	return p, ok
}

// ValidAlias reports whether name matches ADR 0008: lowercase letters,
// digits, and hyphens, not starting or ending with a hyphen.
func ValidAlias(name string) bool {
	if name == "" || strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	return true
}

// ResolveProject applies the ADR 0005 resolution order and returns the
// concrete project reference (alias, URL, or owner/number):
//
//  1. explicit flag value
//  2. BOARDWALK_PROJECT environment variable
//  3. saved default (itself resolved through aliases)
//
// It returns "" with a nil error when nothing resolves, so callers can
// fall back to interactive selection or an actionable error.
func (c *Config) ResolveProject(flag string) string {
	if flag != "" {
		return c.expand(flag)
	}
	if env := os.Getenv("BOARDWALK_PROJECT"); env != "" {
		return c.expand(env)
	}
	if c.Default != "" {
		return c.expand(c.Default)
	}
	return ""
}

func (c *Config) expand(ref string) string {
	if target, ok := c.Aliases[ref]; ok {
		return target
	}
	return ref
}
