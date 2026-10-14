package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/cli/browser"

	"github.com/juanrubio/gh-boardwalk/internal/core"
	"github.com/juanrubio/gh-boardwalk/internal/githubx"
)

// repaintCmd forces the renderer to repaint the whole frame. The
// diff renderer can drift a few cells behind terminals whose width
// tables differ from its own (observed in Kitty); a full repaint on
// modal transitions keeps borders aligned.
func repaintCmd() tea.Cmd {
	return func() tea.Msg { return tea.ClearScreen() }
}

func (m *model) onKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.Key()
	s := msg.String()
	// Key.String for special keys yields names like "up"; text keys yield runes.
	// Normalise via Keystroke for ctrl combos.
	ks := msg.Keystroke()
	_ = key

	// Global quit
	if ks == "ctrl+c" {
		if len(m.pending) > 0 && !m.forceQuit {
			m.forceQuit = true
			m.footer = fmt.Sprintf("saving %d change… (Ctrl-C again to quit now)", len(m.pending))
			return m, nil
		}
		if m.form != nil && m.form.dirtyCount() > 0 && m.overlay != overlayQuitDirty {
			m.overlay = overlayQuitDirty
			return m, nil
		}
		return m, tea.Quit
	}

	// Too-small: only q / ctrl-c
	if m.width > 0 && (m.width < 60 || m.height < 15) {
		if s == "q" {
			return m, tea.Quit
		}
		return m, nil
	}

	// Overlays first
	switch m.overlay {
	case overlayHelp:
		m.overlay = overlayNone
		return m, nil
	case overlayArchive:
		if s == "y" {
			return m, m.archiveCmd()
		}
		m.overlay = overlayNone
		return m, nil
	case overlayDelete:
		if s == "y" && m.deleteStage == 0 {
			m.deleteStage = 1
			return m, nil
		}
		if s == "y" && m.deleteStage == 1 {
			return m, m.deleteCmd()
		}
		m.overlay = overlayNone
		m.deleteStage = 0
		return m, nil
	case overlayDiscard, overlayQuitDirty:
		if s == "y" {
			if m.overlay == overlayQuitDirty {
				return m, tea.Quit
			}
			m.pane, m.form, m.summary = paneNone, nil, nil
			m.overlay = overlayNone
			return m, nil
		}
		m.overlay = overlayNone
		return m, nil
	case overlayBoards:
		return m.onBoardsKey(msg, ks)
	case overlayConvert:
		return m.onConvertKey(msg, ks)
	case overlayGroup:
		return m.onGroupKey(s)
	case overlaySearch:
		return m.onSearchKey(msg, ks)
	case overlayAdd:
		return m.onAddKey(msg, ks)
	case overlayNone:
	}

	// Summary pane keys (named Enter/e/Esc per A2)
	if m.pane == paneSummary && m.summary != nil {
		return m.onSummaryKey(s, ks)
	}
	// Edit / new-draft form keys
	if (m.pane == paneEdit || m.pane == paneNewDraft) && m.form != nil {
		return m.onFormKey(msg, s, ks)
	}
	// Detail pane focused scrolling
	if m.pane == paneDetail && m.paneFocus {
		switch s {
		case "tab":
			m.paneFocus = false
			return m, nil
		case "esc", "q":
			m.pane, m.paneFocus = paneNone, false
			return m, nil
		case "j", "down":
			m.detailScroll++
			return m, nil
		case "k", "up":
			if m.detailScroll > 0 {
				m.detailScroll--
			}
			return m, nil
		}
		return m, nil
	}

	// Board / list keys
	switch ks {
	case "tab":
		if m.pane == paneDetail {
			m.paneFocus = !m.paneFocus
		}
		return m, nil
	}
	switch s {
	case "q":
		if m.pane != paneNone {
			m.pane, m.paneFocus = paneNone, false
			return m, nil
		}
		if len(m.pending) > 0 {
			m.quitting = true
			m.footer = fmt.Sprintf("saving %d change…", len(m.pending))
			// quit when pending settles (onMoveResult)
			return m, nil
		}
		return m, tea.Quit
	case "?":
		m.overlay = overlayHelp
		return m, nil
	case "j", "down":
		m.moveFocus(0, 1)
		return m, m.maybeLoadDetail()
	case "k", "up":
		m.moveFocus(0, -1)
		return m, m.maybeLoadDetail()
	case "l", "right":
		m.moveFocus(1, 0)
		return m, m.maybeLoadDetail()
	case "h", "left":
		m.moveFocus(-1, 0)
		return m, m.maybeLoadDetail()
	case "H":
		return m, m.doMove(-1)
	case "L":
		return m, m.doMove(1)
	case "u":
		return m, m.doUndo()
	case "enter":
		it, ok := m.focusedItem()
		if !ok {
			return m, nil
		}
		m.pane = paneDetail
		m.paneFocus = false
		m.detailScroll = 0
		m.detail = &it
		return m, m.loadDetailCmd(it.ID)
	case "e":
		it, ok := m.focusedItem()
		if !ok {
			return m, nil
		}
		m.pane = paneDetail
		m.detail = &it
		focusCmd := m.startEdit(it)
		return m, tea.Batch(focusCmd, m.loadDetailCmd(it.ID), repaintCmd())
	case "n":
		return m, tea.Batch(m.startNewDraft(), repaintCmd())
	case "a":
		m.overlay = overlayAdd
		m.addText, m.addPreview = "", ""
		return m, nil
	case "x":
		if _, ok := m.focusedItem(); ok {
			m.overlay = overlayArchive
		}
		return m, nil
	case "D":
		if _, ok := m.focusedItem(); ok {
			m.overlay = overlayDelete
			m.deleteStage = 0
		}
		return m, nil
	case "C":
		// Convert draft to issue (A7): repo prompt, drafts only.
		if it, ok := m.focusedItem(); ok {
			if it.Kind != githubx.KindDraft {
				m.footer = "Only drafts can be converted (" + it.Ref() + " is a " + string(it.Kind) + ")."
				return m, nil
			}
			m.overlay = overlayConvert
			m.convertText = ""
			if m.board != nil {
				m.convertText = core.SingleRepo(m.board)
			}
		}
		return m, nil
	case "o":
		if it, ok := m.focusedItem(); ok && it.URL != "" {
			_ = browser.OpenURL(it.URL)
			m.footer = "Opened in browser."
		} else {
			m.footer = "No URL for this item."
		}
		return m, nil
	case "y":
		if it, ok := m.focusedItem(); ok {
			m.footer = "Copied " + it.Ref()
			return m, tea.SetClipboard(it.Ref())
		}
		return m, nil
	case "Y":
		if it, ok := m.focusedItem(); ok && it.URL != "" {
			m.footer = "Copied URL."
			return m, tea.SetClipboard(it.URL)
		}
		return m, nil
	case "g":
		if len(singleSelectFields(m.board.Fields)) == 0 {
			m.footer = "This project has no single-select field to group by."
			return m, nil
		}
		m.overlay = overlayGroup
		m.groupIdx = 0
		fields := singleSelectFields(m.board.Fields)
		for i, f := range fields {
			if m.hasGroup && f.Name == m.groupField.Name {
				m.groupIdx = i
			}
		}
		return m, nil
	case "v":
		if m.view == viewKanban {
			m.view = viewList
			// carry focus
			if it, ok := m.focusedItem(); ok {
				_ = it
			}
			m.syncListToFocus()
		} else {
			if m.forceList {
				m.footer = "Kanban is unavailable: this project has no single-select field."
				return m, nil
			}
			m.view = viewKanban
			m.syncKanbanToFocus()
		}
		return m, nil
	case "z":
		m.toggleCollapse()
		return m, nil
	case "d":
		if m.density == "comfortable" {
			m.density = "compact"
		} else {
			m.density = "comfortable"
		}
		m.savePrefs()
		m.footer = "Density: " + m.density
		return m, nil
	case "P":
		// Board picker (A6): board-level only, never over a pane/form.
		if m.pane != paneNone {
			return m, nil
		}
		if m.boards != nil {
			m.openBoardPicker()
			return m, nil
		}
		m.overlay = overlayBoards
		m.boardFilter = ""
		m.boardIdx = 0
		m.boardsErr = ""
		return m, m.requestBoards(0, true)
	case "[", "]":
		if m.pane != paneNone {
			return m, nil
		}
		dir := 1
		if s == "[" {
			dir = -1
		}
		return m, m.requestBoards(dir, false)
	case "/":
		m.overlay = overlaySearch
		m.searchText = m.filter
		return m, nil
	case "r":
		m.loading = true
		m.footer = "Refreshing…"
		focusID := ""
		if it, ok := m.focusedItem(); ok {
			focusID = it.ID
		}
		m.refreshFocusID = focusID
		return m, m.loadPageCmd("")
	case "esc":
		if m.filter != "" {
			m.filter = ""
			m.footer = "Filter cleared."
			m.clampFocus()
			return m, nil
		}
		if m.pane != paneNone {
			m.pane = paneNone
			return m, nil
		}
		return m, nil
	}
	return m, nil
}

