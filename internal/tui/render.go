package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/juanrubio/gh-boardwalk/internal/core"
	"github.com/juanrubio/gh-boardwalk/internal/githubx"
	"github.com/juanrubio/gh-boardwalk/internal/theme"
)

func darkTheme() *theme.Theme  { return theme.Dark() }
func lightTheme() *theme.Theme { return theme.Light() }

// View renders the current frame.
func (m *model) View() tea.View {
	w, h := m.width, m.height
	if w == 0 {
		w, h = 80, 24
	}
	var content string
	if w < 60 || h < 15 {
		content = m.viewTooSmall(w, h)
	} else {
		content = m.viewMain(w, h)
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m *model) viewTooSmall(w, h int) string {
	lines := []string{
		"", "", center("Terminal too small", w),
		center("Boardwalk needs 60 × 15 cells", w),
		center(fmt.Sprintf("this terminal is %d × %d", w, h), w), "",
		center("Resize to continue · q quit", w),
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines[:min(h, len(lines))], "\n")
}

func center(s string, w int) string {
	n := len([]rune(s))
	if n >= w {
		return truncateRunes(s, w)
	}
	pad := (w - n) / 2
	return strings.Repeat(" ", pad) + s
}

func (m *model) viewMain(w, h int) string {
	header := m.renderHeader(w)
	footer := m.renderFooter(w)
	bodyH := max(1, h-2)
	var body string
	body = m.renderBoardArea(w, bodyH)
	switch m.overlay {
	case overlayHelp:
		body = overlayCenter(body, m.renderHelp(w), bodyH)
	case overlayArchive:
		body = overlayCenter(body, m.renderArchiveModal(w), bodyH)
	case overlayGroup:
		body = overlayCenter(body, m.renderGroupPicker(w), bodyH)
	}
	// Overlay prompts (search/add) replace header line 2 handling via footer/header
	lines := []string{header}
	lines = append(lines, splitLines(body, bodyH)...)
	lines = append(lines, footer)
	// Pad/truncate to h
	out := make([]string, 0, h)
	for i := 0; i < h && i < len(lines); i++ {
		out = append(out, truncateANSI(lines[i], w))
	}
	for len(out) < h {
		out = append(out, "")
	}
	return strings.Join(out, "\n")
}

func splitLines(s string, n int) []string {
	ls := strings.Split(s, "\n")
	for len(ls) < n {
		ls = append(ls, "")
	}
	return ls[:n]
}

// overlayCenter draws box over body lines (simple replacement of middle lines).
func overlayCenter(body, box string, h int) string {
	lines := splitLines(body, h)
	boxLines := strings.Split(box, "\n")
	start := max(0, (h-len(boxLines))/2)
	for i, bl := range boxLines {
		if start+i < len(lines) {
			lines[start+i] = bl
		}
	}
	return strings.Join(lines, "\n")
}

func (m *model) renderHeader(w int) string {
	title := m.project.Title
	if title == "" {
		title = m.project.Ref()
	}
	left := fmt.Sprintf("Boardwalk · %s (%s)", title, m.project.Ref())
	if m.view == viewList {
		left += " · List"
	}
	if m.hasGroup {
		left += " · Grouped by " + m.groupField.Name
	}
	count := 0
	if m.board != nil {
		count = len(m.visibleItems())
	}
	left += fmt.Sprintf(" · %d items", count)
	if m.filter != "" {
		left += fmt.Sprintf(" · /%s · matching %d of %d loaded", m.filter, count, len(allVisibleBase(m)))
	}
	if m.overlay == overlaySearch {
		left = "Filter: " + m.searchText + "▌"
	}
	if m.overlay == overlayAdd {
		left = "Add issue/PR reference: " + m.addText + "▌  (Enter adds, Esc cancels)"
	}
	right := ""
	switch {
	case m.loading:
		right = "refreshing…"
	case m.stale:
		right = "snapshot · updated " + relAgo(m.snapAt) + " · stale"
	case !m.loadedAt.IsZero():
		right = "updated " + relAgo(m.loadedAt)
	}
	right += " · ? Help"
	// fit
	if len([]rune(left))+len([]rune(right))+1 > w {
		left = truncateRunes(left, max(0, w-len([]rune(right))-1))
	}
	gap := w - len([]rune(left)) - len([]rune(right))
	if gap < 1 {
		gap = 1
	}
	return m.theme.Title.Render(left) + strings.Repeat(" ", gap) + m.theme.Meta.Render(right)
}

func allVisibleBase(m *model) []githubx.Item {
	if m.board == nil {
		return nil
	}
	n := 0
	for _, it := range m.board.Items {
		if !it.Archived {
			n++
		}
	}
	_ = n
	return m.board.Items
}

func (m *model) renderFooter(w int) string {
	if m.footer != "" {
		// transient footer message takes precedence, still show keys after ·
		msg := m.footer
		hint := m.footerHint()
		s := msg
		if hint != "" {
			s = msg + " · " + hint
		}
		return truncateRunes(s, w)
	}
	return truncateRunes(m.footerHint(), w)
}

func (m *model) footerHint() string {
	switch m.overlay {
	case overlayHelp:
		return "? close help · Esc back"
	case overlayArchive:
		return "y archive (not optimistic) · any other key cancels"
	case overlayGroup:
		return "↑↓ pick grouping · Enter apply · Esc cancel"
	case overlaySearch:
		return "Enter keep filter · Esc clear filter"
	case overlayAdd:
		return "Enter add · Esc cancel"
	case overlayDiscard:
		return "y discard changes · any other key keep editing"
	case overlayQuitDirty:
		return "y quit and discard · any other key stay"
	}
	switch m.pane {
	case paneSummary:
		if m.summary != nil && m.summary.sent && anyFailed(m.summary.results) {
			return "Enter retry failed · e back to form · Esc back (asks before discarding)"
		}
		if m.savingMsg != "" {
			return m.savingMsg
		}
		return "Enter confirm & save · e back to form · Esc back"
	case paneEdit, paneNewDraft:
		if m.form != nil && m.form.picker {
			return "↑↓ choose · Enter select · Esc cancel picker"
		}
		dirty := ""
		if m.form != nil && m.form.dirtyCount() > 0 {
			dirty = fmt.Sprintf(" · %d changes · Ctrl-S review & save", m.form.dirtyCount())
		}
		return "Tab/↑↓ field · Enter pick · Ctrl-S review & save · Esc back (asks if dirty)" + dirty
	case paneDetail:
		if m.paneFocus {
			return "↑↓/jk scroll · Tab board · Esc close pane"
		}
		return "↑↓/jk board · Tab pane · H/L move · e edit · x archive · Esc close pane"
	}
	if m.stale {
		return "Showing snapshot — moves wait for live data · ↑↓/jk card · ←→/hl column · Enter detail · q quit"
	}
	if m.view == viewList {
		return "↑↓ row · Enter detail · H/L move · e edit · v kanban · / filter · q quit"
	}
	return "↑↓/jk card · ←→/hl column · H/L move · Enter detail · n draft · a add · / filter · d density · q quit"
}

func (m *model) renderBoardArea(w, h int) string {
	// Detail split: board 60% / pane 40% on wide terminals (≥90), else full pane.
	if m.pane != paneNone && m.detail != nil || m.pane == paneNewDraft || m.pane == paneEdit || m.pane == paneSummary {
		if w >= 90 {
			paneW := w * 40 / 100
			boardW := w - paneW - 1 // 1 separator
			board := m.renderBoard(boardW, h)
			pane := m.renderPane(paneW, h)
			return joinHorizontal(board, pane, boardW, paneW, h)
		}
		return m.renderPane(w, h)
	}
	return m.renderBoard(w, h)
}

func joinHorizontal(left, right string, lw, rw, h int) string {
	ll := splitLines(left, h)
	rl := splitLines(right, h)
	out := make([]string, h)
	for i := 0; i < h; i++ {
		l := truncateANSI(ll[i], lw)
		// pad left to lw using visible width approx (strip ANSI for width)
		l = padVisible(l, lw)
		r := truncateANSI(rl[i], rw)
		out[i] = l + "│" + r
	}
	return strings.Join(out, "\n")
}

func padVisible(s string, w int) string {
	vw := lipgloss.Width(s)
	if vw >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vw)
}

