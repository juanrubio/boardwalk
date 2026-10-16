package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/juanrubio/boardwalk/internal/config"
	"github.com/juanrubio/boardwalk/internal/core"
	"github.com/juanrubio/boardwalk/internal/fixture"
	"github.com/juanrubio/boardwalk/internal/githubx"
	"github.com/juanrubio/boardwalk/internal/theme"
)

// TestProgramEnvKittyREP pins the Kitty compatibility rewrite: early
// Kitty builds (0.16) ignore CSI b (REP), which ultraviolet enables
// from TERM=xterm-kitty alone; every REP-compressed padding run then
// vanishes and card/modal borders shift left. The program environment
// must present xterm-256color for Kitty TERM values (ECH still works
// there) and leave every other terminal untouched.
func TestProgramEnvKittyREP(t *testing.T) {
	cases := []struct {
		name  string
		env   []string
		want  []string
		kitty bool
	}{
		{"kitty", []string{"TERM=xterm-kitty", "HOME=/x"}, []string{"TERM=xterm-256color", "HOME=/x"}, true},
		{"kitty 256color", []string{"TERM=xterm-kitty-256color"}, []string{"TERM=xterm-256color"}, true},
		{"tmux", []string{"TERM=screen-256color"}, []string{"TERM=screen-256color"}, false},
		{"xterm", []string{"TERM=xterm-256color"}, []string{"TERM=xterm-256color"}, false},
		{"no term", []string{"HOME=/x"}, []string{"HOME=/x"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, kitty := programEnv(tc.env)
			if kitty != tc.kitty {
				t.Fatalf("kitty = %v, want %v", kitty, tc.kitty)
			}
			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Fatalf("env = %v, want %v", got, tc.want)
			}
		})
	}
	// The caller's slice must not be mutated.
	env := []string{"TERM=xterm-kitty"}
	programEnv(env)
	if env[0] != "TERM=xterm-kitty" {
		t.Fatalf("input env mutated: %v", env)
	}
}

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

