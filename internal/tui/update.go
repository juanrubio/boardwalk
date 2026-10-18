package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/cli/browser"

	"github.com/juanrubio/boardwalk/internal/cache"
	"github.com/juanrubio/boardwalk/internal/core"
	"github.com/juanrubio/boardwalk/internal/githubx"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.syncFormLayout()
		return m, nil
	case tea.BackgroundColorMsg:
		// Keep current theme; dark is the fallback per spec. Light is
		// chosen only via explicit luminance of a light background.
		if isLight(msg.Color) && !m.theme.Mono {
			m.theme = lightTheme()
		} else if !m.theme.Mono {
			// stay dark unless currently light and bg is dark
			if !isLight(msg.Color) {
				m.theme = darkTheme()
			}
		}
		return m, nil
	case pageMsg:
		if msg.projectID != "" && msg.projectID != m.project.ID {
			return m, nil // page for a board we have left (A6)
		}
		return m.onPage(msg)
	case boardsMsg:
		m.boardsLoading = false
		if msg.err != nil {
			m.boardsErr = msg.err.Error()
			m.pendingCycle = 0
			if m.overlay == overlayBoards {
				return m, nil
			}
			m.setFooter(statusError, "Boards failed: "+msg.err.Error())
			return m, nil
		}
		m.boards = msg.projects
		sort.Slice(m.boards, func(i, j int) bool {
			if m.boards[i].Owner != m.boards[j].Owner {
				return m.boards[i].Owner < m.boards[j].Owner
			}
			return m.boards[i].Number < m.boards[j].Number
		})
		m.boardsErr = ""
		dir := msg.cycle
		if m.pendingCycle != 0 {
			dir = m.pendingCycle
			m.pendingCycle = 0
		}
		if msg.openPicker || m.overlay == overlayBoards {
			m.openBoardPicker()
			return m, nil
		}
		if dir != 0 {
			return m, m.cycleBoard(dir)
		}
		return m, nil
	case detailMsg:
		if msg.err != nil {
			m.detailErr = msg.err.Error()
			return m, nil
		}
		it := msg.item
		// A detail payload must never strip what the board copy has:
		// field values, content ID, and identity fall back to the
		// existing item when the detail item lacks them.
		if m.board != nil {
			for _, existing := range m.board.Items {
				if existing.ID == it.ID {
					if len(it.Values) == 0 {
						it.Values = existing.Values
					}
					if it.ContentID == "" {
						it.ContentID = existing.ContentID
					}
					if it.Repo == "" {
						it.Repo, it.Number, it.DatabaseID = existing.Repo, existing.Number, existing.DatabaseID
					}
					break
				}
			}
		}
		m.detail = &it
		m.detailBody = msg.body
		m.detailErr = ""
		// If the edit form is open on this item, adopt the fetched
		// Title/Body as the baseline for fields not yet touched.
		if m.form != nil && !m.form.isDraft && m.form.itemID == it.ID {
			for i := range m.form.fields {
				ff := &m.form.fields[i]
				if ff.value() != ff.oldDisp {
					continue
				}
				switch ff.name {
				case "Body":
					ff.oldDisp = it.Body
					ff.setValue(it.Body)
				case "Title":
					ff.oldDisp = it.Title
					ff.setValue(it.Title)
				}
			}
		}
		m.replaceItem(it)
		return m, nil
	case moveTickMsg:
		return m, m.flushMove(msg.itemID)
	case moveResultMsg:
		return m.onMoveResult(msg)
	case reorderTickMsg:
		return m, m.flushReorder(msg.itemID)
	case reorderResultMsg:
		return m.onReorderResult(msg)
	case saveResultMsg:
		if m.summary != nil {
			if m.summary.sent && len(m.summary.results) > 0 {
				// Retry: merge new results into previous by field name.
				byField := map[string]core.Result{}
				for _, r := range msg.results {
					byField[r.Change.Field] = r
				}
				for i, r := range m.summary.results {
					if nr, ok := byField[r.Change.Field]; ok {
						m.summary.results[i] = nr
					}
				}
			} else {
				m.summary.results = msg.results
			}
			m.summary.sent = true
			m.summary.pendingRetry = nil
			m.applySummaryLocally(msg.results)
			if !anyFailed(m.summary.results) {
				m.setFooter(statusSuccess, "Saved.")
			}
		}
		m.savingMsg = ""
		return m, nil
	case archiveResultMsg:
		m.overlay = overlayNone
		if msg.err != nil {
			m.setFooter(statusError, "Archive failed: "+msg.err.Error())
			return m, nil
		}
		if it, ok := m.focusedItem(); ok {
			m.removeItem(it.ID)
		}
		m.pane = paneNone
		m.clampFocus()
		m.setFooter(statusSuccess, "Archived.")
		return m, nil
	case convertResultMsg:
		if msg.err != nil {
			m.setFooter(statusError, "Convert failed: "+msg.err.Error())
			return m, nil
		}
		if m.board != nil {
			for _, existing := range m.board.Items {
				if existing.ID == msg.item.ID && len(msg.item.Values) == 0 {
					msg.item.Values = existing.Values
					break
				}
			}
		}
		m.replaceItem(msg.item)
		m.rebuildFocus()
		m.setFocusedByID(msg.item.ID)
		m.setFooter(statusSuccess, "Converted → "+msg.item.Ref())
		return m, nil
	case deleteResultMsg:
		m.overlay = overlayNone
		m.deleteStage = 0
		if msg.err != nil {
			m.setFooter(statusError, "Delete failed: "+msg.err.Error())
			return m, nil
		}
		if it, ok := m.focusedItem(); ok {
			m.removeItem(it.ID)
		}
		m.pane = paneNone
		m.clampFocus()
		m.setFooter(statusSuccess, "Deleted.")
		return m, nil
	case draftResultMsg:
		if msg.err != nil {
			m.setFooter(statusError, "Create failed: "+msg.err.Error())
			if m.summary != nil {
				m.summary.err = msg.err.Error()
			}
			return m, nil
		}
		m.replaceItem(msg.item)
		m.pane, m.form, m.summary = paneNone, nil, nil
		m.rebuildFocus()
		m.setFocusedByID(msg.item.ID)
		m.setFooter(statusSuccess, "Created "+msg.item.Ref())
		return m, nil
	case addResultMsg:
		m.overlay = overlayNone
		if msg.err != nil {
			m.setFooter(statusError, "Add failed: "+msg.err.Error())
			return m, nil
		}
		if msg.already {
			m.setFooter(statusInfo, msg.item.Ref()+" is already on this project.")
			return m, nil
		}
		m.replaceItem(msg.item)
		m.rebuildFocus()
		m.setFocusedByID(msg.item.ID)
		m.setFooter(statusSuccess, "Added "+msg.item.Ref())
		return m, nil
	case editorDoneMsg:
		if msg.err != nil {
			m.setFooter(statusError, "Editor failed: "+msg.err.Error())
			return m, nil
		}
		if m.form != nil && msg.fieldIdx >= 0 && msg.fieldIdx < len(m.form.fields) {
			m.form.fields[msg.fieldIdx].setValue(msg.body)
			m.setFooter(statusInfo, "Body updated from editor.")
		}
		return m, nil
	case tea.KeyPressMsg:
		return m.onKey(msg)
	}
	if m.form != nil && (m.pane == paneEdit || m.pane == paneNewDraft) {
		// Cursor blink and component-internal messages (A4).
		return m, m.routeFormMsg(msg)
	}
	return m, nil
}