func truncateANSI(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	// Fall back to rune truncation (styles may be cut; acceptable for chrome lines)
	return lipgloss.NewStyle().MaxWidth(w).Render(s)
}

func (m *model) renderBoard(w, h int) string {
	if m.loadErr != "" && (m.board == nil || len(m.board.Items) == 0) {
		return m.theme.Error.Render("✗ " + m.loadErr)
	}
	if m.view == viewList {
		return m.renderList(w, h)
	}
	return m.renderKanban(w, h)
}

// --- Kanban ---

func (m *model) colWidth() int {
	if m.density == "compact" {
		return 31
	}
	return 36
}

func (m *model) renderKanban(w, h int) string {
	cols := m.columns()
	if len(cols) == 0 {
		if m.loading {
			return "Loading board…"
		}
		return "No items match."
	}
	cw := m.colWidth()
	// visible columns: fit full columns; collapsed are 15 wide
	// Compute window starting at colOffset ensuring focused visible.
	m.ensureColVisible(w, cw)
	// Build visible slice
	type visCol struct {
		idx   int
		col   column
		width int
	}
	var vis []visCol
	used := 0
	for i := m.colOffset; i < len(cols); i++ {
		cwid := cw
		if cols[i].collapsed {
			cwid = 15
		}
		if used > 0 && used+1+cwid > w {
			break
		}
		if used == 0 && cwid > w {
			cwid = w
		}
		vis = append(vis, visCol{i, cols[i], cwid})
		used += cwid + 1
	}
	if len(vis) == 0 {
		vis = append(vis, visCol{m.colOffset, cols[m.colOffset], w})
	}
	// Header row
	var headerParts []string
	for _, vc := range vis {
		label := fmt.Sprintf("%s (%d)", vc.col.label, len(vc.col.items))
		if vc.col.collapsed {
			label = fmt.Sprintf("%s (%d) [z]", truncateRunes(vc.col.label, 9), len(vc.col.items))
		}
		style := m.theme.ColumnHeader
		if vc.col.color != "" && !m.theme.Mono {
			style = style.Foreground(m.theme.OptionColor(vc.col.color))
		}
		headerParts = append(headerParts, padRightStyled(style.Render(truncateRunes(label, vc.width)), vc.width))
	}
	header := strings.Join(headerParts, " ")
	// off-screen hints
	hiddenRight := len(cols) - (m.colOffset + len(vis))
	if hiddenRight > 0 {
		hint := fmt.Sprintf("%d cols ▸", hiddenRight)
		// append if fits, else replace tail
		if lipgloss.Width(header)+1+len([]rune(hint)) <= w {
			header += " " + hint
		}
	}
	if m.colOffset > 0 {
		hint := fmt.Sprintf("◂ %d ", m.colOffset)
		header = hint + header
		header = truncateANSI(header, w)
	}
	lines := []string{header}
	bodyH := max(0, h-1)
	// Render each column body lines
	colLines := make([][]string, len(vis))
	for vi, vc := range vis {
		colLines[vi] = m.renderColumn(vc.col, vc.width, bodyH, vc.idx == m.col)
	}
	for r := 0; r < bodyH; r++ {
		var parts []string
		for vi := range vis {
			parts = append(parts, padVisible(colLines[vi][r], vis[vi].width))
		}
		lines = append(lines, strings.Join(parts, " "))
	}
	return strings.Join(lines, "\n")
}

