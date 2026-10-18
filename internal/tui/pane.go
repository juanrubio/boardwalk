package tui

import (
	"fmt"
	"strings"

	glamour "charm.land/glamour/v2"
	"charm.land/lipgloss/v2"

	"github.com/juanrubio/boardwalk/internal/core"
	"github.com/juanrubio/boardwalk/internal/githubx"
	"github.com/juanrubio/boardwalk/internal/theme"
)

// renderBody renders Markdown with Glamour; falls back to plain text.
// Runs inside a tea.Cmd (off the UI loop) per performance.md §3.
func renderBody(body string, width int, themes ...*theme.Theme) string {
	style := "dark"
	if len(themes) > 0 && themes[0] != nil {
		if themes[0].Mono {
			style = "ascii"
		} else if !themes[0].Dark {
			style = "light"
		}
	}
	if strings.TrimSpace(body) == "" {
		return ""
	}
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(glamour.WithStylePath(style), glamour.WithWordWrap(width))
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
	if m.paneFocus {
		// Visible pane focus (A8): ▸ marker in the focus colour.
		title := "▸ " + truncateRunes(m.displayTitle(it.Title), max(1, w-2))
		b.WriteString(m.theme.Title.Foreground(m.theme.Tokens.BorderFocus).Render(title) + "\n")
	} else {
		b.WriteString(m.theme.Title.Render(truncateRunes(m.displayTitle(it.Title), w)) + "\n")
	}
	b.WriteString(strings.Repeat("─", min(w, 45)) + "\n")
	kindRef := string(it.Kind) + " · " + it.Ref()
	b.WriteString(m.theme.Meta.Render("Kind:     ") + m.theme.KeyHint.Render(kindRef) + "\n")
	if it.URL != "" {
		b.WriteString(m.theme.Meta.Render("URL:      ") + m.theme.KeyHint.Render(it.URL) + "\n")
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
					val = m.theme.Stripe(m.fieldOptionColor(f.Name, val)).Render("● " + val)
				}
				if v.Date != nil {
					val = fmt.Sprintf("%s (%s)", val, relDate(*v.Date))
				}
			}
			b.WriteString(m.theme.Meta.Render(fmt.Sprintf("%-10s ", f.Name+":")) + val + "\n")
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
	// Scroll, clamped to the last full page: an unbounded offset
	// used to run past the end, wrap the view back to the top, and
	// leave Up apparently dead until focus cycled out and back.
	if maxScroll := max(0, len(lines)-h); m.detailScroll > maxScroll {
		m.detailScroll = maxScroll
	}
	if m.detailScroll > 0 {
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
	if !m.theme.Mono {
		style = style.Background(m.theme.Tokens.Surface)
	}
	return m.theme.Surface(style.Render(content))
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
	b.WriteString(m.theme.KeyHint.Render(truncateRunes(title, w)))
	if hdr != title {
		b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Tokens.Warning).Render(strings.TrimPrefix(hdr, title)))
	}
	b.WriteString("\n")
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
			dirty = lipgloss.NewStyle().Foreground(m.theme.Tokens.Warning).Bold(true).Render(" ●")
		}
		label := m.fieldLabel(fmt.Sprintf("%s%-*s", focus, labelW, ff.name+":"), i == f.idx)
		if ff.kind == "body" {
			block := []string{label + dirty}
			m.styleArea(ff)
			for _, l := range strings.Split(ff.area.View(), "\n") {
				block = append(block, "  "+l)
			}
			if ff.err != "" {
				block = append(block, m.theme.Error.Render("  ✗ "+ff.err))
			}
			blocks[i] = block
			continue
		}
		var val string
		switch {
		case ff.readonly:
			val = "(read-only)"
			if ff.oldDisp != "" {
				val = ff.oldDisp + "  (read-only)"
			}
		case ff.kind == "select" || ff.kind == "iteration":
			val = m.theme.Stripe(m.fieldOptionColor(ff.name, ff.value())).Render(ff.value())
			if i == f.idx {
				val += m.theme.Meta.Render("  [Enter pick]")
			}
		default: // textinput kinds: the component renders its own cursor
			m.styleInput(ff)
			val = ff.input.View()
		}
		if ff.readonly {
			val = m.theme.Meta.Render(val)
		}
		line := label + " " + val + dirty
		if ff.err != "" {
			line += m.theme.Error.Render("  ✗ " + ff.err)
		}
		line = truncateANSI(line, w)
		block := []string{line}
		if f.picker && i == f.idx {
			for pi, o := range ff.options {
				mark := "    "
				if pi == f.pickIdx {
					mark = "  ▸ "
				}
				option := m.theme.Stripe(m.fieldOptionColor(ff.name, o)).Render(mark + o)
				if pi == f.pickIdx {
					option = m.theme.Selected(mark+o, w)
				}
				block = append(block, option)
			}
		}
		blocks[i] = block
	}
	footerText := "Tab field · Ctrl-E $EDITOR · Ctrl-S review · Esc back"
	if f.dirtyCount() > 0 {
		footerText = "● changed · " + footerText
	}
	footer := m.formFooter(footerText, w)
	avail := max(1, h-2-len(strings.Split(footer, "\n")))
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
	b.WriteString(footer)
	return strings.Join(splitLines(b.String(), h), "\n")
}

