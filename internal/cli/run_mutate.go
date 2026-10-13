package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"charm.land/huh/v2"

	"github.com/juanrubio/gh-boardwalk/internal/core"
	"github.com/juanrubio/gh-boardwalk/internal/exiterr"
	"github.com/juanrubio/gh-boardwalk/internal/githubx"
)

// isTTY reports whether w is a terminal file.
func isTTY(w interface{}) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// confirmGate enforces the mutation safety contract (cli.md §3):
// dry-run prints the plan and stops; piped runs without --yes refuse
// with exit 2; terminal runs confirm interactively unless --yes.
func confirmGate(opts *Options, summary []string) (proceed bool, err error) {
	printSummary := func(prefix string) {
		if opts.JSON {
			enc := json.NewEncoder(opts.Out)
			_ = enc.Encode(map[string]any{"dryRun": opts.DryRun, "changes": summary})
			return
		}
		_, _ = fmt.Fprintln(opts.Out, prefix)
		for _, line := range summary {
			_, _ = fmt.Fprintf(opts.Out, "  %s\n", line)
		}
	}
	if opts.DryRun {
		printSummary("Dry run — planned changes (nothing was applied):")
		return false, nil
	}
	if opts.Yes {
		return true, nil
	}
	if !isTTY(opts.Out) {
		printSummary("Planned changes:")
		return false, exiterr.Usagef("refusing to mutate without confirmation in a non-terminal").
			WithRemediation("Re-run with --yes to proceed, or --dry-run to preview only.")
	}
	printSummary("Apply these changes?")
	ok := false
	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title("Apply these changes?").Affirmative("Apply").Negative("Cancel").Value(&ok),
	))
	if err := form.Run(); err != nil {
		// Fall back to a plain y/N read when the form cannot run.
		_, _ = fmt.Fprint(opts.Out, "  [y/N] ")
		reader := bufio.NewReader(opts.In)
		line, _ := reader.ReadString('\n')
		ok = strings.ToLower(strings.TrimSpace(line)) == "y"
	}
	if !ok {
		_, _ = fmt.Fprintln(opts.Out, "Aborted — nothing was changed.")
		return false, nil
	}
	return true, nil
}

// reportResults prints per-change outcomes and returns an error when
// any change failed (classified; partial success stays visible).
func reportResults(opts *Options, results []core.Result) error {
	var firstErr error
	applied := 0
	for _, r := range results {
		line := fmt.Sprintf("%s: %s → %s", r.Change.Field, emptyDash(r.Change.Old), r.Change.New)
		if r.Err != nil {
			_, _ = fmt.Fprintf(opts.Out, "✗ %s\n", line)
			if firstErr == nil {
				firstErr = r.Err
			}
		} else {
			applied++
			if !opts.JSON {
				_, _ = fmt.Fprintf(opts.Out, "✓ %s\n", line)
			}
		}
	}
	if opts.JSON {
		type resJSON struct {
			Field   string `json:"field"`
			Old     string `json:"old"`
			New     string `json:"new"`
			Applied bool   `json:"applied"`
			Error   string `json:"error,omitempty"`
		}
		out := make([]resJSON, len(results))
		for i, r := range results {
			e := ""
			if r.Err != nil {
				e = r.Err.Error()
			}
			out[i] = resJSON{Field: r.Change.Field, Old: r.Change.Old, New: r.Change.New, Applied: r.Err == nil, Error: e}
		}
		enc := json.NewEncoder(opts.Out)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
	}
	if firstErr != nil {
		if applied > 0 {
			_, _ = fmt.Fprintf(opts.Out, "Partially applied: %d of %d changes succeeded. Applied changes were not rolled back.\n", applied, len(results))
		}
		return firstErr
	}
	return nil
}