// Regression (A5): the focused card's selection background must cover
// the text itself, not only padding and trailing spaces — the inner
// title/meta styles reset SGR, so each line style must carry the
// background. Every content line needs the bg code at least twice
// (stripe segment + text segment).
func TestFocusedCardBackgroundCoversText(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	m.theme = theme.Dark()
	it := githubx.Item{
		ID: "X", Kind: githubx.KindIssue,
		Title: "Add annual billing option",
		Repo:  "juanrubio/snazzyemail", Number: 8,
	}
	card := m.comfortableCard(it, 36, true, "BLUE")
	lines := strings.Split(card, "\n")
	if len(lines) != 5 {
		t.Fatalf("card has %d lines, want 5:\n%s", len(lines), card)
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w != 36 {
			t.Errorf("line %d width %d, want 36: %q", i, w, l)
		}
	}
	const bg = "48;2;31;58;95" // SelectionBG #1F3A5F
	// The SGR sequence immediately before the text must carry the bg;
	// counting bg codes is not enough (padding alone would pass, and
	// the terminal cell model showed text cells on default bg).
	for i, text := range map[int]string{1: "Add annual billing", 2: "issue juanrubio/snazzyemail#8"} {
		idx := strings.Index(lines[i], text)
		if idx < 0 {
			t.Fatalf("line %d lacks %q: %q", i, text, lines[i])
		}
		esc := strings.LastIndex(lines[i][:idx], "\x1b[")
		if esc < 0 || !strings.Contains(lines[i][esc:idx], bg) {
			t.Errorf("line %d SGR before text lacks selection bg: %q", i, lines[i])
		}
	}
	for i := 1; i <= 3; i++ {
		if n := strings.Count(lines[i], bg); n < 2 {
			t.Errorf("content line %d has selection bg %d times, want >= 2: %q", i, n, lines[i])
		}
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
	card := m.comfortableCard(it, 36, true, "GREEN")
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

func TestDeleteTwoStageModal(t *testing.T) {
	m, fc := testModel(t, "fixture-user/1")
	it, ok := m.focusedItem()
	if !ok {
		t.Fatal("no focus")
	}
	m = send(m, "D")
	if m.overlay != overlayDelete || m.deleteStage != 0 {
		t.Fatalf("D should open delete modal stage 0, got overlay=%v stage=%d", m.overlay, m.deleteStage)
	}
	if v := m.View().Content; !strings.Contains(v, "cannot be undone") {
		t.Fatalf("delete modal missing warning:\n%s", v)
	}
	// First y only arms; the fixture item must still exist.
	m = send(m, "y")
	if m.deleteStage != 1 {
		t.Fatalf("first y should arm stage 1, got %d", m.deleteStage)
	}
	if fc.Board("fixture-user/1") == nil {
		t.Fatal("board gone")
	}
	found := false
	for _, bi := range fc.Board("fixture-user/1").Items {
		if bi.ID == it.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("item deleted after only one y — double confirmation broken")
	}
	// Second y executes.
	mm, cmd := m.Update(key("y"))
	m = mm.(*model)
	if cmd == nil {
		t.Fatal("second y returned no delete command")
	}
	mm, _ = m.Update(cmd())
	m = mm.(*model)
	found = false
	for _, bi := range fc.Board("fixture-user/1").Items {
		if bi.ID == it.ID {
			found = true
		}
	}
	if found {
		t.Fatal("item not deleted after two y presses")
	}
	if _, ok := m.focusedItem(); !ok {
		t.Fatal("no focus after delete")
	}
}

func TestFluidWidths(t *testing.T) {
	mk := func(n int) []column {
		cols := make([]column, n)
		for i := range cols {
			cols[i] = column{label: "C"}
		}
		return cols
	}
	// Owner's case: 5 columns at 180 cells shrink-to-fit, no scrolling.
	widths, ok := fluidWidths(180, mk(5), 36)
	if !ok {
		t.Fatal("180 cells / 5 cols should shrink-to-fit")
	}
	sum := 0
	for _, wd := range widths {
		if wd < 30 || wd > 56 {
			t.Fatalf("width %d outside [30,56]", wd)
		}
		sum += wd
	}
	if sum+4 > 180 {
		t.Fatalf("widths+gutters = %d > 180", sum+4)
	}
	// Cap: 3 columns on a very wide terminal stop at 56.
	widths, ok = fluidWidths(300, mk(3), 36)
	if !ok {
		t.Fatal("cap case should fit")
	}
	for _, wd := range widths {
		if wd != 56 {
			t.Fatalf("width %d, want cap 56", wd)
		}
	}
	// Too narrow for all: 5 columns at 120 cells cannot shrink enough.
	if _, ok := fluidWidths(120, mk(5), 36); ok {
		t.Fatal("120 cells / 5 cols should not fit (window mode)")
	}
}

func TestBoardFitsAllColumnsAt180(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	m.width, m.height = 180, 40
	v := m.View().Content
	for _, name := range []string{"Backlog", "Ready", "In progress", "In review", "Done"} {
		if !strings.Contains(v, name) {
			t.Fatalf("column %q missing at 180 cells:\n%s", name, v)
		}
	}
	if strings.Contains(v, "cols ▸") {
		t.Fatalf("no off-screen hint expected when everything fits:\n%s", v)
	}
}

// Regression (owner screenshot): a bare emoji codepoint (U+1F5A5, no
// variation selector) is counted as 1 cell by wcwidth measurers but
// drawn 2 cells wide by terminals, shifting the card's right border
// on the title line. Normalization must give the card exact geometry.
func TestCardGeometryBareEmojiTitle(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	if got := normalizeEmoji("🖥Server maintenance"); got != "🖥️Server maintenance" {
		t.Fatalf("normalizeEmoji = %q", got)
	}
	if got := normalizeEmoji("plain title"); got != "plain title" {
		t.Fatalf("plain title changed: %q", got)
	}
	it := githubx.Item{
		ID: "E1", Kind: githubx.KindDraft, DatabaseID: 157092128,
		Title: "🖥Server maintenance (move Docker to /home)",
	}
	for _, w := range []int{56, 36, 30} {
		card := m.comfortableCard(it, w, true, "GREEN")
		lines := strings.Split(card, "\n")
		if len(lines) != 5 {
			t.Fatalf("w=%d: card has %d lines:\n%s", w, len(lines), card)
		}
		for i, l := range lines {
			if lw := lipgloss.Width(l); lw != w {
				t.Errorf("w=%d line %d lipgloss width %d: %q", w, i, lw, l)
			}
		}
	}
}

// VTE mode (Ptyxis/GNOME Terminal): text-default emoji count one cell
// there, so displayTitle strips the selector and geometry still holds.
func TestCardGeometryVTEMode(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	m.vte = true
	if got := m.displayTitle("🖥️ Server maintenance"); got != "🖥 Server maintenance" {
		t.Fatalf("VTE displayTitle = %q", got)
	}
	if got := m.displayTitle("✉️ SnazzyEmail"); got != "✉ SnazzyEmail" {
		t.Fatalf("VTE displayTitle = %q", got)
	}
	it := githubx.Item{
		ID: "E2", Kind: githubx.KindDraft, DatabaseID: 157092128,
		Title: "🖥️ Server maintenance (move Docker to /home)",
	}
	card := m.comfortableCard(it, 56, true, "GREEN")
	lines := strings.Split(card, "\n")
	if len(lines) != 5 {
		t.Fatalf("card has %d lines:\n%s", len(lines), card)
	}
	for i, l := range lines {
		if lw := lipgloss.Width(l); lw != 56 {
			t.Errorf("line %d width %d, want 56: %q", i, lw, l)
		}
	}
	if strings.Contains(card, "\uFE0F") {
		t.Errorf("VTE card should not contain VS16:\n%s", card)
	}
}

// A4: the editor is a centred modal with a real cursor — arrow keys
// move the insertion point and typed text lands there.
func TestEditorModalCursorAndArrows(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	m = send(m, "e")
	if m.form == nil {
		t.Fatal("edit did not open")
	}
	v := m.View().Content
	if !strings.Contains(v, "╭") {
		t.Fatalf("editor modal border missing:\n%s", v)
	}
	centered := false
	for _, line := range strings.Split(v, "\n") {
		if strings.Contains(line, "╭") && strings.HasPrefix(line, "    ") {
			centered = true
		}
	}
	if !centered {
		t.Fatalf("editor modal is not centred:\n%s", v)
	}
	// Title field is focused with the cursor at the end. Left then
	// type: the new character must land before the last character.
	title := m.form.fields[0].value()
	mm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	m = mm.(*model)
	mm, _ = m.Update(tea.KeyPressMsg{Code: 'X', Text: "X"})
	m = mm.(*model)
	got := m.form.fields[0].value()
	want := title[:len(title)-1] + "X" + title[len(title)-1:]
	if got != want {
		t.Fatalf("cursor insert: got %q want %q (title %q)", got, want, title)
	}
	if m.form.dirtyCount() == 0 {
		t.Fatal("edit via cursor did not mark dirty")
	}
	// Body: Tab to it, type two lines, Up returns to the first line.
	mm, _ = m.Update(key("tab"))
	m = mm.(*model)
	if m.form.idx != 1 {
		t.Fatalf("Tab should focus Body (idx 1), got %d", m.form.idx)
	}
	for _, r := range "first" {
		mm, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = mm.(*model)
	}
	mm, _ = m.Update(key("enter")) // newline in textarea
	m = mm.(*model)
	for _, r := range "second" {
		mm, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = mm.(*model)
	}
	mm, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m = mm.(*model)
	mm, _ = m.Update(tea.KeyPressMsg{Code: '!', Text: "!"})
	m = mm.(*model)
	body := m.form.fields[1].value()
	if !strings.Contains(body, "first!") || !strings.Contains(body, "second") {
		t.Fatalf("body arrow navigation failed, body:\n%s", body)
	}
}

// (A8) Enter with the detail pane open and focus on the board
// dismisses the pane; Tab into the pane shows a focus marker.
func TestEnterDismissesDetailPane(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	m = send(m, "enter")
	if m.pane != paneDetail {
		t.Fatalf("Enter should open detail, pane=%v", m.pane)
	}
	m = send(m, "enter")
	if m.pane != paneNone {
		t.Fatalf("Enter with pane open (board focus) should dismiss it, pane=%v", m.pane)
	}
	// Reopen, Tab into the pane: title carries the focus marker.
	m = send(m, "enter")
	m = send(m, "tab")
	if !m.paneFocus {
		t.Fatal("Tab should focus the pane")
	}
	it, _ := m.focusedItem()
	if v := m.View().Content; !strings.Contains(v, "▸ "+it.Title) {
		t.Fatalf("focused pane should mark its title with ▸:\n%s", v)
	}
}

// Regression: detail scrolling must clamp at the last page. Past
// the end it used to wrap back to the top with an ever-growing
// offset, so Up appeared dead until Tab out/in.
func TestDetailScrollClampsAtEnd(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	m = send(m, "enter")
	var body strings.Builder
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&body, "body line %03d\n", i)
	}
	m.detailBody = body.String()
	m = send(m, "tab") // focus pane
	for i := 0; i < 200; i++ {
		m = send(m, "down")
	}
	out := m.renderDetail(40, 10)
	if strings.Contains(out, "body line 000") {
		t.Fatalf("scroll past end wrapped to the top:\n%s", out)
	}
	if !strings.Contains(out, "body line 099") {
		t.Fatalf("scroll should rest on the last page:\n%s", out)
	}
	// One Up from the end must move immediately (the first render
	// clamped the offset; a single Up steps exactly one line).
	clamped := m.detailScroll
	topBefore := strings.Split(out, "\n")[0]
	m = send(m, "up")
	out = m.renderDetail(40, 10)
	if m.detailScroll != clamped-1 {
		t.Fatalf("one Up should step one line: clamped=%d now=%d", clamped, m.detailScroll)
	}
	if strings.Split(out, "\n")[0] == topBefore {
		t.Fatalf("one Up should change the top line:\n%s", out)
	}
}

