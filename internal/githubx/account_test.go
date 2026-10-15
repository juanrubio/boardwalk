package githubx

import (
	"context"
	"errors"
	"testing"

	"github.com/juanrubio/gh-boardwalk/internal/exiterr"
)

// (A11) Account clients resolve the named account's token via gh
// and refuse cleanly (exit 3) when that account has none.
func TestAccountClientCheckAuth(t *testing.T) {
	orig := accountToken
	defer func() { accountToken = orig }()

	accountToken = func(account string) (string, error) {
		if account == "juanrubio" {
			return "tok-secret-value", nil
		}
		return "", errors.New("no token")
	}
	c := NewClientForAccount("juanrubio")
	if err := c.CheckAuth(context.Background()); err != nil {
		t.Fatalf("known account should pass CheckAuth: %v", err)
	}
	// The GraphQL doer builds without network when a token exists.
	gc, ok := c.(*gqlClient)
	if !ok {
		t.Fatal("NewClientForAccount should return a gqlClient")
	}
	if _, err := gc.gql(); err != nil {
		t.Fatalf("gql() with account token: %v", err)
	}

	bad := NewClientForAccount("ghost")
	err := bad.CheckAuth(context.Background())
	if err == nil || exiterr.CodeOf(err) != exiterr.CodeAuth {
		t.Fatalf("unknown account: err=%v code=%d, want auth(3)", err, exiterr.CodeOf(err))
	}
}