func isLight(c interface {
	RGBA() (uint32, uint32, uint32, uint32)
},
) bool {
	r, g, b, _ := c.RGBA()
	// luminance > 0.6 => light
	return (0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)) > 153
}

func (m *model) onPage(msg pageMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.loading = false
		if m.stale && m.board != nil && len(m.board.Items) > 0 {
			m.loadErr = msg.err.Error()
			m.setFooter(statusError, "Refresh failed: "+msg.err.Error()+" (showing snapshot; r retries)")
			return m, nil
		}
		m.loadErr = msg.err.Error()
		m.setFooter(statusError, msg.err.Error())
		return m, nil
	}
	focusID := ""
	if it, ok := m.focusedItem(); ok {
		focusID = it.ID
	}
	if msg.first {
		// Live data replaces snapshot wholesale on first page.
		m.board = &githubx.Board{Project: m.project, Fields: msg.fields, Items: append([]githubx.Item{}, msg.items...)}
		m.stale = false
		m.loadedAt = time.Now()
	} else if m.board != nil {
		m.board.Items = append(m.board.Items, msg.items...)
	}
	m.rebuildFocus()
	if focusID != "" {
		m.setFocusedByID(focusID)
	}
	m.clampFocus()
	// Prefetch focused detail body neighbours are handled on Enter/live preview.
	if msg.next != "" {
		return m, m.loadPageCmd(msg.next)
	}
	m.loading = false
	// Persist snapshot via cache (LoadBoard normally does; TUI pages do it here).
	if m.svc != nil && m.svc.Cache != nil && m.board != nil {
		snap := cache.Snapshot{Project: m.board.Project, Fields: m.board.Fields, Items: m.board.Items}
		go m.svc.Cache.SaveSnapshot(snap)
	}
	return m, nil
}