func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// editInEditor opens seed in $VISUAL/$EDITOR (split on whitespace)
// and returns the edited content (cli.md edit UX).
func editInEditor(seed string) (string, error) {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		return "", exiterr.Usagef("no editor configured: set $VISUAL or $EDITOR, or use --body/--body-file")
	}
	f, err := os.CreateTemp("", "boardwalk-body-*.md")
	if err != nil {
		return "", exiterr.Wrap(exiterr.CodeFailure, "create editor temp file", err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.WriteString(seed); err != nil {
		return "", exiterr.Wrap(exiterr.CodeFailure, "seed editor temp file", err)
	}
	_ = f.Close()
	parts := strings.Fields(editor)
	args := append(parts[1:], f.Name()) //nolint:gosec // the user's own configured editor
	cmd := newEditorCmd(parts[0], args)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", exiterr.Wrap(exiterr.CodeFailure, "editor exited with an error (body not saved)", err)
	}
	data, err := os.ReadFile(f.Name())
	if err != nil {
		return "", exiterr.Wrap(exiterr.CodeFailure, "read edited body", err)
	}
	return string(data), nil
}

// loadBoardFor resolves the selected project and loads its board.
func loadBoardFor(ctx context.Context, opts *Options) (*core.Service, githubx.Project, *githubx.Board, error) {
	svc, err := opts.service()
	if err != nil {
		return nil, githubx.Project{}, nil, err
	}
	project, err := svc.ResolveSelectedProject(ctx, opts.Project)
	if err != nil {
		return nil, githubx.Project{}, nil, err
	}
	board, err := svc.LoadBoard(ctx, project, "", nil)
	if err != nil {
		return nil, githubx.Project{}, nil, err
	}
	return svc, project, board, nil
}

type editArgs struct {
	Ref      string
	Title    string
	Body     string
	BodyFile string
	Editor   bool
	Fields   []string
}

func runItemEdit(ctx context.Context, opts *Options, a editArgs) error {
	svc, project, board, err := loadBoardFor(ctx, opts)
	if err != nil {
		return err
	}
	it, err := core.ResolveItemRef(board, a.Ref)
	if err != nil {
		return err
	}
	title, body := a.Title, a.Body
	sources := 0
	if body != "" {
		sources++
	}
	if a.BodyFile != "" {
		sources++
	}
	if a.Editor {
		sources++
	}
	if sources > 1 {
		return exiterr.Usagef("--body, --body-file, and --editor are mutually exclusive")
	}
	if a.BodyFile != "" {
		data, err := os.ReadFile(a.BodyFile) //nolint:gosec // user-named body file
		if err != nil {
			return exiterr.Usagef("cannot read --body-file %s: %v", a.BodyFile, err)
		}
		body = string(data)
	}
	if a.Editor {
		detail, err := svc.Client.LoadItemDetail(ctx, it.ID)
		if err != nil {
			return err
		}
		edited, err := editInEditor(detail.Body)
		if err != nil {
			return err
		}
		if edited == detail.Body {
			_, _ = fmt.Fprintln(opts.Out, "Body unchanged — nothing to do.")
			if title == "" && len(a.Fields) == 0 {
				return nil
			}
		} else {
			body = edited
		}
	}
	var changes []core.Change
	if title != "" {
		changes = append(changes, core.Change{Field: "Title", Old: it.Title, New: title, IsTitle: true, NewTitle: title})
	}
	if body != "" {
		changes = append(changes, core.Change{Field: "Body", Old: "(current body)", New: "(new body)", IsBody: true, NewBody: body})
	}
	for _, fa := range a.Fields {
		ch, err := core.ParseFieldAssignment(board, it, fa)
		if err != nil {
			return err
		}
		changes = append(changes, ch)
	}
	if len(changes) == 0 {
		return exiterr.Usagef("nothing to change: pass --title, --body, or at least one --field \"Name=Value\"")
	}
	proceed, err := confirmGate(opts, core.SummaryLines(changes))
	if err != nil || !proceed {
		return err
	}
	results := svc.ApplyChanges(ctx, project, it, changes)
	return reportResults(opts, results)
}

