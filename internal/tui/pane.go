package tui

import (
	"fmt"
	"strings"

	glamour "charm.land/glamour/v2"
	"charm.land/lipgloss/v2"

	"github.com/juanrubio/gh-boardwalk/internal/core"
	"github.com/juanrubio/gh-boardwalk/internal/githubx"
)

// renderBody renders Markdown with Glamour; falls back to plain text.
// Runs inside a tea.Cmd (off the UI loop) per performance.md §3.
func renderBody(body string, width int) string {
	if strings.TrimSpace(body) == "" {
		return ""
	}
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(glamour.WithStylePath("dark"), glamour.WithWordWrap(width))
	if err != nil {
		return body
	}
	out, err := r.Render(body)
	if err != nil {
		return body
	}
	return strings.TrimRight(out, "\n")
}

func (m *model) renderPane(w, h int) string {
	switch m.pane {
	case paneSummary:
		return m.renderSummary(w, h)
	case paneEdit, paneNewDraft:
		return m.renderForm(w, h)
	default:
		return m.renderDetail(w, h)
	}
}

func (m *model) renderDetail(w, h int) string {
	if m.detail == nil {
		return "No item selected."
	}
	it := *m.detail
	var b strings.Builder
	b.WriteString(m.theme.Title.Render(truncateRunes(m.displayTitle(it.Title), w)) + "\n")
	b.WriteString(strings.Repeat("─", min(w, 45)) + "\n")
	kindRef := string(it.Kind) + " · " + it.Ref()
	fmt.Fprintf(&b, "Kind:     %s\n", kindRef)
	if it.URL != "" {
		fmt.Fprintf(&b, "URL:      %s\n", it.URL)
	}
	if m.board != nil {
		for _, f := range m.board.Fields {
			if f.DataType == "TITLE" {
				continue
			}
			val := "-"
			if v, ok := it.Value(f.Name); ok {
				val = core.DisplayValue(v)
				if f.DataType == "SINGLE_SELECT" {
					val = "● " + val
				}
				if v.Date != nil {
					val = fmt.Sprintf("%s (%s)", val, relDate(*v.Date))
				}
			}
			fmt.Fprintf(&b, "%-10s %s\n", f.Name+":", val)
		}
	}
	if reason, ok := m.failed[it.ID]; ok {
		b.WriteString(m.theme.Error.Render("✗ "+reason) + "\n")
	}
	b.WriteString("\n")
	body := m.detailBody
	if body == "" && it.Body != "" {
		body = it.Body
	}
	if m.detailErr != "" {
		b.WriteString(m.theme.Error.Render("✗ "+m.detailErr) + "\n")
	}
	b.WriteString(body)
	lines := strings.Split(b.String(), "\n")
	// scroll
	if m.detailScroll > 0 && m.detailScroll < len(lines) {
		lines = lines[m.detailScroll:]
	}
	for i, l := range lines {
		lines[i] = truncateANSI(l, w)
	}
	return strings.Join(splitLines(strings.Join(lines, "\n"), h), "\n")
}

// renderFormModal is the edit form as a centred modal (A4).
func (m *model) renderFormModal() string {
	mw, mh := m.modalSize()
	return m.modalBox(m.renderForm(mw-4, mh-4), mw, mh)
}

// renderSummaryModal is the change summary as a centred modal (A4).
func (m *model) renderSummaryModal() string {
	mw, mh := m.modalSize()
	return m.modalBox(m.renderSummary(mw-4, mh-6), mw, mh)
}

func (m *model) modalBox(content string, mw, mh int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Tokens.BorderFocus).
		Padding(0, 1).
		Width(mw).
		MaxHeight(mh)
	return style.Render(content)
}

