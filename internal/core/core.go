// Package core is Boardwalk's shared service layer. CLI commands and
// the TUI both call into it; neither contains GitHub logic of its own
// (ADR 0002, ADR 0006, ADR 0007).
package core

import (
	"context"
	"strings"

	"github.com/juanrubio/boardwalk/internal/cache"
	"github.com/juanrubio/boardwalk/internal/config"
	"github.com/juanrubio/boardwalk/internal/exiterr"
	"github.com/juanrubio/boardwalk/internal/githubx"
)

// Service wires the GitHub client, config, and cache together.
type Service struct {
	Client githubx.Client
	Config *config.Config
	Cache  *cache.Cache
}

// New builds a Service. client must be non-nil for GitHub-backed calls.
func New(client githubx.Client, cfg *config.Config, c *cache.Cache) *Service {
	return &Service{Client: client, Config: cfg, Cache: c}
}

// CheckAuth delegates the upfront auth/scope gate (ADR 0010).
func (s *Service) CheckAuth(ctx context.Context) error {
	if s.Client == nil {
		return exiterr.Authf("no GitHub client configured")
	}
	return s.Client.CheckAuth(ctx)
}

// ResolveProjectRef returns the concrete project reference for a
// command (ADR 0005 order: flag > BOARDWALK_PROJECT > default).
func (s *Service) ResolveProjectRef(flag string) string {
	if s.Config == nil {
		return flag
	}
	return s.Config.ResolveProject(flag)
}

// LoadBoard loads a full board progressively: every page is delivered
// through onPage (may be nil) and the merged board is returned. Query
// is an optional GitHub project filter expression (B9); items are
// filtered locally as a fallback so results match either path.
func (s *Service) LoadBoard(ctx context.Context, project githubx.Project, query string, onPage func(items []githubx.Item)) (*githubx.Board, error) {
	if s.Client == nil {
		return nil, exiterr.Authf("no GitHub client configured")
	}
	board := &githubx.Board{Project: project}
	cursor := ""
	for page := 0; ; page++ {
		size := 100
		if page == 0 {
			size = 50 // first page returns sooner (performance.md)
		}
		fields, items, next, err := s.Client.LoadBoardPage(ctx, project.ID, cursor, size, query)
		if err != nil {
			return nil, err
		}
		if page == 0 {
			board.Fields = fields
			if s.Cache != nil {
				s.Cache.SaveSchema(cache.SchemaEntry{Project: project, Fields: fields, DatabaseIDs: databaseIDMap(items)})
			}
		}
		board.Items = append(board.Items, items...)
		if onPage != nil {
			onPage(items)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	if s.Cache != nil {
		s.Cache.SaveSnapshot(cache.Snapshot{Project: project, Fields: board.Fields, Items: board.Items})
		s.Cache.SaveSchema(cache.SchemaEntry{Project: project, Fields: board.Fields, DatabaseIDs: databaseIDMap(board.Items)})
	}
	return board, nil
}

func databaseIDMap(items []githubx.Item) map[int64]string {
	m := make(map[int64]string, len(items))
	for _, it := range items {
		if it.DatabaseID != 0 {
			m[it.DatabaseID] = it.ID
		}
	}
	return m
}

// MatchesQuery reports whether an item matches free-text search
// (title, reference, repository). Structured GitHub filter syntax is
// evaluated by the server; this is the local fallback for plain text.
func MatchesQuery(it githubx.Item, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	hay := strings.ToLower(it.Title + " " + it.Ref() + " " + it.Repo)
	return strings.Contains(hay, q)
}
