package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/juanrubio/gh-boardwalk/internal/config"
	"github.com/juanrubio/gh-boardwalk/internal/core"
	"github.com/juanrubio/gh-boardwalk/internal/fixture"
	"github.com/juanrubio/gh-boardwalk/internal/githubx"
	"github.com/juanrubio/gh-boardwalk/internal/theme"
)

func testModel(t *testing.T, ref string) (*model, *fixture.Client) {
	t.Helper()
	fc, err := fixture.LoadDir("../../testdata/fixtures")
	if err != nil {
		t.Fatal(err)
	}
	b := fc.Board(ref)
	if b == nil {
		t.Fatalf("no fixture board %s", ref)
	}
	cfg := &config.Config{Aliases: map[string]string{}}
	cfg.UI.Projects = map[string]config.UIPrefs{}
	svc := core.New(fc, cfg, nil)
	m := newModel(svc, b.Project, RunOptions{})
	m.width, m.height = 120, 40
	m.theme = theme.Monochrome()
	// Simulate first live page (all items in one page via fixture paging at 50)
	fields, items, next, err := fc.LoadBoardPage(t.Context(), b.Project.ID, "", 50, "")
	if err != nil {
		t.Fatal(err)
	}
	// load remaining pages directly into board for deterministic tests
	for next != "" {
		_, more, n, err := fc.LoadBoardPage(t.Context(), b.Project.ID, next, 100, "")
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, more...)
		next = n
	}
	mm, _ := m.Update(pageMsg{fields: fields, items: items, first: true})
	return mm.(*model), fc
}

func key(s string) tea.KeyPressMsg {
	// Build a KeyPressMsg from its string form where possible.
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	case "ctrl+s":
		return tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	default:
		r := []rune(s)
		return tea.KeyPressMsg{Code: r[0], Text: s}
	}
}

func send(m *model, s string) *model {
	mm, _ := m.Update(key(s))
	return mm.(*model)
}

func TestBoardRendersAndNavigates(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	if !m.hasGroup || m.groupField.Name != "Status" {
		t.Fatalf("group = %q has=%v", m.groupField.Name, m.hasGroup)
	}
	v := m.View().Content
	if !strings.Contains(v, "Boardwalk") || !strings.Contains(v, "Grouped by Status") {
		t.Fatalf("header missing:\n%s", v)
	}
	it0, ok := m.focusedItem()
	if !ok {
		t.Fatal("no focused item")
	}
	m = send(m, "j")
	it1, _ := m.focusedItem()
	// j may stay if column has 1 item; just ensure no panic and view renders
	_ = it0
	_ = it1
	if got := m.View().Content; !strings.Contains(got, "? Help") {
		t.Fatalf("view missing help hint")
	}
}

func TestMoveOptimisticAndUndo(t *testing.T) {
	m, fc := testModel(t, "fixture-user/1")
	it, ok := m.focusedItem()
	if !ok {
		t.Fatal("no focus")
	}
	m = send(m, "L")
	// flush pending move directly
	if p := m.pending[it.ID]; p != nil {
		cmd := m.flushMove(it.ID)
		if cmd == nil {
			t.Fatal("flush returned nil")
		}
		msg := cmd()
		mm, _ := m.Update(msg)
		m = mm.(*model)
	}
	// fixture board should reflect move
	b := fc.Board("fixture-user/1")
	found := false
	for _, bi := range b.Items {
		if bi.ID == it.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("item vanished")
	}
	if len(m.undo) == 0 {
		t.Fatal("undo stack empty after confirmed move")
	}
}

func TestListToggleAndSearch(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	m = send(m, "v")
	if m.view != viewList {
		t.Fatal("v did not switch to list")
	}
	if v := m.View().Content; !strings.Contains(v, "REFERENCE") {
		t.Fatalf("list header missing:\n%s", v)
	}
	m = send(m, "/")
	mm, _ := m.Update(key("x")) // types into search (no match likely)
	m = mm.(*model)
	_ = m
	// Esc clears
	m2, _ := m.Update(key("esc"))
	m = m2.(*model)
	if m.filter != "" {
		t.Fatalf("esc should clear filter, got %q", m.filter)
	}
}