func (m *model) onMoveResult(msg moveResultMsg) (tea.Model, tea.Cmd) {
	p := m.pending[msg.itemID]
	if p != nil {
		p.inFlight = false
	}
	if msg.err != nil {
		// A failed write keeps the app open so its outcome stays visible.
		m.quitting = false
		// Roll back to last confirmed value.
		if p != nil {
			m.setItemGroupValue(msg.itemID, p.confirmed, "")
			m.failed[msg.itemID] = msg.err.Error()
			m.setFooter(statusError, "Move failed: "+msg.err.Error())
			delete(m.pending, msg.itemID)
		}
		m.clampFocus()
		// Keep focus on card
		m.setFocusedByID(msg.itemID)
		return m, nil
	}
	// Confirmed: record undo, clear pending (unless a newer target queued)
	if p != nil {
		m.undo = append(m.undo, undoEntry{itemID: msg.itemID, fromName: p.confirmed, fromID: p.fromOptID})
		p.confirmed = msg.toName
		if p.toName == msg.toName && !p.scheduled {
			delete(m.pending, msg.itemID)
		}
	}
	delete(m.failed, msg.itemID)
	m.setFocusedByID(msg.itemID)
	if m.quitting && m.pendingWrites() == 0 {
		return m, tea.Quit
	}
	// If a coalesced newer target is waiting, flush it.
	if p != nil && p.scheduled {
		return m, m.flushMove(msg.itemID)
	}
	return m, nil
}

