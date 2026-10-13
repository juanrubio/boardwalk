package cli

import (
	"context"

	"github.com/juanrubio/gh-boardwalk/internal/exiterr"
)

// Command runners. The read path is implemented in run_read.go and the
// mutations in run_mutate.go; board launches the TUI package.

type listFilters struct {
	Status   string
	Assignee string
	Repo     string
	Type     string
	Search   string
}

func unimplemented(opts *Options, what string) error {
	if _, err := opts.service(); err != nil {
		return err
	}
	return exiterr.Failuref("%s is not implemented yet in this build step", what)
}

func runBoard(_ context.Context, opts *Options, _ bool) error {
	return unimplemented(opts, "board (TUI)")
}