func padRightStyled(s string, w int) string { return padVisible(s, w) }

func (m *model) ensureColVisible(w, cw int) {
	// Estimate: count columns fitting from colOffset
	for {
		cols := m.columns()
		if m.col < m.colOffset {
			m.colOffset = m.col
			continue
		}
		// compute visible count
		used, visCount := 0, 0
		for i := m.colOffset; i < len(cols); i++ {
			cwid := cw
			if cols[i].collapsed {
				cwid = 15
			}
			if visCount > 0 && used+1+cwid > w {
				break
			}
			used += cwid + 1
			visCount++
		}
		if m.col < m.colOffset+visCount {
			break
		}
		m.colOffset++
		if m.colOffset > m.col {
			m.colOffset = m.col
			break
		}
	}
}

func (m *model) renderColumn(col column, width, bodyH int, focusedCol bool) []string {
	lines := make([]string, bodyH)
	for i := range lines {
		lines[i] = ""
	}
	if col.collapsed {
		lines[0] = truncateRunes("collapsed", width)
		return lines
	}
	if m.density == "compact" {
		// 2 lines per card
		per := max(1, bodyH/2)
		start := scrollStart(m.row, per, focusedCol)
		idx := 0
		for i := start; i < len(col.items) && idx < per; i++ {
			it := col.items[i]
			focused := focusedCol && i == m.row
			l1, l2 := m.compactCardLines(it, width, focused)
			lines[idx*2] = l1
			if idx*2+1 < bodyH {
				lines[idx*2+1] = l2
			}
			idx++
		}
		if focusedCol && start+per < len(col.items) {
			lines[bodyH-1] = fmt.Sprintf("↓ %d more", len(col.items)-(start+per))
		}
		return lines
	}
	// comfortable: 5 lines per card
	per := max(1, bodyH/5)
	start := scrollStart(m.row, per, focusedCol)
	idx := 0
	for i := start; i < len(col.items) && idx < per; i++ {
		card := m.comfortableCard(col.items[i], width, focusedCol && i == m.row)
		cl := strings.Split(card, "\n")
		for j, l := range cl {
			if idx*5+j < bodyH {
				lines[idx*5+j] = l
			}
		}
		idx++
	}
	if focusedCol && start+per < len(col.items) && bodyH > 0 {
		lines[bodyH-1] = fmt.Sprintf("↓ %d more", len(col.items)-(start+per))
	}
	return lines
}

