package core

import (
	"testing"

	"github.com/juanrubio/gh-boardwalk/internal/exiterr"
	"github.com/juanrubio/gh-boardwalk/internal/githubx"
)

func fixtureBoard() *githubx.Board {
	return &githubx.Board{
		Project: githubx.Project{ID: "PVT_1", Number: 1, Title: "SnazzyEmail", Owner: "juanrubio"},
		Items: []githubx.Item{
			{ID: "PVTI_a", DatabaseID: 101, Kind: githubx.KindIssue, Title: "Export crash", Repo: "juanrubio/snazzyemail", Number: 42},
			{ID: "PVTI_b", DatabaseID: 102, Kind: githubx.KindDraft, Title: "Spike"},
			{ID: "PVTI_c", DatabaseID: 103, Kind: githubx.KindDraft, Title: "Spike"},
		},
	}
}

func TestResolveItemRefForms(t *testing.T) {
	b := fixtureBoard()
	tests := []struct {
		name       string
		ref        string
		wantNumber int
		wantKind   githubx.Kind
	}{
		{name: "issue reference", ref: "juanrubio/snazzyemail#42", wantNumber: 42, wantKind: githubx.KindIssue},
		{name: "node id", ref: "PVTI_a", wantNumber: 42, wantKind: githubx.KindIssue},
		{name: "exact title", ref: "Export crash", wantNumber: 42, wantKind: githubx.KindIssue},
		{name: "issue URL", ref: "https://github.com/juanrubio/snazzyemail/issues/42", wantNumber: 42, wantKind: githubx.KindIssue},
		{name: "draft short id", ref: "draft:102", wantNumber: 0, wantKind: githubx.KindDraft},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it, err := ResolveItemRef(b, tt.ref)
			if err != nil {
				t.Fatalf("ResolveItemRef(%q): %v", tt.ref, err)
			}
			if it.Number != tt.wantNumber || it.Kind != tt.wantKind {
				t.Fatalf("ResolveItemRef(%q) = #%d %s, want #%d %s", tt.ref, it.Number, it.Kind, tt.wantNumber, tt.wantKind)
			}
		})
	}
}

func TestResolveItemRefErrors(t *testing.T) {
	b := fixtureBoard()
	tests := []struct {
		name     string
		ref      string
		wantCode int
	}{
		{name: "missing issue", ref: "juanrubio/snazzyemail#99", wantCode: exiterr.CodeNotFound},
		{name: "ambiguous title", ref: "Spike", wantCode: exiterr.CodeUsage},
		{name: "missing draft", ref: "draft:999", wantCode: exiterr.CodeNotFound},
		{name: "empty reference", ref: "", wantCode: exiterr.CodeUsage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveItemRef(b, tt.ref)
			if err == nil {
				t.Fatalf("ResolveItemRef(%q) succeeded, want code %d", tt.ref, tt.wantCode)
			}
			if exiterr.CodeOf(err) != tt.wantCode {
				t.Fatalf("ResolveItemRef(%q) code = %d, want %d (%v)", tt.ref, exiterr.CodeOf(err), tt.wantCode, err)
			}
		})
	}
}
