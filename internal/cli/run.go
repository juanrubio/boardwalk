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

func runProjectList(_ context.Context, opts *Options, _ string) error {
	return unimplemented(opts, "project list")
}

func runProjectUse(_ context.Context, opts *Options, _ string) error {
	return unimplemented(opts, "project use")
}

func runItemList(_ context.Context, opts *Options, _ listFilters) error {
	return unimplemented(opts, "item list")
}

func runItemAdd(_ context.Context, opts *Options, _ string, _ bool, _, _ string) error {
	return unimplemented(opts, "item add")
}

func runItemEdit(_ context.Context, opts *Options, _ string, _, _ string, _ []string) error {
	return unimplemented(opts, "item edit")
}

func runItemMove(_ context.Context, opts *Options, _ string, _, _ string) error {
	return unimplemented(opts, "item move")
}

func runItemArchive(_ context.Context, opts *Options, _ string) error {
	return unimplemented(opts, "item archive")
}

func runBoard(_ context.Context, opts *Options, _ bool) error {
	return unimplemented(opts, "board (TUI)")
}
