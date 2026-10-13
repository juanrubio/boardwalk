// Package tui implements the Bubble Tea board (tui.md). The model,
// views, forms, and interaction rules land in build step 6; this
// package currently fixes the public entry point the CLI calls.
package tui

import (
	"context"

	"github.com/juanrubio/gh-boardwalk/internal/core"
	"github.com/juanrubio/gh-boardwalk/internal/exiterr"
	"github.com/juanrubio/gh-boardwalk/internal/githubx"
)

// RunOptions configures a TUI session.
type RunOptions struct {
	StartInList bool
}

// Run starts the interactive board for project. Implemented in step 6.
func Run(_ context.Context, _ *core.Service, _ githubx.Project, _ RunOptions) error {
	return exiterr.Failuref("board (TUI) is not implemented yet in this build step")
}
