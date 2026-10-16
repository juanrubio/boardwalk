package cli

import (
	"context"

	"github.com/juanrubio/boardwalk/internal/tui"
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

func runBoard(ctx context.Context, opts *Options, list bool) error {
	svc, err := opts.service()
	if err != nil {
		return err
	}
	project, err := svc.ResolveSelectedProject(ctx, opts.Project)
	if err != nil {
		return err
	}
	return tui.Run(ctx, svc, project, tui.RunOptions{StartInList: list})
}