// (A10) Columns sort by the Priority field's option order by
// default (no priority last, ties stable); s returns to manual
// order for the session.
func TestPrioritySortDefaultAndToggle(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	pf, ok := m.priorityField()
	if !ok {
		t.Fatal("fixture board should have a Priority field")
	}
	cols := m.columns()
	target := -1
	for i, c := range cols {
		if len(c.items) >= 4 {
			target = i
			break
		}
	}
	if target < 0 {
		t.Fatal("no column with 4+ items")
	}
	// Assign P2, none, P0, P1 across the column in board order.
	assign := []string{"P2", "", "P0", "P1"}
	optID := map[string]string{}
	for _, o := range pf.Options {
		optID[o.Name] = o.ID
	}
	nameOf := map[string]string{}
	for idx, it := range cols[target].items[:4] {
		nameOf[it.ID] = assign[idx]
		for bi := range m.board.Items {
			if m.board.Items[bi].ID == it.ID && assign[idx] != "" {
				m.board.Items[bi].Values = append(m.board.Items[bi].Values, githubx.FieldValue{
					FieldID: pf.ID, FieldName: pf.Name, DataType: pf.DataType,
					OptionID: optID[assign[idx]], OptionName: assign[idx],
				})
			}
		}
	}
	got := m.columns()[target].items
	want := []string{"P0", "P1", "P2", ""}
	for i, w := range want {
		if nameOf[got[i].ID] != w {
			t.Fatalf("sorted position %d = priority %q, want %q (all: %v)", i, nameOf[got[i].ID], w, nameOf)
		}
	}
	m.width = 160 // indicator shows when the header has room
	if !m.sortedByPriority() || !strings.Contains(m.View().Content, "Sorted by Priority") {
		t.Fatal("header should show Sorted by Priority")
	}
	// s → manual order (original sequence), header note gone.
	m = send(m, "s")
	if m.sortedByPriority() {
		t.Fatal("s should disable Priority sort")
	}
	got = m.columns()[target].items
	for i, w := range assign {
		if nameOf[got[i].ID] != w {
			t.Fatalf("manual position %d = priority %q, want %q", i, nameOf[got[i].ID], w)
		}
	}
}

