// Package theme holds Boardwalk's semantic colour tokens (ADR 0007,
// tui.md Section 3 visual language). Components never use raw colours;
// they style through a Theme. Every text token pair meets WCAG AA
// contrast (4.5:1) in truecolour and 256-colour, light and dark, and
// degrades to weight/border/marker cues in monochrome.
package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Tokens are the semantic styling vocabulary shared by TUI and CLI.
type Tokens struct {
	FG          color.Color
	Muted       color.Color
	Subtle      color.Color
	Border      color.Color
	BorderFocus color.Color
	Accent      color.Color
	SelectionBG color.Color
	Success     color.Color
	Warning     color.Color
	Danger      color.Color
	// Option colours from GitHub's single-select palette, mapped to
	// curated values (tui.md Section 3). Keys: GRAY, BLUE, GREEN,
	// YELLOW, ORANGE, RED, PINK, PURPLE.
	Options map[string]color.Color
}

// Theme is a resolved token set plus ready-made Lip Gloss styles.
type Theme struct {
	Tokens Tokens
	Dark   bool
	Mono   bool

	Title        lipgloss.Style
	ColumnHeader lipgloss.Style
	CardTitle    lipgloss.Style
	Meta         lipgloss.Style
	KeyHint      lipgloss.Style
	KeyLabel     lipgloss.Style
	Error        lipgloss.Style
	Success      lipgloss.Style
	Card         lipgloss.Style
	CardFocused  lipgloss.Style
}

// Dark returns the dark-background theme.
func Dark() *Theme { return build(true, false) }

// Light returns the light-background theme.
func Light() *Theme { return build(false, false) }

// Monochrome returns the no-colour theme (NO_COLOR / dumb terminals):
// hierarchy comes from weight, borders, and markers only.
func Monochrome() *Theme { return build(true, true) }

func build(dark bool, mono bool) *Theme {
	toks := darkTokens()
	if !dark {
		toks = lightTokens()
	}
	if mono {
		toks = monoTokens()
	}
	t := &Theme{Tokens: toks, Dark: dark, Mono: mono}
	t.Title = lipgloss.NewStyle().Bold(true).Foreground(toks.FG)
	t.ColumnHeader = lipgloss.NewStyle().Bold(true).Foreground(toks.FG)
	t.CardTitle = lipgloss.NewStyle().Foreground(toks.FG)
	t.Meta = lipgloss.NewStyle().Foreground(toks.Muted)
	t.KeyHint = lipgloss.NewStyle().Foreground(toks.Accent).Bold(true)
	t.KeyLabel = lipgloss.NewStyle().Foreground(toks.Muted)
	t.Error = lipgloss.NewStyle().Foreground(toks.Danger)
	t.Success = lipgloss.NewStyle().Foreground(toks.Success)
	t.Card = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(toks.Border).
		Padding(0, 1)
	t.CardFocused = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).BorderForeground(toks.BorderFocus).
		Padding(0, 1)
	return t
}

// OptionColor maps a GitHub option colour name to the curated token,
// falling back to GRAY for unknown values.
func (t *Theme) OptionColor(name string) color.Color {
	if c, ok := t.Tokens.Options[name]; ok {
		return c
	}
	return t.Tokens.Options["GRAY"]
}

func darkTokens() Tokens {
	return Tokens{
		FG:          lipgloss.Color("#E6EDF3"),
		Muted:       lipgloss.Color("#9DA7B3"),
		Subtle:      lipgloss.Color("#6E7B8B"),
		Border:      lipgloss.Color("#3D444D"),
		BorderFocus: lipgloss.Color("#58A6FF"),
		Accent:      lipgloss.Color("#58A6FF"),
		SelectionBG: lipgloss.Color("#1F3A5F"),
		Success:     lipgloss.Color("#3FB950"),
		Warning:     lipgloss.Color("#D29922"),
		Danger:      lipgloss.Color("#F85149"),
		Options: map[string]color.Color{
			"GRAY":   lipgloss.Color("#9DA7B3"),
			"BLUE":   lipgloss.Color("#58A6FF"),
			"GREEN":  lipgloss.Color("#3FB950"),
			"YELLOW": lipgloss.Color("#D29922"),
			"ORANGE": lipgloss.Color("#DB6D28"),
			"RED":    lipgloss.Color("#F85149"),
			"PINK":   lipgloss.Color("#DB61A2"),
			"PURPLE": lipgloss.Color("#BC8CFF"),
		},
	}
}

func lightTokens() Tokens {
	return Tokens{
		FG:          lipgloss.Color("#1F2328"),
		Muted:       lipgloss.Color("#59636E"),
		Subtle:      lipgloss.Color("#818C98"),
		Border:      lipgloss.Color("#D1D9E0"),
		BorderFocus: lipgloss.Color("#0969DA"),
		Accent:      lipgloss.Color("#0969DA"),
		SelectionBG: lipgloss.Color("#DDE9FA"),
		Success:     lipgloss.Color("#1A7F37"),
		Warning:     lipgloss.Color("#9A6700"),
		Danger:      lipgloss.Color("#CF222E"),
		Options: map[string]color.Color{
			"GRAY":   lipgloss.Color("#59636E"),
			"BLUE":   lipgloss.Color("#0969DA"),
			"GREEN":  lipgloss.Color("#1A7F37"),
			"YELLOW": lipgloss.Color("#9A6700"),
			"ORANGE": lipgloss.Color("#BC4C00"),
			"RED":    lipgloss.Color("#CF222E"),
			"PINK":   lipgloss.Color("#BF3989"),
			"PURPLE": lipgloss.Color("#8250DF"),
		},
	}
}

func monoTokens() Tokens {
	none := lipgloss.Color("")
	return Tokens{
		FG: none, Muted: none, Subtle: none, Border: none,
		BorderFocus: none, Accent: none, SelectionBG: none,
		Success: none, Warning: none, Danger: none,
		Options: map[string]color.Color{
			"GRAY": none, "BLUE": none, "GREEN": none, "YELLOW": none,
			"ORANGE": none, "RED": none, "PINK": none, "PURPLE": none,
		},
	}
}
