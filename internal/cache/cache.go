// Package cache implements ADR 0011: a disposable, private, local
// cache with a per-project schema/ID cache and a TUI board snapshot.
// It never stores credentials. Deleting the directory is always safe.
package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/juanrubio/gh-boardwalk/internal/githubx"
)

const snapshotVersion = 1

// key turns a project reference (owner/number) into a file-safe key.
func key(projectRef string) string {
	r := strings.NewReplacer("/", "-", ":", "-", " ", "-", "#", "-")
	return r.Replace(projectRef)
}

// Cache is a rooted cache directory.
type Cache struct {
	dir     string
	enabled bool
}

// Open resolves the OS cache directory (honouring --no-cache via
// disabled). A nil, disabled cache is valid: every read misses.
func Open(disabled bool) (*Cache, error) {
	if disabled || os.Getenv("BOARDWALK_NO_CACHE") == "1" {
		return &Cache{enabled: false}, nil
	}
	base, err := os.UserCacheDir()
	if err != nil {
		// Caching is an optimisation; failing to locate it disables it.
		return &Cache{enabled: false}, nil //nolint:nilerr // deliberate degrade
	}
	dir := filepath.Join(base, "boardwalk")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return &Cache{enabled: false}, nil //nolint:nilerr // deliberate degrade
	}
	return &Cache{dir: dir, enabled: true}, nil
}

// Dir returns the cache directory ("" when disabled).
func (c *Cache) Dir() string { return c.dir }

// SchemaEntry is the cached per-project schema and ID map.
type SchemaEntry struct {
	Project     githubx.Project
	Fields      []githubx.Field
	DatabaseIDs map[int64]string // item databaseId -> node ID
	SavedAt     time.Time
}

// SaveSchema stores (or refreshes) a project's schema entry.
func (c *Cache) SaveSchema(e SchemaEntry) {
	if !c.enabled {
		return
	}
	e.SavedAt = time.Now()
	_ = c.writeJSON("schema-"+key(e.Project.Ref())+".json", e)
}

// LoadSchema returns the cached schema, or ok=false on any problem.
// Corrupt or stale entries are misses, never errors (ADR 0011).
func (c *Cache) LoadSchema(projectRef string) (SchemaEntry, bool) {
	var e SchemaEntry
	if !c.enabled {
		return e, false
	}
	if err := c.readJSON("schema-"+key(projectRef)+".json", &e); err != nil {
		return SchemaEntry{}, false
	}
	return e, true
}

// Snapshot is the last loaded board for a project (displayed fields
// only, never bodies).
type Snapshot struct {
	Version int
	Project githubx.Project
	Fields  []githubx.Field
	Items   []githubx.Item
	SavedAt time.Time
}

// SaveSnapshot stores a board snapshot after a successful live load.
func (c *Cache) SaveSnapshot(s Snapshot) {
	if !c.enabled {
		return
	}
	s.Version = snapshotVersion
	s.SavedAt = time.Now()
	_ = c.writeJSON("board-"+key(s.Project.Ref())+".json", s)
}

// LoadSnapshot returns the snapshot, or ok=false when absent, corrupt,
// or from an incompatible version.
func (c *Cache) LoadSnapshot(projectRef string) (Snapshot, bool) {
	var s Snapshot
	if !c.enabled {
		return s, false
	}
	if err := c.readJSON("board-"+key(projectRef)+".json", &s); err != nil {
		return Snapshot{}, false
	}
	if s.Version != snapshotVersion {
		return Snapshot{}, false
	}
	return s, true
}

// Prune deletes snapshots unused for more than 30 days (ADR 0011).
func (c *Cache) Prune() {
	if !c.enabled {
		return
	}
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "board-") {
			continue
		}
		info, err := e.Info()
		if err == nil && info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(c.dir, e.Name()))
		}
	}
}

func (c *Cache) writeJSON(name string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := filepath.Join(c.dir, name+".tmp")
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(c.dir, name))
}

func (c *Cache) readJSON(name string, v any) error {
	data, err := os.ReadFile(filepath.Join(c.dir, name)) //nolint:gosec // name is internally generated
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return err
	}
	return nil
}
