package githubx

import (
	"context"

	"github.com/juanrubio/gh-boardwalk/internal/exiterr"
)

// Mutation implementations land in build step 4 (one file, one
// request per save). These stubs keep the Client contract complete.

func notYet(what string) error {
	return exiterr.Failuref("GitHub %s is not implemented yet in this build step", what)
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
