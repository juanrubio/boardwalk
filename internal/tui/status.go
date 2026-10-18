package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type statusTone int

const (
	statusInfo statusTone = iota
	statusSuccess
	statusWarning
	statusError
	statusPending
)

// setFooter attaches severity at the originating action, never by parsing a
// user-supplied message. Legacy assignments safely fall back to information.
func (m *model) setFooter(tone statusTone, text string) {
	m.footer = text
	m.footerAnnotated = text
	m.footerTone = tone
}

func (m *model) statusStyle(tone statusTone) lipgloss.Style {
	switch tone {
	case statusError:
		return m.theme.Error
	case statusSuccess:
		return m.theme.Success
	case statusWarning:
		return m.theme.Meta.Foreground(m.theme.Tokens.Warning)
	case statusPending:
		return m.theme.KeyHint
	default:
		return m.theme.Meta.Foreground(m.theme.Tokens.Accent)
	}
}

func (m *model) keyHints(separator string, pairs ...string) string {
	parts := make([]string, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, m.theme.KeyHint.Render(pairs[i])+m.theme.KeyLabel.Render(" "+pairs[i+1]))
	}
	return strings.Join(parts, m.theme.KeyLabel.Render(separator))
}
func (m *model) plainHint(text string) string { return m.theme.KeyLabel.Render(text) }
func (m *model) renderFooter(w int) string {
	hint := m.footerHint()
	message := m.footer
	tone := statusInfo
	if message == m.footerAnnotated {
		tone = m.footerTone
	}
	if m.quitting && m.pendingWrites() > 0 {
		message = fmt.Sprintf("saving %d change…", m.pendingWrites())
		tone = statusPending
	}
	if message != "" {
		message = strings.NewReplacer("\n", " ", "\r", " ").Replace(message)
		hint = m.statusStyle(tone).Render(message) + m.plainHint(" · ") + hint
	}
	return ansi.Truncate(hint, max(0, w), "…")
}

func (m *model) footerHint() string {
	keys := m.keyHints
	switch m.overlay {
	case overlayQuit:
		return keys(" · ", "←→/Tab", "choose", "Enter", "confirm", "Esc/n", "cancel", "y", "quit")
	case overlayHelp:
		return keys(" · ", "?", "close help", "Esc", "back")
	case overlayArchive:
		return keys(" · ", "y", "archive (not optimistic)") + m.plainHint(" · any other key cancels")
	case overlayDelete:
		if m.deleteStage == 1 {
			return keys(" · ", "y", "again to permanently delete") + m.plainHint(" · any other key cancels")
		}
		return keys(" · ", "y", "confirm deletion") + m.plainHint(" · any other key cancels")
	case overlayGroup:
		return keys(" · ", "↑↓", "pick grouping", "Enter", "apply", "Esc", "cancel")
	case overlayBoards:
		return keys(" · ", "↑↓", "pick board") + m.plainHint(" · type to filter · ") + keys(" · ", "Enter", "switch", "Esc", "cancel")
	case overlaySearch:
		return keys(" · ", "Enter", "keep filter", "Esc", "clear filter")
	case overlayAdd:
		return keys(" · ", "Enter", "add", "Esc", "cancel")
	case overlayConvert:
		return keys(" · ", "Enter", "convert draft to issue", "Esc", "cancel")
	case overlayDiscard:
		return keys(" · ", "y", "discard changes") + m.plainHint(" · any other key keep editing")
	case overlayQuitDirty:
		return keys(" · ", "y", "quit and discard") + m.plainHint(" · any other key stay")
	}
	switch m.pane {
	case paneSummary:
		if m.summary != nil && m.summary.sent && anyFailed(m.summary.results) {
			return keys(" · ", "Enter", "retry failed", "e", "back to form", "Esc", "back (asks before discarding)")
		}
		if m.savingMsg != "" {
			return m.statusStyle(statusPending).Render(m.savingMsg)
		}
		return keys(" · ", "Enter", "confirm & save", "e", "back to form", "Esc", "back")
	case paneEdit, paneNewDraft:
		if m.form != nil && m.form.picker {
			return keys(" · ", "↑↓", "choose", "Enter", "select", "Esc", "cancel picker")
		}
		hint := m.plainHint("Type to edit · ") + keys(" · ", "←→", "cursor", "Tab/↑↓", "field", "Enter", "pick", "Ctrl-E", "$EDITOR", "Ctrl-S", "review & save", "Esc", "back")
		if m.form != nil && m.form.dirtyCount() > 0 {
			hint += m.plainHint(fmt.Sprintf(" · %d changes · ", m.form.dirtyCount())) + keys("", "Ctrl-S", "review & save")
		}
		return hint
	case paneDetail:
		if m.paneFocus {
			return keys(" · ", "↑↓/jk", "scroll", "Tab", "board", "Esc", "close pane")
		}
		return keys(" · ", "↑↓/jk", "board", "Tab", "pane", "Enter/Esc", "close pane", "H/L", "move", "e", "edit", "x", "archive")
	}
	if m.stale {
		return m.statusStyle(statusWarning).Render("Showing snapshot — moves wait for live data") + m.plainHint(" · ") + keys(" · ", "↑↓/jk", "card", "←→/hl", "column", "Enter", "detail", "q", "quit")
	}
	if m.view == viewList {
		return keys(" · ", "↑↓", "row", "Enter", "detail", "H/L", "move", "e", "edit", "v", "kanban", "/", "filter", "q", "quit")
	}
	return keys(" · ", "↑↓/jk", "card", "←→/hl", "column", "H/L", "move", "Enter", "detail", "n", "draft", "a", "add", "/", "filter", "d", "density", "P", "boards", "q", "quit")
}
