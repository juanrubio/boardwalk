package githubx

import (
	"context"

	"github.com/juanrubio/gh-boardwalk/internal/exiterr"
)

// gqlClient is the go-gh-backed GraphQL client (ADR 0002). The full
// implementation lands with the core read path; this scaffold keeps
// the binary honest about what exists so far.
type gqlClient struct{}

// NewClient returns the GitHub-backed Client.
func NewClient() Client { return &gqlClient{} }

func notYet(what string) error {
	return exiterr.Failuref("GitHub %s is not implemented yet in this build step", what)
}

func (c *gqlClient) CheckAuth(_ context.Context) error { return notYet("auth check") }

func (c *gqlClient) ListProjects(_ context.Context, _ string) ([]Project, error) {
	return nil, notYet("project list")
}

func (c *gqlClient) ResolveProject(_ context.Context, _ string) (Project, error) {
	return Project{}, notYet("project resolve")
}

func (c *gqlClient) LoadBoardPage(_ context.Context, _ string, _ string, _ int, _ string) ([]Field, []Item, string, error) {
	return nil, nil, "", notYet("board load")
}

func (c *gqlClient) LoadItemDetail(_ context.Context, _ string) (Item, error) {
	return Item{}, notYet("item detail")
}

func (c *gqlClient) AddItem(_ context.Context, _, _ string) (Item, error) {
	return Item{}, notYet("item add")
}

func (c *gqlClient) CreateDraft(_ context.Context, _, _, _ string) (Item, error) {
	return Item{}, notYet("draft create")
}

func (c *gqlClient) ResolveContent(_ context.Context, _ string) (string, Item, error) {
	return "", Item{}, notYet("content resolve")
}

func (c *gqlClient) SetFieldValue(_ context.Context, _, _, _ string, _ MutationValue) (Item, error) {
	return Item{}, notYet("field update")
}

func (c *gqlClient) UpdateTitleBody(_ context.Context, _ Item, _, _ *string) (Item, error) {
	return Item{}, notYet("title/body update")
}

func (c *gqlClient) ArchiveItem(_ context.Context, _, _ string) error {
	return notYet("item archive")
}