func (m *model) moveFocus(dcol, drow int) {
	if m.view == viewList {
		items := m.listItems()
		m.listRow += drow
		if dcol != 0 {
			// H/L are moves; h/l in list move by page-ish (10 rows)
			m.listRow += dcol * 10
		}
		if m.listRow < 0 {
			m.listRow = 0
		}
		if m.listRow >= len(items) {
			m.listRow = max(0, len(items)-1)
		}
		return
	}
	cols := m.columns()
	if len(cols) == 0 {
		return
	}
	if dcol != 0 {
		// skip collapsed? Focus can land on collapsed to expand; keep simple: allow
		m.col += dcol
		if m.col < 0 {
			m.col = 0
		}
		if m.col >= len(cols) {
			m.col = len(cols) - 1
		}
		// nearest card
		if m.row >= len(cols[m.col].items) {
			m.row = max(0, len(cols[m.col].items)-1)
		}
	}
	if drow != 0 {
		m.row += drow
		if m.row < 0 {
			m.row = 0
		}
		if m.col < len(cols) && m.row >= len(cols[m.col].items) {
			m.row = max(0, len(cols[m.col].items)-1)
		}
	}
	m.detailScroll = 0
}

func (m *model) syncListToFocus() {
	if it, ok := m.focusedItem(); ok {
		for i, li := range m.listItems() {
			if li.ID == it.ID {
				m.listRow = i
			}
		}
	}
}

