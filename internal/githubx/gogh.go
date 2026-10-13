package githubx

import (
	"context"
	"os/exec"
	"strings"
	"sync"

	ghapi "github.com/cli/go-gh/v2/pkg/api"
	ghauth "github.com/cli/go-gh/v2/pkg/auth"

	"github.com/juanrubio/gh-boardwalk/internal/exiterr"
)

// gqlDoer is the subset of go-gh's GraphQLClient we use, so decoding
// can be tested against recorded responses.
type gqlDoer interface {
	DoWithContext(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error
}

// gqlClient is the go-gh-backed GraphQL client (ADR 0002): credentials
// resolve from gh's own store and never pass through Boardwalk files.
type gqlClient struct {
	mu   sync.Mutex
	doer gqlDoer
}

// NewClient returns the GitHub-backed Client.
func NewClient() Client { return &gqlClient{} }

func (c *gqlClient) gql() (gqlDoer, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.doer != nil {
		return c.doer, nil
	}
	client, err := ghapi.NewGraphQLClient(ghapi.ClientOptions{})
	if err != nil {
		return nil, exiterr.Authf("GitHub CLI session unavailable: %v", err).
			WithRemediation("Install gh and run `gh auth login`, then `gh auth refresh -s project,repo`.")
	}
	c.doer = client
	return c.doer, nil
}

// CheckAuth implements ADR 0010's upfront gate without a dedicated
// auth round trip: gh must exist and hold a token locally; scope
// enforcement happens on the first real response's headers/errors.
func (c *gqlClient) CheckAuth(_ context.Context) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return exiterr.Authf("GitHub CLI (gh) was not found in PATH").
			WithRemediation("Install gh from https://cli.github.com, then run `gh auth login` and `gh auth refresh -s project,repo`.")
	}
	host, _ := ghauth.DefaultHost()
	token, _ := ghauth.TokenForHost(host)
	if token == "" {
		return exiterr.Authf("gh is not authenticated for %s", host).
			WithRemediation("Run `gh auth login`, then `gh auth refresh -s project,repo`.")
	}
	return nil
}

// do runs one GraphQL document and classifies failures (ADR 0009).
func (c *gqlClient) do(ctx context.Context, query string, vars map[string]interface{}, resp interface{}) error {
	gql, err := c.gql()
	if err != nil {
		return err
	}
	if err := gql.DoWithContext(ctx, query, vars, resp); err != nil {
		return classify(err)
	}
	return nil
}

// classify maps GraphQL transport errors to exit codes.
func classify(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "INSUFFICIENT_SCOPES"), strings.Contains(msg, "insufficient scopes"):
		return exiterr.Authf("GitHub token is missing project/repo scopes: %s", firstLine(msg)).
			WithRemediation("Run `gh auth refresh -s project,repo`, then retry.")
	case strings.Contains(msg, "RATE_LIMITED"), strings.Contains(msg, "rate limit"):
		return exiterr.Failuref("GitHub rate limit reached: %s", firstLine(msg)).
			WithRemediation("Wait for the rate-limit window to reset, then retry. Boardwalk never retries a mutation silently.")
	case strings.Contains(msg, "NOT_FOUND"), strings.Contains(msg, "Could not resolve"):
		return exiterr.NotFoundf("GitHub resource not found: %s", firstLine(msg))
	case strings.Contains(msg, "FORBIDDEN"), strings.Contains(msg, "permission"):
		return exiterr.Permissionf("GitHub permission denied: %s", firstLine(msg)).
			WithRemediation("Check that your account can write to this project, or run `gh auth refresh -s project,repo` if scopes are missing.")
	default:
		return exiterr.Wrap(exiterr.CodeFailure, "GitHub request failed: "+firstLine(msg), err)
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// ---- shared GraphQL fragments -------------------------------------------

const projectFieldsFragment = `
fragment ProjFields on ProjectV2 {
  id number title url closed shortDescription
  owner { __typename ... on User { login } ... on Organization { login } }
  items { totalCount }
}`

const fieldsFragment = `
fragment BoardFields on ProjectV2 {
  fields(first: 100) {
    nodes {
      ... on ProjectV2FieldCommon { id name dataType }
      ... on ProjectV2SingleSelectField { id name dataType options { id name color description } }
      ... on ProjectV2IterationField { id name dataType configuration { iterations { id title startDate duration } completedIterations { id title startDate duration } } }
    }
  }
}`

const itemFragment = `
fragment BoardItem on ProjectV2Item {
  id databaseId type isArchived
  content {
    __typename
    ... on Issue { id number title url body assignees(first: 10) { nodes { login } } repository { nameWithOwner } }
    ... on PullRequest { id number title url body assignees(first: 10) { nodes { login } } repository { nameWithOwner } }
    ... on DraftIssue { id title body assignees(first: 10) { nodes { login } } }
  }
  fieldValues(first: 50) {
    nodes {
      __typename
      ... on ProjectV2ItemFieldTextValue { text field { ... on ProjectV2Field { id } ... on ProjectV2SingleSelectField { id } ... on ProjectV2IterationField { id } } }
      ... on ProjectV2ItemFieldNumberValue { number field { ... on ProjectV2Field { id } ... on ProjectV2SingleSelectField { id } ... on ProjectV2IterationField { id } } }
      ... on ProjectV2ItemFieldDateValue { date field { ... on ProjectV2Field { id } ... on ProjectV2SingleSelectField { id } ... on ProjectV2IterationField { id } } }
      ... on ProjectV2ItemFieldSingleSelectValue { name optionId field { ... on ProjectV2Field { id } ... on ProjectV2SingleSelectField { id } ... on ProjectV2IterationField { id } } }
      ... on ProjectV2ItemFieldIterationValue { iterationId title field { ... on ProjectV2Field { id } ... on ProjectV2SingleSelectField { id } ... on ProjectV2IterationField { id } } }
      ... on ProjectV2ItemFieldUserValue { users(first: 10) { nodes { login } } field { ... on ProjectV2Field { id } ... on ProjectV2SingleSelectField { id } ... on ProjectV2IterationField { id } } }
      ... on ProjectV2ItemFieldLabelValue { labels(first: 10) { nodes { name } } field { ... on ProjectV2Field { id } ... on ProjectV2SingleSelectField { id } ... on ProjectV2IterationField { id } } }
    }
  }
}`