// (A9) With detail open, the board side shows only the focused
// column and the pane takes the remaining width.
func TestDetailPaneLeavesOneColumn(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	m.width, m.height = 160, 40 // wide: old split showed 2+ board columns
	m = send(m, "enter")
	v := m.View().Content
	if !strings.Contains(v, "Backlog (") {
		t.Fatalf("focused column should remain visible:\n%s", v)
	}
	for _, other := range []string{"Ready (", "In Progress ("} {
		if strings.Contains(v, other) {
			t.Fatalf("detail split should hide other columns (%s):\n%s", other, v)
		}
	}
	it, _ := m.focusedItem()
	if !strings.Contains(v, it.Title) {
		t.Fatalf("pane should show the item:\n%s", v)
	}
}

// (A7) C on a draft opens the repo prompt (pre-filled with the
// board's single repo); Enter converts and the card stays an issue.
func TestConvertDraftFlow(t *testing.T) {
	m, _ := testModel(t, "fixture-user/3")
	var draft *githubx.Item
	for i := range m.board.Items {
		if m.board.Items[i].Kind == githubx.KindDraft {
			draft = &m.board.Items[i]
			break
		}
	}
	if draft == nil {
		t.Fatal("no draft in fixture board")
	}
	m.setFocusedByID(draft.ID)
	m = send(m, "C")
	if m.overlay != overlayConvert {
		t.Fatalf("C should open convert prompt, overlay=%v", m.overlay)
	}
	if m.convertText != "fixture/repo" {
		t.Fatalf("convert prompt prefill = %q, want fixture/repo", m.convertText)
	}
	mm, cmd := m.Update(key("enter"))
	m = mm.(*model)
	if cmd == nil {
		t.Fatal("convert should return a command")
	}
	mm, _ = m.Update(cmd())
	m = mm.(*model)
	for _, bi := range m.board.Items {
		if bi.ID == draft.ID {
			if bi.Kind != githubx.KindIssue || bi.Repo != "fixture/repo" {
				t.Fatalf("converted item = kind %s repo %s", bi.Kind, bi.Repo)
			}
			if !strings.HasPrefix(m.footer, "Converted → ") {
				t.Fatalf("footer = %q", m.footer)
			}
			return
		}
	}
	t.Fatal("converted item missing from board")
}