func (m *model) syncKanbanToFocus() {
	items := m.listItems()
	if m.listRow >= 0 && m.listRow < len(items) {
		m.setFocusedByID(items[m.listRow].ID)
	}
}

func (m *model) toggleCollapse() {
	if m.view == viewList {
		// collapse group of focused row
		it, ok := m.focusedItem()
		if !ok || !m.hasGroup {
			return
		}
		v, _ := it.Value(m.groupField.Name)
		name := v.OptionName
		if name == "" {
			name = "No " + m.groupField.Name
		}
		m.collapsed[name] = !m.collapsed[name]
		m.savePrefs()
		return
	}
	cols := m.columns()
	if m.col < 0 || m.col >= len(cols) {
		return
	}
	label := cols[m.col].label
	m.collapsed[label] = !m.collapsed[label]
	m.savePrefs()
	m.clampFocus()
}

func (m *model) maybeLoadDetail() tea.Cmd {
	if m.pane != paneDetail || m.form != nil {
		return nil
	}
	it, ok := m.focusedItem()
	if !ok {
		return nil
	}
	if m.detail != nil && m.detail.ID == it.ID {
		return nil
	}
	cp := it
	m.detail = &cp
	m.detailBody = ""
	return m.loadDetailCmd(it.ID)
}

func (m *model) loadDetailCmd(itemID string) tea.Cmd {
	svc := m.svc
	return func() tea.Msg {
		if svc == nil || svc.Client == nil {
			return detailMsg{err: fmt.Errorf("no GitHub client configured")}
		}
		it, err := svc.Client.LoadItemDetail(m.ctx, itemID)
		if err != nil {
			return detailMsg{err: err}
		}
		body := renderBody(it.Body, 44)
		return detailMsg{item: it, body: body}
	}
}

func (m *model) deleteCmd() tea.Cmd {
	it, ok := m.focusedItem()
	if !ok {
		return nil
	}
	svc, proj := m.svc, m.project
	return func() tea.Msg {
		if svc == nil || svc.Client == nil {
			return deleteResultMsg{err: fmt.Errorf("no GitHub client configured")}
		}
		return deleteResultMsg{err: svc.Client.DeleteItem(m.ctx, proj.ID, it.ID)}
	}
}

func (m *model) convertCmd(repo string) tea.Cmd {
	it, ok := m.focusedItem()
	if !ok {
		return nil
	}
	svc, proj := m.svc, m.project
	return func() tea.Msg {
		if svc == nil {
			return convertResultMsg{err: fmt.Errorf("no GitHub client configured")}
		}
		converted, err := svc.ConvertDraft(m.ctx, proj, it, repo)
		return convertResultMsg{item: converted, err: err}
	}
}

// onConvertKey handles the convert repo prompt (A7), mirroring the
// add-reference prompt: printable keys edit, Enter converts, Esc cancels.
func (m *model) onConvertKey(msg tea.KeyPressMsg, ks string) (tea.Model, tea.Cmd) {
	switch ks {
	case "esc":
		m.overlay = overlayNone
		return m, nil
	case "enter":
		repo := strings.TrimSpace(m.convertText)
		m.overlay = overlayNone
		if repo == "" {
			m.footer = "Enter a repository (owner/repo) to convert into."
			return m, nil
		}
		m.footer = "Converting…"
		return m, m.convertCmd(repo)
	case "backspace":
		r := []rune(m.convertText)
		if len(r) > 0 {
			m.convertText = string(r[:len(r)-1])
		}
		return m, nil
	}
	if msg.Key().Text != "" {
		m.convertText += msg.Key().Text
	}
	return m, nil
}