func (m *model) renderForm(w, h int) string {
	f := m.form
	if f == nil {
		return ""
	}
	var b strings.Builder
	title := "Edit"
	if f.isDraft {
		title = "New draft item"
	} else if m.detail != nil {
		title = "Edit — " + m.displayTitle(m.detail.Title)
	}
	hdr := title
	if !f.isDraft && f.dirtyCount() > 0 {
		hdr += fmt.Sprintf("  (%d changes)", f.dirtyCount())
	}
	b.WriteString(m.theme.Title.Render(truncateRunes(hdr, w)) + "\n")
	b.WriteString(strings.Repeat("─", min(w, 64)) + "\n")
	labelW := 12
	for i := range f.fields {
		if n := len(f.fields[i].name) + 1; n > labelW {
			labelW = n
		}
	}
	// Build one block of lines per field, then window the blocks so
	// the focused field is always visible inside the modal (A4).
	blocks := make([][]string, len(f.fields))
	for i := range f.fields {
		ff := &f.fields[i]
		focus := "  "
		if i == f.idx {
			focus = "▸ "
		}
		dirty := ""
		if ff.value() != ff.oldDisp {
			dirty = " ●"
		}
		label := fmt.Sprintf("%s%-*s", focus, labelW, ff.name+":")
		if ff.kind == "body" {
			block := []string{label + dirty}
			for _, l := range strings.Split(ff.area.View(), "\n") {
				block = append(block, "  "+l)
			}
			if ff.err != "" {
				block = append(block, "  ✗ "+ff.err)
			}
			blocks[i] = block
			continue
		}
		var val string
		switch {
		case ff.readonly:
			val = ff.oldDisp + " (read-only: " + ff.reason + ")"
		case ff.kind == "select" || ff.kind == "iteration":
			val = ff.value()
			if i == f.idx {
				val += "  [Enter pick]"
			}
		default: // textinput kinds: the component renders its own cursor
			val = ff.input.View()
		}
		line := label + " " + val + dirty
		if ff.err != "" {
			line += "  ✗ " + ff.err
		}
		if i == f.idx && ff.kind != "title" && ff.kind != "date" && ff.kind != "number" && ff.kind != "text" {
			line = m.theme.CardTitle.Bold(true).Render(truncateRunes(line, w))
		} else if i != f.idx {
			line = truncateRunes(line, w)
		}
		block := []string{line}
		if f.picker && i == f.idx {
			for pi, o := range ff.options {
				mark := "    "
				if pi == f.pickIdx {
					mark = "  ▸ "
				}
				block = append(block, mark+o)
			}
		}
		blocks[i] = block
	}
	avail := max(1, h-4) // header (2) + footer (2)
	start := 0
	for {
		used, focusEnd := 0, 0
		for i := start; i < len(blocks); i++ {
			used += len(blocks[i])
			if i == f.idx {
				focusEnd = used
			}
		}
		if focusEnd <= avail || start >= f.idx {
			break
		}
		start++
	}
	shown := 0
	for i := start; i < len(blocks) && shown < avail; i++ {
		for _, l := range blocks[i] {
			if shown >= avail {
				break
			}
			b.WriteString(l + "\n")
			shown++
		}
	}
	if !f.isDraft {
		b.WriteString("\n● = changed · Type to edit, arrows move the cursor · Tab field · Ctrl-E $EDITOR · Ctrl-S review & save · Esc back\n")
	} else {
		b.WriteString("\nType to edit, arrows move the cursor · Tab field · Ctrl-E $EDITOR · Ctrl-S review & create · Esc back\n")
	}
	return strings.Join(splitLines(b.String(), h), "\n")
}

