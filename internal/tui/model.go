package tui

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/juanrubio/gh-boardwalk/internal/config"
	"github.com/juanrubio/gh-boardwalk/internal/core"
	"github.com/juanrubio/gh-boardwalk/internal/githubx"
	"github.com/juanrubio/gh-boardwalk/internal/theme"
)

type viewKind int

const (
	viewKanban viewKind = iota
	viewList
)

type overlayKind int

const (
	overlayNone overlayKind = iota
	overlayHelp
	overlayArchive
	overlayDelete
	overlayGroup
	overlaySearch
	overlayAdd
	overlayDiscard // discard dirty form?
	overlayQuitDirty
)

type paneKind int

const (
	paneNone paneKind = iota
	paneDetail
	paneEdit
	paneSummary
	paneNewDraft
)

type pendingMove struct {
	itemID    string
	fromOptID string
	toOptID   string
	toName    string
	confirmed string // last GitHub-confirmed option name
	scheduled bool
	inFlight  bool
}

type undoEntry struct {
	itemID   string
	fromName string
	fromID   string
}

// model is the Bubble Tea model for the board.
type model struct {
	svc     *core.Service
	project githubx.Project
	ctx     context.Context

	width, height int
	theme         *theme.Theme

	board    *githubx.Board // live (or snapshot) board
	stale    bool           // showing snapshot only
	snapAt   time.Time
	loading  bool
	loadErr  string
	loadedAt time.Time

	view       viewKind
	forceList  bool // no single-select field
	groupField githubx.Field
	hasGroup   bool
	density    string // comfortable|compact
	collapsed  map[string]bool

	col, row  int // focused column / index within column (kanban)
	listRow   int
	colOffset int

	filter     string
	overlay    overlayKind
	pane       paneKind
	paneFocus  bool // Tab into pane
	detail     *githubx.Item
	detailBody string // glamour-rendered (or plain) body
	detailErr  string

	form    *formState
	summary *summaryState

	pending   map[string]*pendingMove
	failed    map[string]string // itemID -> failure reason
	undo      []undoEntry
	footer    string
	quitting  bool
	forceQuit bool

	groupIdx       int
	searchText     string
	addText        string
	addPreview     string
	savingMsg      string
	deleteStage    int
	detailScroll   int
	refreshFocusID string
	vte            bool // VTE-family terminal (Ptyxis/GNOME Terminal): emoji width rules differ
}

type pageMsg struct {
	fields []githubx.Field
	items  []githubx.Item
	next   string
	err    error
	first  bool
}

type moveResultMsg struct {
	itemID string
	toName string
	err    error
}

type moveTickMsg struct{ itemID string }

type detailMsg struct {
	item githubx.Item
	body string
	err  error
}

type (
	saveResultMsg    struct{ results []core.Result }
	archiveResultMsg struct{ err error }
	deleteResultMsg  struct{ err error }
	draftResultMsg   struct {
		item githubx.Item
		err  error
	}
)

type editorDoneMsg struct {
	body     string
	fieldIdx int
	err      error
}

type addResultMsg struct {
	item    githubx.Item
	already bool
	err     error
}

func newModel(svc *core.Service, project githubx.Project, opts RunOptions) *model {
	m := &model{
		svc: svc, project: project, ctx: context.Background(),
		vte:   os.Getenv("VTE_VERSION") != "",
		theme: initialTheme(), density: "comfortable",
		collapsed: map[string]bool{}, pending: map[string]*pendingMove{},
		failed: map[string]string{}, board: &githubx.Board{Project: project},
	}
	if opts.StartInList {
		m.view = viewList
	}
	// Snapshot first (ADR 0011): paint with zero requests.
	if svc != nil && svc.Cache != nil {
		if snap, ok := svc.Cache.LoadSnapshot(project.Ref()); ok && snap.Project.ID == project.ID {
			b := githubx.Board{Project: snap.Project, Fields: snap.Fields, Items: snap.Items}
			m.board = &b
			m.stale = true
			m.snapAt = snap.SavedAt
		}
	}
	m.applyPrefs()
	m.rebuildFocus()
	return m
}

func (m *model) applyPrefs() {
	m.density = "comfortable"
	m.collapsed = map[string]bool{}
	if m.svc == nil || m.svc.Config == nil {
		return
	}
	p, ok := m.svc.Config.UIPrefsFor(m.project.Ref())
	if !ok {
		return
	}
	if p.Density == "compact" || p.Density == "comfortable" {
		m.density = p.Density
	}
	for _, c := range p.Collapsed {
		m.collapsed[c] = true
	}
}

