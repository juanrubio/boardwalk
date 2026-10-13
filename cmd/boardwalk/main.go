// Command boardwalk is the Boardwalk executable: a terminal-first way
// to file, edit, and visualize GitHub Projects work (ADR 0001-0011).
package main

import (
	"os"

	"github.com/juanrubio/gh-boardwalk/internal/cache"
	"github.com/juanrubio/gh-boardwalk/internal/cli"
	"github.com/juanrubio/gh-boardwalk/internal/config"
	"github.com/juanrubio/gh-boardwalk/internal/core"
	"github.com/juanrubio/gh-boardwalk/internal/githubx"
)

func main() {
	opts := &cli.Options{}
	opts.NewService = func(o *cli.Options) (*core.Service, error) {
		cfg, err := config.Load()
		if err != nil {
			return nil, err
		}
		c, _ := cache.Open(o.NoCache)
		if c != nil {
			c.Prune()
		}
		return core.New(githubx.NewClient(), cfg, c), nil
	}
	os.Exit(cli.Execute(opts))
}
