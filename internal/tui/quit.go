package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m *model) requestQuit() {
	if m.form != nil && m.form.dirtyCount() > 0 {
		m.overlay = overlayQuitDirty
		return
	}
	m.overlay = overlayQuit
	m.quitChoice = false
}

func (m *model) onQuitKey(s string) (tea.Model, tea.Cmd) {
	switch s {
	case "esc", "n":
		m.overlay = overlayNone
	case "left", "right", "up", "down", "tab", "shift+tab":
		m.quitChoice = !m.quitChoice
	case "y":
		return m.confirmQuit()
	case "enter":
		if m.quitChoice {
			return m.confirmQuit()
		}
		m.overlay = overlayNone
	}
	return m, nil
}

func (m *model) confirmQuit() (tea.Model, tea.Cmd) {
	m.overlay = overlayNone
	if m.pendingWrites() > 0 {
		m.quitting = true
		m.setFooter(statusPending, fmt.Sprintf("saving %d change…", m.pendingWrites()))
		return m, nil
	}
	return m, tea.Quit
}

func (m *model) quitButtons() string {
	cancel, quit := "  Cancel  ", "  Quit  "
	if m.quitChoice {
		quit = m.quitButton("› Quit", 8, true)
	} else {
		cancel = m.quitButton("› Cancel", 10, false)
	}
	return cancel + "   " + quit
}

func (m *model) renderQuitModal(w int) string {
	width := min(46, w-6)
	lines := []string{m.theme.KeyHint.Render("Quit Boardwalk?"), "", m.quitButtons(), "", m.keyHints(" · ", "←→/Tab", "choose", "Enter", "confirm", "Esc", "cancel")}
	if m.pendingWrites() > 0 {
		lines = append(lines, m.theme.Meta.Render("Pending writes finish before exit."))
	}
	content := strings.Join(lines, "\n")
	return m.theme.Surface(m.theme.Card.BorderForeground(m.theme.Tokens.BorderFocus).Width(width).Render(content))
}

func (m *model) renderCompactQuit(w, h int) string {
	lines := []string{m.theme.KeyHint.Render("Quit Boardwalk?"), m.quitButtons(), m.keyHints(" · ", "y", "quit", "n/Esc", "stay"), m.theme.Meta.Render("Ctrl-C exits")}
	if h == 1 {
		lines = []string{m.theme.Title.Render("Quit? ") + m.keyHints(" / ", "y", "quit", "n", "stay")}
	}
	out := make([]string, 0, h)
	for _, line := range lines {
		if len(out) >= h {
			break
		}
		out = append(out, truncateANSI(line, w))
	}
	for len(out) < h {
		out = append(out, "")
	}
	return strings.Join(out, "\n")
}

func (m *model) renderCompactQuitDirty(w, h int) string {
	lines := []string{m.theme.Title.Render("Quit and discard changes?"), m.keyHints(" · ", "y", "discard and quit", "n/Esc", "stay")}
	out := make([]string, 0, h)
	for _, line := range lines {
		if len(out) >= h {
			break
		}
		out = append(out, truncateANSI(line, w))
	}
	for len(out) < h {
		out = append(out, "")
	}
	return strings.Join(out, "\n")
}

func (m *model) quitButton(text string, width int, destructive bool) string {
	style := m.theme.KeyHint.Width(width)
	if !m.theme.Mono {
		fill := m.theme.Tokens.Accent
		if destructive {
			fill = m.theme.Tokens.Warning
		}
		style = style.Foreground(m.theme.Tokens.BG).Background(fill)
	}
	return style.Render(text)
}
