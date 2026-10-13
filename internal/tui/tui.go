// Package tui implements the Bubble Tea board (tui.md, ADR 0006/0007).
package tui

import (
	"context"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

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
	env, kitty := programEnv(os.Environ())
	progOpts := []tea.ProgramOption{tea.WithContext(ctx), tea.WithEnvironment(env)}
	if kitty {
		// programEnv downgrades TERM for the renderer; Kitty is a
		// truecolour terminal, so pin the profile it would have had.
		progOpts = append(progOpts, tea.WithColorProfile(colorprofile.TrueColor))
	}
	p := tea.NewProgram(m, progOpts...)
	_, err := p.Run()
	return err
}

// programEnv returns a copy of env adapted for the Bubble Tea renderer,
// reporting whether the terminal is Kitty. Ultraviolet enables CSI b
// (REP, repeat previous character) purely from TERM=xterm-kitty, but
// early Kitty builds (0.16 has no REP handler at all) silently drop the
// sequence: every REP-compressed run of padding cells vanishes and the
// border after it shifts left by the run length, on board cards and
// modal panels alike. Presenting TERM=xterm-256color keeps the renderer
// on ECH erasure, which those builds implement correctly. The input
// slice is never mutated.
func programEnv(env []string) ([]string, bool) {
	out := make([]string, len(env))
	copy(out, env)
	for i, kv := range out {
		term, ok := strings.CutPrefix(kv, "TERM=")
		if !ok {
			continue
		}
		if strings.Contains(term, "kitty") {
			out[i] = "TERM=xterm-256color"
			return out, true
		}
		return out, false
	}
	return out, false
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