func scrollStart(focus, per int, focusedCol bool) int {
	if !focusedCol || per <= 0 {
		return 0
	}
	if focus < per {
		return 0
	}
	return focus - per + 1
}

func (m *model) cardMarker(it githubx.Item) string {
	if _, ok := m.pending[it.ID]; ok {
		return "◌ "
	}
	if _, ok := m.failed[it.ID]; ok {
		return "✗ "
	}
	return ""
}

func (m *model) comfortableCard(it githubx.Item, width int, focused bool) string {
	inner := max(4, width-4) // border 2 + padding 2
	title := truncateRunes(it.Title, inner-2)
	if focused {
		title = "▸" + title
	} else {
		title = " " + title
	}
	marker := m.cardMarker(it)
	if marker != "" {
		title = truncateRunes(marker+it.Title, inner)
		if focused {
			title = "▸" + truncateRunes(marker+it.Title, inner-1)
		}
	}
	kindRef := string(it.Kind) + " " + it.Ref()
	if it.Kind == githubx.KindDraft {
		kindRef = it.Ref()
	}
	line2 := "  " + truncateRunes(kindRef, inner-2)
	meta := assigneeOf(it)
	pairs := metaPairs(it, m.groupField.Name)
	if len(pairs) > 0 {
		if meta != "" {
			meta += "  "
		}
		meta += strings.Join(pairs, " · ")
	}
	line3 := "  " + truncateRunes(meta, inner-2)
	content := strings.Join([]string{
		m.theme.CardTitle.Render(title),
		m.theme.Meta.Render(line2),
		m.theme.Meta.Render(line3),
	}, "\n")
	style := m.theme.Card
	if focused {
		style = m.theme.CardFocused
	}
	return style.Width(width - 2).Render(content)
}