func (m *model) renderSummary(w, h int) string {
	s := m.summary
	if s == nil {
		return ""
	}
	var b strings.Builder
	if s.isDraft {
		b.WriteString(m.theme.Title.Render("Review new draft") + "\n")
		b.WriteString(strings.Repeat("─", min(w, 45)) + "\n")
		fmt.Fprintf(&b, "Create draft: %s\n", s.title)
		b.WriteString("\nEnter create · e back to form · Esc back\n")
		return strings.Join(splitLines(b.String(), h), "\n")
	}
	ref := ""
	if m.detail != nil {
		ref = m.detail.Ref()
	}
	b.WriteString(m.theme.Title.Render(truncateRunes("Review changes — "+ref, w)) + "\n")
	b.WriteString(strings.Repeat("─", min(w, 45)) + "\n")
	if !s.sent {
		for _, line := range core.SummaryLines(s.changes) {
			b.WriteString("  " + line + "\n")
		}
		if m.savingMsg != "" {
			b.WriteString("\n" + m.savingMsg + "\n")
		}
		b.WriteString("\nEnter confirm & save · e back to form · Esc back\n")
		return strings.Join(splitLines(b.String(), h), "\n")
	}
	// Sent: per-field results
	b.WriteString("Applied (GitHub confirmed):\n")
	anyFail := false
	for _, r := range s.results {
		if r.Err == nil {
			b.WriteString(m.theme.Success.Render(fmt.Sprintf("  ✓ %s: %s → %s", r.Change.Field, dash(r.Change.Old), r.Change.New)) + "\n")
		}
	}
	for _, r := range s.results {
		if r.Err != nil {
			anyFail = true
		}
	}
	if anyFail {
		b.WriteString("\nFailed (nothing else was changed):\n")
		for _, r := range s.results {
			if r.Err != nil {
				b.WriteString(m.theme.Error.Render(fmt.Sprintf("  ✗ %s: %s → %s", r.Change.Field, dash(r.Change.Old), r.Change.New)) + "\n")
				b.WriteString("    " + r.Err.Error() + "\n")
			}
		}
	}
	if s.err != "" {
		b.WriteString(m.theme.Error.Render("✗ "+s.err) + "\n")
	}
	b.WriteString("\nEnter retry failed changes only\ne      back to the form (edits kept)\nEsc    back (asks before discarding)\n")
	return strings.Join(splitLines(b.String(), h), "\n")
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func (m *model) renderHelp(w int) string {
	boxW := min(w-4, 48)
	inner := max(10, boxW-4)
	lines := []string{
		"─ Keys " + strings.Repeat("─", max(0, inner-7)),
		"↑↓/jk card     ←→/hl column  H/L move",
		"Enter detail   Tab pane      e edit",
		"n draft        a add issue   x archive",
		"/ filter       g grouping    v list/board",
		"z collapse     d density     r refresh",
		"o browser      y/Y copy      u undo move",
		"D delete       x archive",
		"? this help    q quit        Esc back",
	}
	var b strings.Builder
	b.WriteString("╭" + strings.Repeat("─", boxW-2) + "╮\n")
	for _, l := range lines {
		b.WriteString("│ " + padRight(l, boxW-4) + " │\n")
	}
	b.WriteString("╰" + strings.Repeat("─", boxW-2) + "╯")
	return b.String()
}

func (m *model) renderArchiveModal(w int) string {
	it, ok := m.focusedItem()
	ref, kind, title := "", "", ""
	if ok {
		ref, kind, title = it.Ref(), string(it.Kind), it.Title
	}
	boxW := min(w-4, 44)
	lines := []string{
		"─ Archive item? " + strings.Repeat("─", 10),
		ref,
		fmt.Sprintf("%s · %s", kind, title),
		"",
		"It leaves the board; the issue itself",
		"stays open on GitHub.",
		"",
		"[y] Archive    [any key] Cancel",
	}
	var b strings.Builder
	b.WriteString("╭" + strings.Repeat("─", boxW-2) + "╮\n")
	for _, l := range lines {
		b.WriteString("│ " + padRight(truncateRunes(l, boxW-4), boxW-4) + " │\n")
	}
	b.WriteString("╰" + strings.Repeat("─", boxW-2) + "╯")
	return b.String()
}

func (m *model) renderDeleteModal(w int) string {
	it, ok := m.focusedItem()
	ref, kind, title := "", "", ""
	if ok {
		ref, kind, title = it.Ref(), string(it.Kind), it.Title
	}
	consequence := "The issue itself stays open on GitHub."
	if kind == "draft" {
		consequence = "This draft's content will be permanently gone."
	}
	stage := "Delete permanently?"
	if m.deleteStage == 1 {
		stage = "Confirm again — this cannot be undone"
	}
	boxW := min(w-4, 46)
	lines := []string{
		"─ " + stage + " " + strings.Repeat("─", 8),
		ref,
		fmt.Sprintf("%s · %s", kind, title),
		"",
		"This cannot be undone.",
		consequence,
		"",
		"[y] Delete    [any key] Cancel",
	}
	if m.deleteStage == 1 {
		lines[len(lines)-1] = "[y] Yes, delete permanently    [any key] Cancel"
	}
	var b strings.Builder
	b.WriteString("╭" + strings.Repeat("─", boxW-2) + "╮\n")
	for _, l := range lines {
		b.WriteString("│ " + padRight(truncateRunes(l, boxW-4), boxW-4) + " │\n")
	}
	b.WriteString("╰" + strings.Repeat("─", boxW-2) + "╯")
	return b.String()
}

func (m *model) renderGroupPicker(w int) string {
	fields := singleSelectFields(m.board.Fields)
	boxW := min(w-4, 44)
	var b strings.Builder
	b.WriteString("╭" + "─ Group by " + strings.Repeat("─", max(0, boxW-13)) + "╮\n")
	for i, f := range fields {
		mark := "  "
		if i == m.groupIdx {
			mark = "▸ "
		}
		desc := ""
		if len(f.Options) > 0 && f.Options[0].Description != "" {
			desc = ""
		}
		b.WriteString("│ " + padRight(truncateRunes(mark+f.Name+desc, boxW-4), boxW-4) + " │\n")
		for _, o := range f.Options {
			if o.Description != "" {
				// option descriptions shown in picker per §3 (only for selected field)
				if i == m.groupIdx {
					b.WriteString("│ " + padRight(truncateRunes("    "+o.Name+": "+o.Description, boxW-4), boxW-4) + " │\n")
				}
			}
		}
	}
	b.WriteString("╰" + strings.Repeat("─", boxW-2) + "╯")
	return b.String()
}

var _ = githubx.Item{}