func (m *model) onReorderResult(msg reorderResultMsg) (tea.Model, tea.Cmd) {
	p := m.pendingPos[msg.itemID]
	if p != nil {
		p.inFlight = false
	}
	if msg.err != nil {
		// A failed write keeps the app open so its outcome stays visible.
		m.quitting = false
		// Roll back only the failed item to its last confirmed
		// neighbourhood, preserving every other item's current
		// order (including other confirmed reorders).
		if p != nil {
			if m.board != nil && p.confirmed != nil {
				present := make(map[string]bool, len(m.board.Items))
				for _, it := range m.board.Items {
					present[it.ID] = true
				}
				if present[msg.itemID] {
					for i, it := range p.confirmed {
						if it.ID != msg.itemID {
							continue
						}
						predecessorID := ""
						for j := i - 1; j >= 0; j-- {
							if present[p.confirmed[j].ID] {
								predecessorID = p.confirmed[j].ID
								break
							}
						}
						m.board.Items = githubx.ApplyReorder(m.board.Items, msg.itemID, predecessorID)
						break
					}
				}
			}
			m.failed[msg.itemID] = msg.err.Error()
			m.setFooter(statusError, "Reorder failed: "+msg.err.Error())
			delete(m.pendingPos, msg.itemID)
		}
		m.clampFocus()
		m.setFocusedByID(msg.itemID)
		return m, nil
	}
	if p != nil {
		if p.scheduled {
			// A newer target is queued: the confirmed order is the
			// previous one with the just-sent reorder applied.
			p.confirmed = githubx.ApplyReorder(p.confirmed, msg.itemID, p.sentAfterID)
		} else {
			if m.board != nil {
				p.confirmed = m.board.Items
			}
			delete(m.pendingPos, msg.itemID)
		}
	}
	delete(m.failed, msg.itemID)
	m.setFocusedByID(msg.itemID)
	if m.quitting && m.pendingWrites() == 0 {
		return m, tea.Quit
	}
	// If a coalesced newer target is waiting, flush it.
	if p != nil && p.scheduled {
		return m, m.flushReorder(msg.itemID)
	}
	return m, nil
}

// setItemGroupValue sets the grouping field value on a local item copy.
func (m *model) setItemGroupValue(itemID, optName, optID string) {
	if m.board == nil || !m.hasGroup {
		return
	}
	for i := range m.board.Items {
		if m.board.Items[i].ID != itemID {
			continue
		}
		found := false
		for j, v := range m.board.Items[i].Values {
			if v.FieldName == m.groupField.Name {
				m.board.Items[i].Values[j].OptionName = optName
				m.board.Items[i].Values[j].OptionID = optID
				found = true
			}
		}
		if !found && optName != "" {
			m.board.Items[i].Values = append(m.board.Items[i].Values, githubx.FieldValue{
				FieldID: m.groupField.ID, FieldName: m.groupField.Name, DataType: "SINGLE_SELECT",
				OptionName: optName, OptionID: optID,
			})
		}
		if optName == "" {
			// remove grouping value
			vals := m.board.Items[i].Values[:0]
			for _, v := range m.board.Items[i].Values {
				if v.FieldName != m.groupField.Name {
					vals = append(vals, v)
				}
			}
			m.board.Items[i].Values = vals
		}
	}
}

// doMove performs an optimistic H/L move (dir -1/+1).
func (m *model) doMove(dir int) tea.Cmd {
	if m.stale {
		m.setFooter(statusWarning, "Showing snapshot — moves are available once live data arrives.")
		return nil
	}
	if !m.hasGroup {
		m.setFooter(statusWarning, "No grouping field to move on.")
		return nil
	}
	it, ok := m.focusedItem()
	if !ok {
		return nil
	}
	cols := m.columns()
	// find current column index of item
	cur := m.col
	if m.view == viewList {
		// find column by item's grouping value
		v, _ := it.Value(m.groupField.Name)
		cur = 0
		for i, c := range cols {
			if c.name == v.OptionName {
				cur = i
			}
		}
		if v.OptionName == "" {
			cur = 0
		}
	}
	target := cur + dir
	if target < 0 || target >= len(cols) {
		m.setFooter(statusInfo, "Already at the "+map[bool]string{true: "first", false: "last"}[dir < 0]+" column.")
		return nil
	}
	tc := cols[target]
	// Optimistic local move
	p := m.pending[it.ID]
	if p == nil {
		confirmed := ""
		if v, ok2 := it.Value(m.groupField.Name); ok2 {
			confirmed = v.OptionName
		}
		p = &pendingMove{itemID: it.ID, confirmed: confirmed}
		m.pending[it.ID] = p
	}
	p.toName, p.toOptID = tc.name, tc.optionID
	m.setItemGroupValue(it.ID, tc.name, tc.optionID)
	m.rebuildFocus()
	m.setFocusedByID(it.ID)
	delete(m.failed, it.ID)
	if p.inFlight {
		p.scheduled = true
		return nil
	}
	p.scheduled = true
	itemID := it.ID
	// Coalesce ~250ms: tick, then flush only if still scheduled.
	return tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg { return moveTickMsg{itemID: itemID} })
}