func (m *model) savePrefs() {
	if m.svc == nil || m.svc.Config == nil {
		return
	}
	remembered := ""
	if prev, ok := m.svc.Config.UIPrefsFor(m.project.Ref()); ok {
		remembered = prev.GroupBy
	}
	if m.hasGroup {
		remembered = m.groupField.Name
	}
	var collapsed []string
	// preserve deterministic order by column order
	for _, c := range m.columns() {
		if c.collapsed {
			collapsed = append(collapsed, c.label)
		}
	}
	m.svc.Config.SetUIPrefs(m.project.Ref(), config.UIPrefs{GroupBy: remembered, Density: m.density, Collapsed: collapsed})
	_ = m.svc.Config.Save()
}

func (m *model) rememberedGroup() string {
	if m.svc == nil || m.svc.Config == nil {
		return ""
	}
	p, ok := m.svc.Config.UIPrefsFor(m.project.Ref())
	if !ok {
		return ""
	}
	return p.GroupBy
}

// rebuildFocus recomputes grouping after board data changes, keeping
// focus on the same item ID where possible.
func (m *model) rebuildFocus() {
	if m.board == nil {
		return
	}
	f, ok := groupFieldFor(m.board, m.rememberedGroup())
	m.groupField, m.hasGroup = f, ok
	if !ok {
		// No schema yet (pre-first-page) is not "no single-select field".
		if len(m.board.Fields) == 0 {
			m.clampFocus()
			return
		}
		if m.view == viewKanban {
			m.footer = "Kanban grouping is unavailable for this project: it has no single-select field. Showing list view."
		}
		m.forceList = true
		m.view = viewList
	} else if m.forceList {
		// Grouping became available after having been forced into list.
		m.forceList = false
		m.view = viewKanban
	} else {
		m.forceList = false
	}
	m.clampFocus()
}

func (m *model) visibleItems() []githubx.Item {
	if m.board == nil {
		return nil
	}
	items := make([]githubx.Item, 0, len(m.board.Items))
	for _, it := range m.board.Items {
		if !it.Archived {
			items = append(items, it)
		}
	}
	return filterItems(items, m.filter)
}

func (m *model) columns() []column {
	return buildColumns(m.visibleItems(), m.groupField, m.hasGroup, m.collapsed)
}

func (m *model) clampFocus() {
	cols := m.columns()
	if len(cols) == 0 {
		m.col, m.row = 0, 0
		return
	}
	if m.col >= len(cols) {
		m.col = len(cols) - 1
	}
	if m.col < 0 {
		m.col = 0
	}
	if m.row >= len(cols[m.col].items) {
		m.row = max(0, len(cols[m.col].items)-1)
	}
	if m.row < 0 {
		m.row = 0
	}
	lr := m.listItems()
	if m.listRow >= len(lr) {
		m.listRow = max(0, len(lr)-1)
	}
	if m.listRow < 0 {
		m.listRow = 0
	}
}

func (m *model) listItems() []githubx.Item {
	items := m.visibleItems()
	if m.hasGroup && m.board != nil {
		return core.SortByGrouping(m.board, items, m.groupField)
	}
	return items
}

func (m *model) focusedItem() (githubx.Item, bool) {
	if m.view == viewList {
		items := m.listItems()
		if m.listRow >= 0 && m.listRow < len(items) {
			return items[m.listRow], true
		}
		return githubx.Item{}, false
	}
	cols := m.columns()
	if m.col < 0 || m.col >= len(cols) || m.row < 0 || m.row >= len(cols[m.col].items) {
		return githubx.Item{}, false
	}
	return cols[m.col].items[m.row], true
}

func (m *model) setFocusedByID(id string) {
	for ci, c := range m.columns() {
		for ri, it := range c.items {
			if it.ID == id {
				m.col, m.row = ci, ri
			}
		}
	}
	for i, it := range m.listItems() {
		if it.ID == id {
			m.listRow = i
		}
	}
}

func (m *model) replaceItem(it githubx.Item) {
	if m.board == nil {
		return
	}
	for i := range m.board.Items {
		if m.board.Items[i].ID == it.ID {
			m.board.Items[i] = it
			return
		}
	}
	m.board.Items = append(m.board.Items, it)
}

func (m *model) removeItem(id string) {
	if m.board == nil {
		return
	}
	for i := range m.board.Items {
		if m.board.Items[i].ID == id {
			m.board.Items = append(m.board.Items[:i], m.board.Items[i+1:]...)
			return
		}
	}
}

func (m *model) Init() tea.Cmd {
	m.loading = true
	return m.loadPageCmd("")
}

func (m *model) loadPageCmd(cursor string) tea.Cmd {
	svc, proj := m.svc, m.project
	return func() tea.Msg {
		if svc == nil || svc.Client == nil {
			return pageMsg{err: fmt.Errorf("no GitHub client configured"), first: cursor == ""}
		}
		size := 100
		if cursor == "" {
			size = 50
		}
		fields, items, next, err := svc.Client.LoadBoardPage(m.ctx, proj.ID, cursor, size, "")
		return pageMsg{fields: fields, items: items, next: next, err: err, first: cursor == ""}
	}
}
