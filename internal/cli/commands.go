package cli

import (
	"github.com/spf13/cobra"

	"github.com/juanrubio/boardwalk/internal/exiterr"
)

// Command constructors. Behaviour is implemented incrementally per the
// build plan; each verb exists from the scaffold so --help, completion,
// and the mockup targets are real.

func newProjectCmd(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Discover, select, and alias GitHub Projects",
	}
	var owner string
	list := &cobra.Command{
		Use:   "list",
		Short: "List accessible Projects v2",
		Example: `  boardwalk project list
  boardwalk project list --owner juanrubio`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runProjectList(cmd.Context(), opts, owner)
		},
	}
	list.Flags().StringVar(&owner, "owner", "", "only projects owned by this user or organisation login")
	var as string
	use := &cobra.Command{
		Use:   "use <ref>",
		Short: "Make a project the default, optionally saving an alias for it",
		Example: `  boardwalk project use juanrubio/3
  boardwalk project use juanrubio/3 --as life-board
  boardwalk project use https://github.com/users/juanrubio/projects/3`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectUse(cmd.Context(), opts, args[0], as)
		},
	}
	use.Flags().StringVar(&as, "as", "", "also save this alias name (lowercase letters, digits, hyphens) for the project")
	cmd.AddCommand(list, use)
	return cmd
}

func newItemCmd(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "item",
		Short: "List, add, edit, move, and archive project items",
	}
	cmd.AddCommand(newItemListCmd(opts))
	cmd.AddCommand(newItemAddCmd(opts))
	cmd.AddCommand(newItemEditCmd(opts))
	cmd.AddCommand(newItemMoveCmd(opts))
	cmd.AddCommand(newItemArchiveCmd(opts))
	cmd.AddCommand(newItemDeleteCmd(opts))
	cmd.AddCommand(newItemConvertCmd(opts))
	return cmd
}

func newItemListCmd(opts *Options) *cobra.Command {
	var status, assignee, repo, typ, search string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List items on the selected project",
		Example: `  boardwalk item list
  boardwalk item list --status "In Progress" --assignee @me
  boardwalk item list --json | jq '.[].title'`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runItemList(cmd.Context(), opts, listFilters{Status: status, Assignee: assignee, Repo: repo, Type: typ, Search: search})
		},
	}
	f := cmd.Flags()
	f.StringVar(&status, "status", "", "only items whose status/default board field equals this option")
	f.StringVar(&assignee, "assignee", "", "only items assigned to this login (or @me)")
	f.StringVar(&repo, "repo", "", "only items from this owner/repo")
	f.StringVar(&typ, "type", "", "only items of this type: issue, pr, or draft")
	f.StringVar(&search, "search", "", "free text or GitHub project filter syntax")
	return cmd
}

func newItemAddCmd(opts *Options) *cobra.Command {
	var draft bool
	var title, body string
	cmd := &cobra.Command{
		Use:   "add [issue-or-pr-ref]",
		Short: "Add an existing issue/PR, or create a draft item",
		Example: `  boardwalk item add juanrubio/snazzyemail#42
  boardwalk item add --draft --title "Investigate export flow"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref := ""
			if len(args) == 1 {
				ref = args[0]
			}
			return runItemAdd(cmd.Context(), opts, ref, draft, title, body)
		},
	}
	f := cmd.Flags()
	f.BoolVar(&draft, "draft", false, "create a draft item instead of adding an issue/PR")
	f.StringVar(&title, "title", "", "draft title (required with --draft)")
	f.StringVar(&body, "body", "", "draft body")
	return cmd
}

func newItemEditCmd(opts *Options) *cobra.Command {
	var title, body, bodyFile string
	var editor bool
	var fields []string
	cmd := &cobra.Command{
		Use:   "edit <ref>",
		Short: "Edit an item's title/body and project fields",
		Example: `  boardwalk item edit juanrubio/snazzyemail#42 --field "Priority=P0"
  boardwalk item edit draft:91204910 --title "New title" --dry-run
  boardwalk item edit juanrubio/snazzyemail#42 --editor`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runItemEdit(cmd.Context(), opts, editArgs{Ref: args[0], Title: title, Body: body, BodyFile: bodyFile, Editor: editor, Fields: fields})
		},
	}
	f := cmd.Flags()
	f.StringVar(&title, "title", "", "new title (where GitHub permits)")
	f.StringVar(&body, "body", "", "new body (where GitHub permits)")
	f.StringVar(&bodyFile, "body-file", "", "read the new body from this file (mutually exclusive with --body and --editor)")
	f.BoolVar(&editor, "editor", false, "open the body in $VISUAL/$EDITOR (seeded with the current body)")
	f.StringArrayVar(&fields, "field", nil, `set a project field, e.g. --field "Status=In Progress" (repeatable)`)
	return cmd
}

func newItemMoveCmd(opts *Options) *cobra.Command {
	var to, field string
	cmd := &cobra.Command{
		Use:   "move <ref>",
		Short: "Move an item to another status/column",
		Example: `  boardwalk item move juanrubio/snazzyemail#42 --to Done
  boardwalk item move draft:91204910 --to "In Progress" --field Status`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runItemMove(cmd.Context(), opts, args[0], to, field)
		},
	}
	f := cmd.Flags()
	f.StringVar(&to, "to", "", "target option of the board field (required)")
	f.StringVar(&field, "field", "", "single-select field to move on (default: the project's default board field)")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func newItemArchiveCmd(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:     "archive <ref>",
		Short:   "Archive an item (leaves the board; the issue itself stays open)",
		Example: `  boardwalk item archive juanrubio/snazzyemail#42`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runItemArchive(cmd.Context(), opts, args[0])
		},
	}
}

func newItemDeleteCmd(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <ref>",
		Short: "Permanently delete an item (cannot be undone; see also archive)",
		Example: `  boardwalk item delete juanrubio/snazzyemail#42
  boardwalk item delete draft:91204910 --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runItemDelete(cmd.Context(), opts, args[0])
		},
	}
}

func newItemConvertCmd(opts *Options) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "convert <ref>",
		Short: "Convert a draft item into an issue",
		Example: `  boardwalk item convert draft:91204910 --repo juanrubio/snazzyemail
  boardwalk item convert draft:91204910 --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runItemConvert(cmd.Context(), opts, args[0], repo)
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "repository for the new issue (default: the board's single linked repository)")
	return cmd
}

func newBoardCmd(opts *Options) *cobra.Command {
	var list bool
	cmd := &cobra.Command{
		Use:   "board",
		Short: "Open the interactive TUI board",
		Example: `  boardwalk board
  boardwalk board --project life-board --list`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runBoard(cmd.Context(), opts, list)
		},
	}
	cmd.Flags().BoolVar(&list, "list", false, "start in list view instead of kanban")
	return cmd
}

// errNotWiredYet is replaced command-by-command as the build proceeds.
var _ = exiterr.CodeUsage