func (m *model) flushMove(itemID string) tea.Cmd {
	p := m.pending[itemID]
	if p == nil || !p.scheduled || p.inFlight {
		return nil
	}
	p.scheduled = false
	p.inFlight = true
	svc, proj := m.svc, m.project
	field := m.groupField
	toName, toOptID := p.toName, p.toOptID
	return func() tea.Msg {
		if svc == nil || svc.Client == nil {
			return moveResultMsg{itemID: itemID, toName: toName, err: fmt.Errorf("no GitHub client configured")}
		}
		assign := githubx.FieldAssign{FieldID: field.ID, FieldName: field.Name}
		if toName == "" {
			assign.Value = githubx.MutationValue{Clear: true}
		} else {
			id := toOptID
			assign.Value = githubx.MutationValue{OptionID: &id}
		}
		res := svc.Client.SetFieldValues(m.ctx, proj.ID, itemID, []githubx.FieldAssign{assign})
		if len(res) > 0 && res[0].Err != nil {
			return moveResultMsg{itemID: itemID, toName: toName, err: res[0].Err}
		}
		return moveResultMsg{itemID: itemID, toName: toName}
	}
}

// doReorder performs an optimistic J/K within-column reorder
// (dir -1 = up, +1 = down) against the manual board order (A13).
func (m *model) doReorder(dir int) tea.Cmd {
	if m.board == nil {
		return nil
	}
	if m.stale {
		m.setFooter(statusWarning, "Showing snapshot — moves are available once live data arrives.")
		return nil
	}
	if m.sortedByPriority() {
		m.setFooter(statusWarning, "Priority sort is on — press s for manual board order to reorder.")
		return nil
	}
	it, ok := m.focusedItem()
	if !ok {
		return nil
	}
	groupName := ""
	if m.hasGroup {
		groupName = m.groupField.Name
	}
	afterID, ok := githubx.PositionAfterID(m.board.Items, groupName, it.ID, dir)
	if !ok {
		if dir < 0 {
			m.setFooter(statusInfo, "Already at the top of the column.")
		} else {
			m.setFooter(statusInfo, "Already at the bottom of the column.")
		}
		return nil
	}
	if m.pendingPos == nil {
		m.pendingPos = map[string]*pendingReorder{}
	}
	p := m.pendingPos[it.ID]
	if p == nil {
		p = &pendingReorder{itemID: it.ID, confirmed: m.board.Items}
		m.pendingPos[it.ID] = p
	}
	p.afterID = afterID
	m.board.Items = githubx.ApplyReorder(m.board.Items, it.ID, afterID)
	m.rebuildFocus()
	m.setFocusedByID(it.ID)
	delete(m.failed, it.ID)
	if p.inFlight {
		p.scheduled = true
		return nil
	}
	p.scheduled = true
	itemID := it.ID
	// Coalesce ~250ms: tick, then flush only if still scheduled.
	return tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg { return reorderTickMsg{itemID: itemID} })
}

func (m *model) flushReorder(itemID string) tea.Cmd {
	p := m.pendingPos[itemID]
	if p == nil || !p.scheduled || p.inFlight {
		return nil
	}
	p.scheduled = false
	p.inFlight = true
	p.sentAfterID = p.afterID
	svc, proj := m.svc, m.project
	afterID := p.afterID
	return func() tea.Msg {
		if svc == nil || svc.Client == nil {
			return reorderResultMsg{itemID: itemID, err: fmt.Errorf("no GitHub client configured")}
		}
		return reorderResultMsg{itemID: itemID, err: svc.Client.UpdateItemPosition(m.ctx, proj.ID, itemID, afterID)}
	}
}