func (m *model) renderSummary(w, h int) string {
	s := m.summary
	if s == nil {
		return ""
	}
	var b strings.Builder
	if s.isDraft {
		b.WriteString(m.theme.KeyHint.Render("Review new draft") + "\n")
		b.WriteString(strings.Repeat("─", min(w, 45)) + "\n")
		b.WriteString(m.theme.Meta.Render("Create draft: ") + m.theme.Title.Render(s.title) + "\n")
		b.WriteString("\nEnter create · e back to form · Esc back\n")
		return strings.Join(splitLines(b.String(), h), "\n")
	}
	ref := ""
	if m.detail != nil {
		ref = m.detail.Ref()
	}
	b.WriteString(m.theme.KeyHint.Render(truncateRunes("Review changes — "+ref, w)) + "\n")
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
	anyFail, anyOK := false, false
	for _, r := range s.results {
		if r.Err != nil {
			anyFail = true
		} else {
			anyOK = true
		}
	}
	if anyOK {
		b.WriteString("Applied (GitHub confirmed):\n")
		for _, r := range s.results {
			if r.Err == nil {
				b.WriteString(m.theme.Success.Render(fmt.Sprintf("  ✓ %s: %s → %s", r.Change.Field, dash(r.Change.Old), r.Change.New)) + "\n")
			}
		}
	}
	if anyFail {
		failedHead := "\nFailed:\n"
		if !anyOK {
			failedHead = "Failed (nothing else was changed):\n"
		}
		b.WriteString(failedHead)
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
		"J/K reorder ↑↓",
		"Enter detail   Tab pane      e edit",
		"n draft        a add issue   x archive",
		"/ filter       g grouping    v list/board",
		"z collapse     d density     r refresh",
		"o browser      y/Y copy      u undo move",
		"D delete       x archive",
		"C convert draft to issue",
		"P boards       [ ] prev/next board",
		"s sort: Priority / manual order",
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

// renderBoardPicker is the A6 board switcher modal: filter line plus
// the visible boards, current board marked •, selection with ▸.
func (m *model) renderBoardPicker(w int) string {
	boxW := min(w-4, 56)
	inner := max(10, boxW-4)
	var lines []string
	lines = append(lines, m.theme.KeyHint.Render("Projects / Boards"))
	if m.boardFilter != "" {
		lines = append(lines, m.theme.Meta.Render("Filter: ")+m.theme.KeyHint.Render(m.boardFilter))
	}
	switch {
	case m.boardsLoading && m.boards == nil:
		lines = append(lines, "Loading boards…")
	case m.boardsErr != "":
		lines = append(lines, "Failed: "+m.boardsErr)
	default:
		list := m.filteredBoards()
		if len(list) == 0 {
			lines = append(lines, "No boards match.")
		}
		// Window around the selection so long lists stay one screen.
		start := 0
		if m.boardIdx >= 10 {
			start = m.boardIdx - 9
		}
		for i := start; i < len(list) && i < start+10; i++ {
			p := list[i]
			mark := "  "
			if i == m.boardIdx {
				mark = "▸ "
			}
			cur := "  "
			if p.ID == m.project.ID {
				cur = "[current] "
			}
			ref := p.Ref()
			if cur != "  " {
				ref = "• " + ref
			}
			row := mark + m.theme.KeyHint.Render(ref)
			if cur != "  " {
				row += " " + m.theme.Meta.Render(cur)
			}
			row += " — " + p.Title
			if i == m.boardIdx {
				row = m.theme.Selected(row, inner)
			}
			lines = append(lines, row)
		}
		if len(list) > 10 {
			lines = append(lines, fmt.Sprintf("… %d boards (type to filter)", len(list)))
		}
	}
	var b strings.Builder
	b.WriteString(m.theme.KeyHint.Render("╭"+strings.Repeat("─", boxW-2)+"╮") + "\n")
	for _, l := range lines {
		b.WriteString(m.theme.KeyHint.Render("│") + " " + padRight(truncateANSI(l, inner), inner) + " " + m.theme.KeyHint.Render("│") + "\n")
	}
	b.WriteString(m.theme.KeyHint.Render("╰" + strings.Repeat("─", boxW-2) + "╯"))
	return m.theme.Surface(b.String())
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