// Regression: a detail response carrying no field values must not
// strip the board item's values (the item would fall out of its
// Status column into "No Status" after merely opening detail).
func TestDetailMsgKeepsBoardValues(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	it, ok := m.focusedItem()
	if !ok {
		t.Fatal("no focus")
	}
	before, _ := it.Value(m.groupField.Name)
	mm, _ := m.Update(detailMsg{item: githubx.Item{ID: it.ID, Kind: it.Kind, Title: it.Title, Body: "body"}})
	m = mm.(*model)
	for _, bi := range m.board.Items {
		if bi.ID == it.ID {
			after, ok := bi.Value(m.groupField.Name)
			if !ok || after.OptionName != before.OptionName {
				t.Fatalf("detail replaced board values: before=%q after=%q ok=%v", before.OptionName, after.OptionName, ok)
			}
		}
	}
}

// (A6) Board switching: picker selection switches in place, [ / ]
// cycle with wrap, the board list survives a switch, late pages for a
// board already left are ignored, and the picker filters by text.
func TestBoardPickerAndCycle(t *testing.T) {
	m, fc := testModel(t, "fixture-user/1")
	projects, err := fc.ListProjects(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	mm, _ := m.Update(boardsMsg{projects: projects})
	m = mm.(*model)
	refs := []string{m.boards[0].Ref(), m.boards[1].Ref(), m.boards[2].Ref()}
	if want := []string{"fixture-org/7", "fixture-user/1", "fixture-user/3"}; strings.Join(refs, ",") != strings.Join(want, ",") {
		t.Fatalf("board order = %v, want %v", refs, want)
	}

	// Picker: opens, marks current, Down+Enter switches.
	m = send(m, "P")
	if m.overlay != overlayBoards {
		t.Fatalf("P should open board picker, overlay=%v", m.overlay)
	}
	if v := m.View().Content; !strings.Contains(v, "Boards") || !strings.Contains(v, "• fixture-user/1") {
		t.Fatalf("picker should mark current board:\n%s", v)
	}
	m = send(m, "down")
	m = send(m, "enter")
	if m.project.Ref() != "fixture-user/3" {
		t.Fatalf("picker switch landed on %s, want fixture-user/3", m.project.Ref())
	}
	if len(m.boards) != 3 {
		t.Fatal("board list must survive a switch")
	}

	// Cycle wraps at both ends.
	m = send(m, "]")
	if m.project.Ref() != "fixture-org/7" {
		t.Fatalf("] from last should wrap to fixture-org/7, got %s", m.project.Ref())
	}
	m = send(m, "[")
	if m.project.Ref() != "fixture-user/3" {
		t.Fatalf("[ from first should wrap to fixture-user/3, got %s", m.project.Ref())
	}

	// Stale page for another board is ignored.
	before := len(m.board.Items)
	mm, _ = m.Update(pageMsg{projectID: "other-board", first: true, items: []githubx.Item{{ID: "zz"}}})
	m = mm.(*model)
	if len(m.board.Items) != before || m.project.Ref() != "fixture-user/3" {
		t.Fatal("stale page for another board leaked into the current board")
	}

	// Picker filter narrows and Enter switches to the match.
	m = send(m, "P")
	for _, r := range "snazzy" {
		m = send(m, string(r))
	}
	if got := m.filteredBoards(); len(got) != 1 || got[0].Ref() != "fixture-user/1" {
		t.Fatalf("filter snazzy = %v", got)
	}
	m = send(m, "enter")
	if m.project.Ref() != "fixture-user/1" {
		t.Fatalf("filtered switch landed on %s, want fixture-user/1", m.project.Ref())
	}
}
