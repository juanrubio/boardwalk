package githubx

import (
	"errors"
	"strings"
	"testing"

	ghapi "github.com/cli/go-gh/v2/pkg/api"

	"github.com/juanrubio/boardwalk/internal/exiterr"
)

func TestClassifyInsufficientScopes(t *testing.T) {
	const ghScopeMsg = "Your token has not been granted the required scopes to execute this query. " +
		"The 'projectV2' field requires one of the following scopes: ['read:project'], " +
		"but your token has only been granted the: ['gist', 'read:org', 'repo', 'workflow'] scopes."

	cases := []struct {
		name string
		err  error
	}{
		{
			name: "plain github scope message",
			err:  errors.New("GraphQL: " + ghScopeMsg),
		},
		{
			name: "graphql error with INSUFFICIENT_SCOPES type",
			err: &ghapi.GraphQLError{
				Errors: []ghapi.GraphQLErrorItem{
					{Type: "INSUFFICIENT_SCOPES", Message: ghScopeMsg},
					{Type: "INSUFFICIENT_SCOPES", Message: "duplicate field scope error"},
				},
			},
		},
		{
			name: "graphql error without type but with scope message",
			err: &ghapi.GraphQLError{
				Errors: []ghapi.GraphQLErrorItem{
					{Message: ghScopeMsg},
				},
			},
		},
		{
			name: "INSUFFICIENT_SCOPES token",
			err:  errors.New("INSUFFICIENT_SCOPES"),
		},
		{
			name: "insufficient scopes lowercase",
			err:  errors.New("insufficient scopes for project"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classify(tc.err)
			if exiterr.CodeOf(got) != exiterr.CodeAuth {
				t.Fatalf("CodeOf(%v) = %d, want %d (CodeAuth)", got, exiterr.CodeOf(got), exiterr.CodeAuth)
			}
			be := exiterr.Of(got)
			if !strings.Contains(be.Remediation, "gh auth refresh -s project,repo") {
				t.Fatalf("Remediation = %q, want gh auth refresh -s project,repo", be.Remediation)
			}
			if strings.Contains(be.Message, "duplicate field scope error") {
				t.Fatalf("Message should summarize first GraphQL error, got %q", be.Message)
			}
		})
	}
}
