package cli

import (
	"fmt"
	"strings"

	"github.com/juanrubio/gh-boardwalk/internal/githubx"
)

// renderItemList implements cli.md Section 5: aligned tables on TTYs
// (reference never truncated, KIND/REPOSITORY only when wide) and TSV
// with no header when piped.
func renderItemList(opts *Options, board *githubx.Board, project githubx.Project, items []githubx.Item, total int) {
	statusField, _ := githubx.DefaultBoardField(board.Fields)
	statusOf := func(it githubx.Item) string {
		if statusField.Name == "" {
			return "-"
		}
		return fieldValue(it, statusField.Name)
	}
	if !isTTY(opts.Out) {
		for _, it := range items {
			_, _ = fmt.Fprintf(opts.Out, "%s\t%s\t%s\t%s\n", it.Ref(), statusOf(it), it.Title, assigneeDisplay(it))
		}
		return
	}
	width := termWidth(opts)
	colored := colorMode(opts)
	th := themeFor(opts)

	refW := len("REF")
	for _, it := range items {
		if len(it.Ref()) > refW {
			refW = len(it.Ref())
		}
	}
	statusW := len("STATUS")
	for _, it := range items {
		if len(statusOf(it)) > statusW {
			statusW = len(statusOf(it))
		}
	}
	assignW := len("ASSIGNEE")
	for _, it := range items {
		if len(assigneeDisplay(it)) > assignW {
			assignW = len(assigneeDisplay(it))
		}
	}
	// REF + STATUS + ASSIGNEE + 2-space gaps; TITLE takes the rest.
	titleW := width - refW - statusW - assignW - 6
	if titleW < 10 {
		titleW = 10
	}
	header := func(s string) string {
		if colored {
			return th.Title.Render(s)
		}
		return s
	}
	row := func(cols ...string) string { return strings.Join(cols, "  ") }
	_, _ = fmt.Fprintln(opts.Out, row(
		pad(header("REF"), refW), pad(header("STATUS"), statusW),
		pad(header("TITLE"), titleW), header("ASSIGNEE")))
	_ = th
	for _, it := range items {
		ref := it.Ref()
		if colored && it.URL != "" {
			ref = osc8(it.URL, ref, true)
		}
		// OSC 8 sequences are invisible; pad by visible width.
		refCell := ref + strings.Repeat(" ", max(0, refW-displayWidth(it.Ref())))
		status := statusOf(it)
		if colored {
			status = statusChip(opts, board, statusField.Name, status)
		}
		title := truncateRunes(it.Title, titleW)
		cols := []string{refCell, pad(status, statusDisplayW(status, statusOf(it))), pad(title, titleW), assigneeDisplay(it)}
		_, _ = fmt.Fprintln(opts.Out, row(cols...))
	}
	_, _ = fmt.Fprintf(opts.Out, "\nShowing %d of %d items in %s (%s)\n", len(items), total, project.Title, project.Ref())
}

func statusDisplayW(rendered, raw string) int {
	// The chip adds a dot+space when coloured; visible width is raw+2.
	if rendered != raw {
		return len(raw) + 2
	}
	return len(raw)
}

func pad(s string, w int) string {
	d := displayWidth(s)
	if d >= w {
		return s
	}
	return s + strings.Repeat(" ", w-d)
}

// displayWidth counts runes, ignoring ANSI escape sequences.
func displayWidth(s string) int {
	w := 0
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' || r == '\\' {
				inEsc = false
			}
			continue
		}
		w++
	}
	return w
}

func truncateRunes(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	return string(r[:w-1]) + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
