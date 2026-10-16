package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juanrubio/gh-boardwalk/internal/theme"
)

func isQuitCommand(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestQuitConfirmationCancelAndSelection(t *testing.T) {
	for _, cancel := range []string{"enter", "esc", "n"} {
		m, _ := testModel(t, "fixture-user/1")
		m.filter = "fixture"
		before := m.filter
		_, cmd := m.onKey(key("q"))
		if isQuitCommand(cmd) || m.overlay != overlayQuit || m.quitChoice || m.quitting {
			t.Fatal("q did not open Cancel-default confirmation")
		}
		_, cmd = m.onKey(key("q"))
		if isQuitCommand(cmd) || m.overlay != overlayQuit {
			t.Fatal("repeated q bypassed confirmation")
		}
		_, cmd = m.onKey(key(cancel))
		if isQuitCommand(cmd) || m.overlay != overlayNone || m.quitting || m.filter != before {
			t.Fatal("cancel changed state or armed quitting")
		}
	}
	for _, choice := range []string{"y", "tab", "right"} {
		m, _ := testModel(t, "fixture-user/1")
		send(m, "q")
		var cmd tea.Cmd
		if choice == "y" {
			_, cmd = m.onKey(key("y"))
		} else {
			code := tea.KeyTab
			if choice == "right" {
				code = tea.KeyRight
			}
			m.onKey(tea.KeyPressMsg{Code: code})
			_, cmd = m.onKey(key("enter"))
		}
		if !isQuitCommand(cmd) || m.overlay != overlayNone {
			t.Fatal("explicit confirm did not quit")
		}
	}
}

func seedPending(m *model, id string) {
	m.pending[id] = &pendingMove{itemID: id, toName: "Done", confirmed: "Todo", inFlight: true}
}

func TestQuitPendingCompletionRequiresConfirmationAndFailuresStayVisible(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	seedPending(m, "one")
	send(m, "q")
	_, cmd := m.onMoveResult(moveResultMsg{itemID: "one", toName: "Done"})
	if isQuitCommand(cmd) || m.quitting || m.overlay != overlayQuit {
		t.Fatal("pending completion quit before confirmation")
	}
	send(m, "esc")
	seedPending(m, "one")
	send(m, "q")
	send(m, "n")
	_, cmd = m.onMoveResult(moveResultMsg{itemID: "one", toName: "Done"})
	if isQuitCommand(cmd) || m.quitting {
		t.Fatal("cancel armed pending exit")
	}
	seedPending(m, "one")
	seedPending(m, "two")
	send(m, "q")
	_, cmd = m.onKey(key("y"))
	if isQuitCommand(cmd) || !m.quitting {
		t.Fatal("confirmed quit failed to wait")
	}
	_, cmd = m.onMoveResult(moveResultMsg{itemID: "one", toName: "Done"})
	if isQuitCommand(cmd) {
		t.Fatal("quit while another write pending")
	}
	_, cmd = m.onMoveResult(moveResultMsg{itemID: "two", toName: "Done"})
	if !isQuitCommand(cmd) {
		t.Fatal("confirmed quit did not finish when writes settled")
	}
	m, _ = testModel(t, "fixture-user/1")
	seedPending(m, "one")
	seedPending(m, "two")
	send(m, "q")
	send(m, "y")
	_, cmd = m.onMoveResult(moveResultMsg{itemID: "one", err: fmt.Errorf("synthetic failure")})
	if isQuitCommand(cmd) || m.quitting || !strings.Contains(m.footer, "Move failed") {
		t.Fatal("failed write did not cancel pending quit")
	}
	_, cmd = m.onMoveResult(moveResultMsg{itemID: "two", toName: "Done"})
	if isQuitCommand(cmd) {
		t.Fatal("later success hid earlier failed write")
	}
}

func TestQuitInputNestedDirtyAndTinyTerminal(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	m.pane = paneDetail
	m.paneFocus = true
	send(m, "q")
	if m.pane != paneNone || m.overlay != overlayNone {
		t.Fatal("nested q no longer closes detail")
	}
	for _, overlay := range []overlayKind{overlaySearch, overlayAdd, overlayConvert, overlayBoards} {
		m.overlay = overlay
		send(m, "q")
		if m.overlay != overlay {
			t.Fatal("input q opened quit")
		}
	}
	m.overlay = overlayNone
	m.startNewDraft()
	field := &m.form.fields[m.form.idx]
	before := field.value()
	send(m, "q")
	if m.overlay != overlayNone || m.form.fields[m.form.idx].value() != before+"q" {
		t.Fatal("draft q did not remain text input")
	}
	m.width, m.height = 40, 10
	_, cmd := m.onKey(key("q"))
	if isQuitCommand(cmd) || m.overlay != overlayQuitDirty || !strings.Contains(ansi.Strip(m.View().Content), "discard changes") {
		t.Fatal("small terminal bypassed dirty protection")
	}
	m, _ = testModel(t, "fixture-user/1")
	m.width, m.height = 40, 10
	send(m, "q")
	if !strings.Contains(ansi.Strip(m.View().Content), "Quit Boardwalk?") {
		t.Fatal("missing compact confirmation")
	}
	_, cmd = m.onKey(key("enter"))
	if isQuitCommand(cmd) {
		t.Fatal("compact default was destructive")
	}
	_, cmd = m.onKey(key("ctrl+c"))
	if !isQuitCommand(cmd) {
		t.Fatal("Ctrl-C escape changed")
	}
}

func TestColoredFooterAndQuitDimensions(t *testing.T) {
	for _, th := range []*theme.Theme{theme.Dark(), theme.Light(), theme.Monochrome()} {
		m, _ := testModel(t, "fixture-user/1")
		m.theme = th
		plain := ansi.Strip(m.renderFooter(240))
		if !strings.Contains(plain, "H/L move") || !strings.Contains(plain, "q quit") {
			t.Fatal("structured keys changed labels")
		}
		m.setFooter(statusError, "synthetic error q quit")
		footer := m.renderFooter(80)
		if lipgloss.Width(footer) > 80 || strings.Contains(footer, "\n") {
			t.Fatal("footer exceeds one row")
		}
		if !strings.HasPrefix(footer, m.theme.Error.Render("synthetic error q quit")) {
			t.Fatal("arbitrary status text parsed as shortcut keys")
		}
		m.footer = "literal q quit status"
		legacy := m.renderFooter(80)
		if !strings.HasPrefix(legacy, m.statusStyle(statusInfo).Render(m.footer)) {
			t.Fatal("legacy status inherited stale severity")
		}

		if th.Mono && (strings.Contains(footer, "38;") || strings.Contains(footer, "48;")) {
			t.Fatal("monochrome footer gained color")
		}
		for _, size := range [][2]int{{120, 40}, {60, 15}, {40, 10}, {12, 3}} {
			m.width, m.height = size[0], size[1]
			m.overlay = overlayQuit
			m.quitChoice = false
			view := m.View().Content
			if lipgloss.Width(view) > size[0] || lipgloss.Height(view) > size[1] {
				t.Fatalf("quit overlay overflow: %v", size)
			}
		}
	}
}

func TestCtrlCExistingPendingAndDirtyGuards(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	seedPending(m, "one")
	_, cmd := m.onKey(key("ctrl+c"))
	if isQuitCommand(cmd) || !m.forceQuit || m.quitting {
		t.Fatal("first Ctrl-C pending behavior changed")
	}
	_, cmd = m.onKey(key("ctrl+c"))
	if !isQuitCommand(cmd) {
		t.Fatal("second Ctrl-C force escape changed")
	}
	m, _ = testModel(t, "fixture-user/1")
	m.startNewDraft()
	send(m, "q")
	_, cmd = m.onKey(key("ctrl+c"))
	if isQuitCommand(cmd) || m.overlay != overlayQuitDirty {
		t.Fatal("Ctrl-C bypassed dirty discard guard")
	}
	send(m, "n")
	if m.form == nil || m.form.dirtyCount() == 0 {
		t.Fatal("canceling discard lost form edits")
	}
}

func TestFooterTruncationSignalsHiddenHints(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	m.theme = theme.Dark()
	footer := m.renderFooter(80)
	if lipgloss.Width(footer) > 80 || !strings.HasSuffix(ansi.Strip(footer), "…") || strings.Contains(footer, "\n") {
		t.Fatal("truncated footer does not signal hidden hints within one row")
	}
	full := m.renderFooter(300)
	if strings.HasSuffix(ansi.Strip(full), "…") {
		t.Fatal("untruncated footer gained ellipsis")
	}
}