func runItemMove(ctx context.Context, opts *Options, ref, to, fieldName string) error {
	svc, project, board, err := loadBoardFor(ctx, opts)
	if err != nil {
		return err
	}
	it, err := core.ResolveItemRef(board, ref)
	if err != nil {
		return err
	}
	var field githubx.Field
	if fieldName != "" {
		f, ok := githubx.FieldByName(board.Fields, fieldName)
		if !ok {
			return exiterr.Usagef("unknown field %q on project %s", fieldName, board.Project.Ref())
		}
		field = f
	} else {
		f, ok := githubx.DefaultBoardField(board.Fields)
		if !ok {
			return exiterr.Usagef("project %s has no single-select field to move on", board.Project.Ref()).
				WithRemediation("Name one explicitly with --field, or set fields with `item edit --field`.")
		}
		field = f
	}
	if field.DataType != "SINGLE_SELECT" {
		return exiterr.Usagef("field %q is %s, not SINGLE_SELECT; use item edit --field for it", field.Name, field.DataType)
	}
	ch, err := core.ParseFieldAssignment(board, it, field.Name+"="+to)
	if err != nil {
		return err
	}
	proceed, err := confirmGate(opts, core.SummaryLines([]core.Change{ch}))
	if err != nil || !proceed {
		return err
	}
	results := svc.ApplyChanges(ctx, project, it, []core.Change{ch})
	if err := reportResults(opts, results); err != nil {
		return err
	}
	if !opts.JSON {
		_, _ = fmt.Fprintf(opts.Out, "✓ Moved %s  %s → %s\n", it.Ref(), emptyDash(ch.Old), ch.New)
	}
	return nil
}

func runItemAdd(ctx context.Context, opts *Options, ref string, draft bool, title, body string) error {
	svc, project, board, err := loadBoardFor(ctx, opts)
	if err != nil {
		return err
	}
	if draft {
		if strings.TrimSpace(title) == "" {
			return exiterr.Usagef("--draft requires a non-empty --title")
		}
		summary := []string{fmt.Sprintf("Create draft item %q on %s", title, project.Ref())}
		proceed, err := confirmGate(opts, summary)
		if err != nil || !proceed {
			return err
		}
		it, err := svc.Client.CreateDraft(ctx, project.ID, title, body)
		if err != nil {
			return err
		}
		return printCreated(opts, it, project)
	}
	if ref == "" {
		return exiterr.Usagef("pass an issue/PR reference (owner/repo#number or URL), or --draft --title")
	}
	contentID, preview, err := svc.Client.ResolveContent(ctx, ref)
	if err != nil {
		return err
	}
	for _, existing := range board.Items {
		if existing.Repo == preview.Repo && existing.Number == preview.Number && existing.Kind == preview.Kind {
			_, _ = fmt.Fprintf(opts.Out, "%s is already on project %s as %s — nothing to add.\n", preview.Ref(), project.Ref(), existing.Ref())
			return nil
		}
	}
	summary := []string{fmt.Sprintf("Add %s (%s) %q to %s", preview.Ref(), preview.Kind, preview.Title, project.Ref())}
	proceed, err := confirmGate(opts, summary)
	if err != nil || !proceed {
		return err
	}
	it, err := svc.Client.AddItem(ctx, project.ID, contentID)
	if err != nil {
		return err
	}
	if it.Title == "" {
		it.Title = preview.Title
		it.Repo, it.Number, it.Kind = preview.Repo, preview.Number, preview.Kind
	}
	return printCreated(opts, it, project)
}

func printCreated(opts *Options, it githubx.Item, project githubx.Project) error {
	if opts.JSON {
		enc := json.NewEncoder(opts.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(toItemJSON(it))
	}
	_, _ = fmt.Fprintf(opts.Out, "✓ Added %s to %s\n  ID: %s\n", it.Ref(), project.Ref(), it.ID)
	if it.URL != "" {
		_, _ = fmt.Fprintf(opts.Out, "  %s\n", it.URL)
	}
	return nil
}

func runItemArchive(ctx context.Context, opts *Options, ref string) error {
	svc, project, board, err := loadBoardFor(ctx, opts)
	if err != nil {
		return err
	}
	it, err := core.ResolveItemRef(board, ref)
	if err != nil {
		return err
	}
	summary := []string{fmt.Sprintf("Archive %s (%s) %q — it leaves the board; the issue itself stays open", it.Ref(), it.Kind, it.Title)}
	proceed, err := confirmGate(opts, summary)
	if err != nil || !proceed {
		return err
	}
	if err := svc.Client.ArchiveItem(ctx, project.ID, it.ID); err != nil {
		return err
	}
	if opts.JSON {
		enc := json.NewEncoder(opts.Out)
		return enc.Encode(map[string]any{"ref": it.Ref(), "archived": true})
	}
	_, _ = fmt.Fprintf(opts.Out, "✓ Archived %s\n", it.Ref())
	return nil
}
