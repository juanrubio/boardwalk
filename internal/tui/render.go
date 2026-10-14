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
	// Editor and its change summary are centred modals (A4).
	if m.overlay == overlayNone && (m.pane == paneEdit || m.pane == paneNewDraft) {
		body = m.overlayCenter(body, m.renderFormModal(), w, bodyH)
	}
	if m.overlay == overlayNone && m.pane == paneSummary {
		body = m.overlayCenter(body, m.renderSummaryModal(), w, bodyH)
	}
	switch m.overlay {
	case overlayHelp:
		body = m.overlayCenter(body, m.renderHelp(w), w, bodyH)
	case overlayArchive:
		body = m.overlayCenter(body, m.renderArchiveModal(w), w, bodyH)
	case overlayDelete:
		body = m.overlayCenter(body, m.renderDeleteModal(w), w, bodyH)
	case overlayGroup:
		body = m.overlayCenter(body, m.renderGroupPicker(w), w, bodyH)
	case overlayBoards:
		body = m.overlayCenter(body, m.renderBoardPicker(w), w, bodyH)
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
func (m *model) overlayCenter(body, box string, w, h int) string {
	lines := splitLines(body, h)
	boxLines := strings.Split(box, "\n")
	boxW := 0
	for _, bl := range boxLines {
		if lw := lipgloss.Width(bl); lw > boxW {
			boxW = lw
		}
	}
	left := max(0, (w-boxW)/2)
	pad := strings.Repeat(" ", left)
	start := max(0, (h-len(boxLines))/2)
	for i, bl := range boxLines {
		if start+i < len(lines) {
			lines[start+i] = padVisible(pad+bl, w)
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
	// Sort indicator last, and only when it fits without crowding
	// out the count on narrower terminals (A10).
	if m.sortedByPriority() && len([]rune(left))+len(" · Sorted by Priority")+25 < w {
		left += " · Sorted by Priority"
	}
	if m.filter != "" {
		left += fmt.Sprintf(" · /%s · matching %d of %d loaded", m.filter, count, len(allVisibleBase(m)))
	}
	if m.overlay == overlaySearch {
		left = "Filter: " + m.searchText + "▌"
	}
	if m.overlay == overlayAdd {
		left = "Add issue/PR reference: " + m.addText + "▌  (Enter adds, Esc cancels)"
	}
	if m.overlay == overlayConvert {
		left = "Convert draft to issue in repo: " + m.convertText + "▌  (Enter converts, Esc cancels)"
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
	case overlayDelete:
		if m.deleteStage == 1 {
			return "y again to permanently delete · any other key cancels"
		}
		return "y confirm deletion · any other key cancels"
	case overlayGroup:
		return "↑↓ pick grouping · Enter apply · Esc cancel"
	case overlayBoards:
		return "↑↓ pick board · type to filter · Enter switch · Esc cancel"
	case overlaySearch:
		return "Enter keep filter · Esc clear filter"
	case overlayAdd:
		return "Enter add · Esc cancel"
	case overlayConvert:
		return "Enter convert draft to issue · Esc cancel"
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
		return "Type to edit · ←→ cursor · Tab/↑↓ field · Enter pick · Ctrl-E $EDITOR · Ctrl-S review & save · Esc back" + dirty
	case paneDetail:
		if m.paneFocus {
			return "↑↓/jk scroll · Tab board · Esc close pane"
		}
		return "↑↓/jk board · Tab pane · Enter/Esc close pane · H/L move · e edit · x archive"
	}
	if m.stale {
		return "Showing snapshot — moves wait for live data · ↑↓/jk card · ←→/hl column · Enter detail · q quit"
	}
	if m.view == viewList {
		return "↑↓ row · Enter detail · H/L move · e edit · v kanban · / filter · q quit"
	}
	return "↑↓/jk card · ←→/hl column · H/L move · Enter detail · n draft · a add · / filter · d density · P boards · q quit"
}

// detailTextWidth is the Markdown wrap width for detail bodies: the
// A9 pane width, or near-full width below the split breakpoint.
func (m *model) detailTextWidth() int {
	if m.width < 90 {
		return max(20, m.width-2)
	}
	return max(20, m.width-m.colWidth()-3)
}

func (m *model) renderBoardArea(w, h int) string {
	// Detail split (A9): the board keeps exactly one column (the
	// focused one, at its preferred width) and the pane takes all
	// remaining space on wide terminals (≥90), else full pane.
	// Edit/draft/summary render as centred modals (A4).
	if m.pane == paneDetail && m.detail != nil {
		if w >= 90 {
			boardW := m.colWidth()
			paneW := w - boardW - 1 // 1 separator
			board := m.renderBoard(boardW, h)
			pane := m.renderPane(paneW, h)
			sep := "│"
			if m.paneFocus {
				sep = lipgloss.NewStyle().Foreground(m.theme.Tokens.BorderFocus).Render("┃")
			}
			return joinHorizontal(board, pane, boardW, paneW, h, sep)
		}
		return m.renderPane(w, h)
	}
	return m.renderBoard(w, h)
}

func joinHorizontal(left, right string, lw, rw, h int, sep string) string {
	ll := splitLines(left, h)
	rl := splitLines(right, h)
	out := make([]string, h)
	for i := 0; i < h; i++ {
		l := truncateANSI(ll[i], lw)
		// pad left to lw using visible width approx (strip ANSI for width)
		l = padVisible(l, lw)
		r := truncateANSI(rl[i], rw)
		out[i] = l + sep + r
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

// Preferred column widths (tui.md §3 A3): the board stretches these
// toward maxColWidth, shrinks them toward shrinkFloor to fit every
// column without scrolling, and only then falls back to a window.
func (m *model) colWidth() int {
	if m.density == "compact" {
		return 31
	}
	return 36
}

const (
	maxColWidth    = 56
	collapsedWidth = 15
)

func shrinkFloor(minW int) int {
	if minW <= 31 {
		return 26
	}
	return 30
}

// fluidWidths shares the total width w evenly among cols (collapsed
// columns fixed at collapsedWidth). Widths are capped at maxColWidth.
// It reports false when any column would fall below the shrink floor,
// meaning the whole set cannot be shown at once.
func fluidWidths(w int, cols []column, minW int) ([]int, bool) {
	n := len(cols)
	if n == 0 {
		return nil, false
	}
	collapsedSum, k := 0, 0
	for _, c := range cols {
		if c.collapsed {
			collapsedSum += collapsedWidth
		} else {
			k++
		}
	}
	widths := make([]int, n)
	if k == 0 {
		for i := range widths {
			widths[i] = collapsedWidth
		}
		return widths, true
	}
	avail := w - (n - 1) - collapsedSum
	share, rem := avail/k, avail%k
	if share < shrinkFloor(minW) {
		return nil, false
	}
	idx := 0
	for i, c := range cols {
		if c.collapsed {
			widths[i] = collapsedWidth
			continue
		}
		wd := share
		if idx < rem {
			wd++
		}
		idx++
		if wd > maxColWidth {
			wd = maxColWidth
		}
		widths[i] = wd
	}
	return widths, true
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
	type visCol struct {
		idx   int
		col   column
		width int
	}
	var vis []visCol
	if widths, ok := fluidWidths(w, cols, cw); ok {
		// Every column fits (stretched or shrunk): no scrolling (A3).
		m.colOffset = 0
		for i, c := range cols {
			vis = append(vis, visCol{i, c, widths[i]})
		}
	} else {
		// Scrolling window at the preferred width, stretched to fill.
		m.ensureColVisible(w, cw)
		var sel []column
		var selIdx []int
		used := 0
		for i := m.colOffset; i < len(cols); i++ {
			cwid := cw
			if cols[i].collapsed {
				cwid = collapsedWidth
			}
			if used > 0 && used+1+cwid > w {
				break
			}
			if used == 0 && cwid > w {
				cwid = w
			}
			sel = append(sel, cols[i])
			selIdx = append(selIdx, i)
			used += cwid + 1
		}
		if len(sel) == 0 {
			sel = []column{cols[m.colOffset]}
			selIdx = []int{m.colOffset}
		}
		widths, ok := fluidWidths(w, sel, cw)
		if !ok {
			widths = make([]int, len(sel))
			for i, c := range sel {
				widths[i] = cw
				if c.collapsed {
					widths[i] = collapsedWidth
				}
			}
			if len(widths) == 1 && widths[0] > w {
				widths[0] = w
			}
		}
		for j, c := range sel {
			vis = append(vis, visCol{selIdx[j], c, widths[j]})
		}
	}
	// Header row
	var headerParts []string
	for _, vc := range vis {
		label := fmt.Sprintf("%s (%d)", vc.col.label, len(vc.col.items))
		if vc.col.collapsed {
			label = fmt.Sprintf("%s (%d) [z]", truncateRunes(vc.col.label, 9), len(vc.col.items))
		}
		band := m.theme.HeaderBand(vc.col.color)
		headerParts = append(headerParts, band.Width(vc.width).Render(truncateRunes(label, vc.width)))
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
			l1, l2 := m.compactCardLines(it, width, focused, col.color)
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
		card := m.comfortableCard(col.items[i], width, focusedCol && i == m.row, col.color)
		cl := strings.Split(card, "\n")
		if len(cl) > 5 { // fixed card geometry (mockups.md): never taller
			cl = cl[:5]
		}
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

func (m *model) comfortableCard(it githubx.Item, width int, focused bool, colColor string) string {
	inner := max(4, width-4) // border 2 + padding 2
	// The first content cell is the column accent stripe (A5); the
	// remaining body keeps the pre-A5 text budget minus that cell.
	title := truncateRunes(m.displayTitle(it.Title), inner-3)
	if focused {
		title = "▸" + title
	} else {
		title = " " + title
	}
	marker := m.cardMarker(it)
	if marker != "" {
		title = truncateRunes(marker+m.displayTitle(it.Title), inner-1)
		if focused {
			title = "▸" + truncateRunes(marker+m.displayTitle(it.Title), inner-2)
		}
	}
	kindRef := string(it.Kind) + " " + it.Ref()
	if it.Kind == githubx.KindDraft {
		kindRef = it.Ref()
	} else if lipgloss.Width("  "+kindRef) > inner {
		// Shrunk column (A3): shorten like compact cards (A2); the
		// full reference stays in detail, list view, and on copy.
		kindRef = fmt.Sprintf("%s …#%d", it.Kind, it.Number)
	}
	line2 := " " + truncateRunes(kindRef, inner-2)
	meta := assigneeOf(it)
	pairs := metaPairs(it, m.groupField.Name)
	if len(pairs) > 0 {
		if meta != "" {
			meta += "  "
		}
		meta += strings.Join(pairs, " · ")
	}
	line3 := " " + truncateRunes(meta, inner-2)
	// The focused background must live on each line's own styles: the
	// card-level background alone is reset by the inner title/meta
	// styles, leaving text cells on the default background (A5 fix).
	titleStyle, metaStyle := m.theme.CardTitle, m.theme.Meta
	stripeStyle := m.theme.Stripe(colColor)
	if focused {
		bg := m.theme.Tokens.SelectionBG
		titleStyle = titleStyle.Background(bg)
		metaStyle = metaStyle.Background(bg)
		stripeStyle = stripeStyle.Background(bg)
		title = padRight(title, inner-1)
		line2 = padRight(line2, inner-1)
		line3 = padRight(line3, inner-1)
	}
	stripe := stripeStyle.Render("▎")
	content := strings.Join([]string{
		stripe + titleStyle.Render(title),
		stripe + metaStyle.Render(line2),
		stripe + metaStyle.Render(line3),
	}, "\n")
	style := m.theme.Card
	if focused {
		style = m.theme.CardFocused
	} else {
		style = style.BorderForeground(m.theme.ColumnBorder(colColor))
	}
	// Lip Gloss Width is the total card width (border+padding+text),
	// so text area is width-4, matching inner above.
	return style.Width(width).Render(content)
}

func (m *model) compactCardLines(it githubx.Item, width int, focused bool, colColor string) (string, string) {
	// Gutter cell (A5): column-colour stripe, focus block when focused.
	gutter := m.theme.Stripe(colColor).Render("▎")
	if focused {
		gutter = lipgloss.NewStyle().Foreground(m.theme.Tokens.BorderFocus).Render("▌")
	}
	title := m.cardMarker(it) + m.displayTitle(it.Title)
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
	line2 := gutter + m.theme.Meta.Render(truncateRunes(kindRef, width-1))
	return line1, line2
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
			padRight(truncateRunes(m.displayTitle(it.Title), 24), 24),
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