func (m *model) doUndo() tea.Cmd {
	if len(m.undo) == 0 {
		m.setFooter(statusInfo, "Nothing to undo.")
		return nil
	}
	e := m.undo[len(m.undo)-1]
	m.undo = m.undo[:len(m.undo)-1]
	// Move back to fromName optimistically
	curName := ""
	if m.board != nil {
		for _, it := range m.board.Items {
			if it.ID == e.itemID {
				if v, ok := it.Value(m.groupField.Name); ok {
					curName = v.OptionName
				}
			}
		}
	}
	_ = curName
	optID := ""
	if m.hasGroup {
		if o, ok := m.groupField.OptionByName(e.fromName); ok {
			optID = o.ID
		}
	}
	p := &pendingMove{itemID: e.itemID, confirmed: curName, toName: e.fromName, toOptID: optID, scheduled: true}
	// confirmed for rollback should be current confirmed; approximate with curName
	m.pending[e.itemID] = p
	m.setItemGroupValue(e.itemID, e.fromName, optID)
	m.rebuildFocus()
	m.setFocusedByID(e.itemID)
	return tea.Tick(10*time.Millisecond, func(time.Time) tea.Msg { return moveTickMsg{itemID: e.itemID} })
}

func (m *model) applySummaryLocally(results []core.Result) {
	if m.summary == nil || m.detail == nil {
		return
	}
	// Re-read item from fixture/client via local form values for applied fields.
	for _, r := range results {
		if r.Err != nil {
			continue
		}
		ch := r.Change
		switch {
		case ch.IsTitle:
			m.detail.Title = ch.NewTitle
		case ch.IsBody:
			m.detail.Body = ch.NewBody
		case ch.Assign != nil:
			// rebuild display value from change.New
			m.setLocalField(m.detail, ch.Field, ch.New)
		}
	}
	if m.detail != nil {
		m.replaceItem(*m.detail)
	}
	m.rebuildFocus()
	if m.detail != nil {
		m.setFocusedByID(m.detail.ID)
	}
}

func (m *model) setLocalField(it *githubx.Item, fieldName, display string) {
	if m.board == nil {
		return
	}
	f, ok := githubx.FieldByName(m.board.Fields, fieldName)
	if !ok {
		return
	}
	// Reuse ParseFieldAssignment to build the value.
	b := &githubx.Board{Project: m.project, Fields: m.board.Fields}
	ch, err := core.ParseFieldAssignment(b, *it, fieldName+"="+display)
	if err != nil || ch.Assign == nil {
		return
	}
	a := ch.Assign
	nv := githubx.FieldValue{FieldID: f.ID, FieldName: f.Name, DataType: f.DataType}
	switch {
	case a.Value.Clear:
		// clear below
	case a.Value.OptionID != nil:
		nv.OptionID = *a.Value.OptionID
		nv.OptionName = display
	case a.Value.Text != nil:
		nv.Text = *a.Value.Text
	case a.Value.Number != nil:
		nv.Number = a.Value.Number
	case a.Value.Date != nil:
		nv.Date = parseDateStr(*a.Value.Date)
	case a.Value.IterationID != nil:
		nv.IterationID = *a.Value.IterationID
		nv.IterationTitle = display
	}
	for i, v := range it.Values {
		if v.FieldID == f.ID {
			if a.Value.Clear {
				it.Values = append(it.Values[:i], it.Values[i+1:]...)
			} else {
				it.Values[i] = nv
			}
			return
		}
	}
	if !a.Value.Clear {
		it.Values = append(it.Values, nv)
	}
}

func parseDateStr(s string) *time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

var (
	_ = strings.TrimSpace
	_ = browser.OpenURL
)