func (m *model) compactCardLines(it githubx.Item, width int, focused bool) (string, string) {
	gutter := " "
	if focused {
		gutter = "▌"
	}
	title := m.cardMarker(it) + it.Title
	// highlight search match by underline via style when filtered
	t := truncateRunes(title, width-1)
	if focused {
		t = m.theme.CardTitle.Bold(true).Render(t)
	}
	line1 := gutter + t
	ref := it.Ref()
	ass := assigneeOf(it)
	full := ref
	if ass != "" {
		full = ref + " " + ass
	}
	if len([]rune(full))+1 > width {
		// shorten to …#number (A2)
		if it.Kind != githubx.KindDraft {
			full = fmt.Sprintf("…#%d", it.Number)
			if ass != "" {
				full += " " + ass
			}
		}
	}
	kindRef := full
	if it.Kind != githubx.KindDraft {
		// compact mock shows "issue …#42 @user" - prefix kind when fits
		cand := string(it.Kind) + " " + full
		if len([]rune(cand))+1 <= width {
			kindRef = cand
		}
	}
	line2 := gutter + truncateRunes(kindRef, width-1)
	return line1, m.theme.Meta.Render(line2)
}

// --- List ---

func (m *model) renderList(w, h int) string {
	items := m.listItems()
	header := " REFERENCE                  KIND  TITLE                    STATUS         PRIORITY ASSIGNEE"
	// Build dynamically: reference never truncated, others by width
	header = truncateRunes(header, w)
	lines := []string{header}
	bodyH := max(0, h-1)
	per := bodyH
	start := 0
	if m.listRow >= per && per > 0 {
		start = m.listRow - per + 1
	}
	for i := start; i < len(items) && i < start+per; i++ {
		it := items[i]
		gutter := " "
		if i == m.listRow {
			gutter = "▌"
		}
		status := ""
		if m.hasGroup {
			if v, ok := it.Value(m.groupField.Name); ok {
				status = v.OptionName
			}
		}
		pri := fieldDisplay(it, "Priority")
		ass := assigneeOf(it)
		target := fieldDisplay(it, "Target")
		if target == "-" {
			// try any date field
			target = firstDateDisplay(it)
		}
		row := fmt.Sprintf("%s%s %s %s %s %s %s",
			gutter,
			padRight(it.Ref(), 26),
			padRight(string(it.Kind), 5),
			padRight(truncateRunes(it.Title, 24), 24),
			padRight(status, 14),
			padRight(pri, 8),
			ass)
		if target != "" && target != "-" {
			row += " " + target
		}
		if i == m.listRow {
			row = m.theme.CardTitle.Bold(true).Render(truncateRunes(row, w))
		} else {
			row = truncateRunes(row, w)
		}
		lines = append(lines, row)
	}
	if start+per < len(items) {
		lines = append(lines, fmt.Sprintf(" ↓ %d more…", len(items)-(start+per)))
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines[:min(h, len(lines))], "\n")
}

func fieldDisplay(it githubx.Item, name string) string {
	if v, ok := it.Value(name); ok {
		return core.DisplayValue(v)
	}
	return "-"
}

func firstDateDisplay(it githubx.Item) string {
	for _, v := range it.Values {
		if v.Date != nil {
			return fmt.Sprintf("%s (%s)", v.Date.Format("2006-01-02"), relDate(*v.Date))
		}
	}
	return "-"
}

var _ = githubx.Item{}
