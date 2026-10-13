// Package cli implements Boardwalk's Cobra command tree (ADR 0003
// noun-first grammar). Commands are thin: they parse flags, call the
// shared core service, and render. No GitHub logic lives here.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/juanrubio/gh-boardwalk/internal/config"
	"github.com/juanrubio/gh-boardwalk/internal/core"
	"github.com/juanrubio/gh-boardwalk/internal/exiterr"
)

// Version is set at release time via -ldflags.
var Version = "dev"

// Options carries the persistent global flags (cli.md Section 3).
type Options struct {
	Project string
	JSON    bool
	DryRun  bool
	Yes     bool
	NoCache bool
	Color   string

	Out io.Writer // stdout (defaults to os.Stdout)
	Err io.Writer // stderr (defaults to os.Stderr)

	// NewService builds the shared core service. It is a field so
	// tests can substitute fixture-backed services. cmd/boardwalk
	// sets it to the real go-gh-backed factory.
	NewService func(opts *Options) (*core.Service, error)
}

// NewRootCmd builds the full command tree.
func NewRootCmd(opts *Options) *cobra.Command {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.Err == nil {
		opts.Err = os.Stderr
	}
	root := &cobra.Command{
		Use:   "boardwalk",
		Short: "File, edit, and visualize GitHub Projects work from the terminal",
		Long: `Boardwalk is a terminal-first way to file, edit, and visualize
GitHub Projects (Projects v2) work without opening a browser.

It reuses your authenticated GitHub CLI (gh) session; it never stores
tokens. Run 'boardwalk project list' to find a board, then
'boardwalk item list' or 'boardwalk board' to work on it.`,
		Version:       Version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	pf := root.PersistentFlags()
	pf.StringVar(&opts.Project, "project", "", "project alias, URL, or owner/number (overrides BOARDWALK_PROJECT and the saved default)")
	pf.BoolVar(&opts.JSON, "json", false, "print bare JSON (no envelope) for scripts")
	pf.BoolVar(&opts.DryRun, "dry-run", false, "print the planned change and change nothing")
	pf.BoolVar(&opts.Yes, "yes", false, "skip interactive confirmation (never skips validation)")
	pf.BoolVar(&opts.NoCache, "no-cache", false, "bypass the local schema cache and board snapshot for this run")
	pf.StringVar(&opts.Color, "color", "auto", "color handling: auto, always, or never")

	root.AddCommand(newProjectCmd(opts))
	root.AddCommand(newItemCmd(opts))
	root.AddCommand(newBoardCmd(opts))
	return root
}

// Execute runs the root command and maps failures to ADR 0009 exits.
func Execute(opts *Options) int {
	root := NewRootCmd(opts)
	if err := root.Execute(); err != nil {
		be := exiterr.Of(err)
		writeError(opts, be)
		return be.Code
	}
	return exiterr.CodeOK
}

func writeError(opts *Options, be *exiterr.Error) {
	if opts.JSON {
		payload := map[string]any{"error": map[string]any{
			"code": be.Category, "message": be.Message, "remediation": be.Remediation,
		}}
		enc := json.NewEncoder(opts.Err)
		_ = enc.Encode(payload)
		return
	}
	_, _ = fmt.Fprintf(opts.Err, "✗ %s\n", be.Message)
	if be.Remediation != "" {
		_, _ = fmt.Fprintf(opts.Err, "  %s\n", be.Remediation)
	}
}

// service lazily builds the shared core service for a command.
func (o *Options) service() (*core.Service, error) {
	if o.NewService != nil {
		return o.NewService(o)
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return core.New(nil, cfg, nil), nil
}
