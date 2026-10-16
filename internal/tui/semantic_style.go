package tui

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
)

func (m *model) fieldOptionColor(field, value string) string {
	if m.board != nil {
		for _, f := range m.board.Fields {
			if f.Name == field {
				for _, o := range f.Options {
					if o.Name == value {
						return o.Color
					}
				}
			}
		}
	}
	return "GRAY"
}

func (m *model) fieldLabel(text string, active bool) string {
	if active {
		return m.theme.KeyHint.Render(text)
	}
	return m.theme.Meta.Render(text)
}

func (m *model) styleInput(ff *formField) {
	// Components retain cursor/editing state; only their semantic styles change.
	ti := ff.input.Styles()
	ti.Focused.Text = lipgloss.NewStyle().Foreground(m.theme.Tokens.FG)
	ti.Blurred.Text = ti.Focused.Text
	ti.Focused.Prompt = m.theme.KeyHint
	ti.Blurred.Prompt = m.theme.Meta
	ti.Focused.Placeholder = m.theme.Meta
	ti.Blurred.Placeholder = m.theme.Meta
	ti.Focused.Suggestion = m.theme.Meta
	ti.Blurred.Suggestion = m.theme.Meta
	ti.Cursor.Color = m.theme.Tokens.Accent
	ff.input.SetStyles(ti)
}

func (m *model) formFooter(text string, w int) string {
	// Break at natural separators while preserving complete keyboard hints.
	var rows []string
	current := ""
	for _, part := range strings.Split(text, " · ") {
		candidate := part
		if current != "" {
			candidate = current + " · " + part
		}
		if len([]rune(candidate)) > w && current != "" {
			rows = append(rows, m.theme.Meta.Render(current))
			current = part
		} else {
			current = candidate
		}
	}
	if current != "" {
		rows = append(rows, m.theme.Meta.Render(current))
	}
	return strings.Join(rows, "\n")
}

func (m *model) styleArea(ff *formField) {
	styles := ff.area.Styles()
	base := lipgloss.NewStyle().Foreground(m.theme.Tokens.FG)
	state := textarea.StyleState{Base: base, Text: base, LineNumber: m.theme.Meta, CursorLineNumber: m.theme.Meta, CursorLine: base, EndOfBuffer: m.theme.Meta, Placeholder: m.theme.Meta, Prompt: m.theme.KeyHint}
	styles.Focused = state
	styles.Blurred = state
	styles.Cursor.Color = m.theme.Tokens.Accent
	ff.area.SetStyles(styles)
}
