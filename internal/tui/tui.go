// Package tui implements the Bubble Tea board (tui.md, ADR 0006/0007).
package tui

import (
	"context"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/juanrubio/gh-boardwalk/internal/core"
	"github.com/juanrubio/gh-boardwalk/internal/githubx"
	"github.com/juanrubio/gh-boardwalk/internal/theme"
)

// RunOptions configures a TUI session.
type RunOptions struct {
	StartInList bool
}

// Run starts the interactive board for project. A cached snapshot (ADR
// 0011) paints first with zero requests; live pages then stream in.
func Run(ctx context.Context, svc *core.Service, project githubx.Project, opts RunOptions) error {
	m := newModel(svc, project, opts)
	p := tea.NewProgram(m, tea.WithContext(ctx))
	_, err := p.Run()
	return err
}

func initialTheme() *theme.Theme {
	if os.Getenv("NO_COLOR") != "" {
		return theme.Monochrome()
	}
	if os.Getenv("TERM") == "dumb" {
		return theme.Monochrome()
	}
	return theme.Dark()
}