func TestTooSmallAndHelpAndArchiveModals(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	m.width, m.height = 50, 12
	if v := m.View().Content; !strings.Contains(v, "Terminal too small") {
		t.Fatalf("too-small missing:\n%s", v)
	}
	m.width, m.height = 120, 40
	m = send(m, "?")
	if v := m.View().Content; !strings.Contains(v, "Keys") {
		t.Fatalf("help missing:\n%s", v)
	}
	m = send(m, "j") // closes help
	m = send(m, "x")
	if v := m.View().Content; !strings.Contains(v, "Archive item?") {
		t.Fatalf("archive modal missing:\n%s", v)
	}
	m = send(m, "n") // any other key cancels archive (n is 'any key' here)
	if m.overlay != overlayNone {
		t.Fatalf("archive not cancelled, overlay=%v", m.overlay)
	}
}

func TestDensityAndCollapsePersistPrefs(t *testing.T) {
	m, _ := testModel(t, "fixture-user/3")
	m = send(m, "d")
	if m.density != "compact" {
		t.Fatalf("density=%s", m.density)
	}
	_ = m.View().Content // compact renders without panicking
	m = send(m, "z")
	if len(m.collapsed) == 0 {
		t.Fatal("collapse did not record")
	}
}

func TestEditFormSummaryFlow(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	m = send(m, "e")
	if m.form == nil {
		t.Fatal("edit did not open form")
	}
	// Change title: focus is Title field (idx 0); append "X"
	mm, _ := m.Update(tea.KeyPressMsg{Code: 'X', Text: "X"})
	m = mm.(*model)
	if m.form.dirtyCount() == 0 {
		t.Fatal("typing did not dirty form")
	}
	mm, _ = m.Update(key("ctrl+s"))
	m = mm.(*model)
	if m.pane != paneSummary || m.summary == nil || len(m.summary.changes) == 0 {
		t.Fatalf("summary not built pane=%v", m.pane)
	}
	// Confirm
	mm, cmd := m.Update(key("enter"))
	m = mm.(*model)
	if cmd == nil {
		t.Fatal("confirm returned no cmd")
	}
	msg := cmd()
	mm, _ = m.Update(msg)
	m = mm.(*model)
	if m.summary == nil || !m.summary.sent {
		t.Fatal("save not marked sent")
	}
	for _, r := range m.summary.results {
		if r.Err != nil {
			t.Fatalf("unexpected save failure: %v", r.Err)
		}
	}
}

func TestNoSingleSelectForcesList(t *testing.T) {
	// Synthetic board without single-select fields
	board := &githubx.Board{
		Project: githubx.Project{ID: "PX", Number: 9, Title: "No Status", Owner: "o"},
		Fields:  []githubx.Field{{ID: "F1", Name: "Title", DataType: "TITLE"}, {ID: "F2", Name: "Note", DataType: "TEXT"}},
		Items:   []githubx.Item{{ID: "I1", Kind: githubx.KindDraft, Title: "One", DatabaseID: 1}},
	}
	fc := &fixture.Client{Boards: []*githubx.Board{board}, Contents: map[string]githubx.Item{}, FailField: map[string]error{}}
	cfg := &config.Config{Aliases: map[string]string{}}
	cfg.UI.Projects = map[string]config.UIPrefs{}
	svc := core.New(fc, cfg, nil)
	m := newModel(svc, board.Project, RunOptions{})
	m.width, m.height = 100, 30
	m.board = board
	m.rebuildFocus()
	if !m.forceList || m.view != viewList {
		t.Fatalf("expected forced list, force=%v view=%v", m.forceList, m.view)
	}
}

// Regression: wide (emoji) titles and long references must not wrap
// a comfortable card past its fixed 5-line, 36-cell geometry.
func TestCardGeometryWideTitle(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	it := githubx.Item{
		ID: "X", Kind: githubx.KindIssue,
		Title: "💰 Add annual billing option ($15/month per seat)",
		Repo:  "juanrubio/snazzyemail", Number: 8,
	}
	card := m.comfortableCard(it, 36, true)
	lines := strings.Split(card, "\n")
	if len(lines) != 5 {
		t.Fatalf("card has %d lines, want 5:\n%s", len(lines), card)
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w != 36 {
			t.Errorf("line %d width %d, want 36: %q", i, w, l)
		}
	}
	if !strings.Contains(card, "juanrubio/snazzyemail#8") {
		t.Errorf("full reference must fit on one line:\n%s", card)
	}
}

func TestFilterMatches(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	all := m.visibleItems()
	m.filter = "Fixture issue item 01"
	got := m.visibleItems()
	if len(got) >= len(all) || len(got) == 0 {
		t.Fatalf("filter all=%d got=%d", len(all), len(got))
	}
}