func (m *model) archiveCmd() tea.Cmd {
	it, ok := m.focusedItem()
	if !ok {
		return nil
	}
	svc, proj := m.svc, m.project
	return func() tea.Msg {
		if svc == nil || svc.Client == nil {
			return archiveResultMsg{err: fmt.Errorf("no GitHub client configured")}
		}
		return archiveResultMsg{err: svc.Client.ArchiveItem(m.ctx, proj.ID, it.ID)}
	}
}

func (m *model) onGroupKey(s string) (tea.Model, tea.Cmd) {
	fields := singleSelectFields(m.board.Fields)
	switch s {
	case "esc", "q":
		m.overlay = overlayNone
	case "j", "down":
		if m.groupIdx < len(fields)-1 {
			m.groupIdx++
		}
	case "k", "up":
		if m.groupIdx > 0 {
			m.groupIdx--
		}
	case "enter":
		if m.groupIdx >= 0 && m.groupIdx < len(fields) {
			m.groupField = fields[m.groupIdx]
			m.hasGroup = true
			m.forceList = false
			m.col, m.row = 0, 0
			m.savePrefs()
			m.footer = "Grouped by " + m.groupField.Name
		}
		m.overlay = overlayNone
	}
	return m, nil
}

// onBoardsKey handles the board picker (A6): arrows move, printable
// keys extend the filter, Enter switches, Esc cancels.
func (m *model) onBoardsKey(msg tea.KeyPressMsg, ks string) (tea.Model, tea.Cmd) {
	list := m.filteredBoards()
	switch ks {
	case "esc":
		m.overlay = overlayNone
		return m, nil
	case "enter":
		if m.boardIdx >= 0 && m.boardIdx < len(list) {
			p := list[m.boardIdx]
			m.overlay = overlayNone
			return m, m.switchTo(p)
		}
		return m, nil
	case "down":
		if m.boardIdx < len(list)-1 {
			m.boardIdx++
		}
		return m, nil
	case "up":
		if m.boardIdx > 0 {
			m.boardIdx--
		}
		return m, nil
	case "backspace":
		r := []rune(m.boardFilter)
		if len(r) > 0 {
			m.boardFilter = string(r[:len(r)-1])
			m.boardIdx = 0
		}
		return m, nil
	}
	if msg.Key().Text != "" {
		m.boardFilter += msg.Key().Text
		m.boardIdx = 0
	}
	return m, nil
}

func (m *model) onSearchKey(msg tea.KeyPressMsg, ks string) (tea.Model, tea.Cmd) {
	switch ks {
	case "enter":
		m.filter = m.searchText
		m.overlay = overlayNone
		m.col, m.row, m.listRow = 0, 0, 0
		m.clampFocus()
		return m, nil
	case "esc":
		m.filter = ""
		m.searchText = ""
		m.overlay = overlayNone
		m.clampFocus()
		return m, nil
	case "backspace":
		r := []rune(m.searchText)
		if len(r) > 0 {
			m.searchText = string(r[:len(r)-1])
		}
		m.filter = m.searchText
		m.clampFocus()
		return m, nil
	}
	if msg.Key().Text != "" {
		m.searchText += msg.Key().Text
		m.filter = m.searchText
		m.col, m.row, m.listRow = 0, 0, 0
		m.clampFocus()
	}
	return m, nil
}

func (m *model) onAddKey(msg tea.KeyPressMsg, ks string) (tea.Model, tea.Cmd) {
	switch ks {
	case "esc":
		m.overlay = overlayNone
		return m, nil
	case "backspace":
		r := []rune(m.addText)
		if len(r) > 0 {
			m.addText = string(r[:len(r)-1])
		}
		return m, nil
	case "enter":
		ref := strings.TrimSpace(m.addText)
		if ref == "" {
			m.overlay = overlayNone
			return m, nil
		}
		// duplicate check
		if m.board != nil {
			if it, err := core.ResolveItemRef(m.board, ref); err == nil {
				m.footer = it.Ref() + " is already on this project."
				m.overlay = overlayNone
				return m, nil
			}
		}
		svc, proj := m.svc, m.project
		return m, func() tea.Msg {
			if svc == nil || svc.Client == nil {
				return addResultMsg{err: fmt.Errorf("no GitHub client configured")}
			}
			contentID, tpl, err := svc.Client.ResolveContent(m.ctx, ref)
			if err != nil {
				return addResultMsg{err: err}
			}
			_ = tpl
			it, err := svc.Client.AddItem(m.ctx, proj.ID, contentID)
			if err != nil {
				return addResultMsg{err: err}
			}
			// If AddItem returned an item already present (fixture), detect by ref pre-check done above.
			return addResultMsg{item: it}
		}
	}
	if msg.Key().Text != "" {
		m.addText += msg.Key().Text
	}
	return m, nil
}

var _ = githubx.Item{}
