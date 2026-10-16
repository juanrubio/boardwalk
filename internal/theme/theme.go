// Package theme holds Boardwalk's semantic colour tokens (ADR 0007,
// tui.md Section 3 visual language). Components never use raw colours;
// they style through a Theme. Every text token pair meets WCAG AA
// contrast (4.5:1) in truecolour and 256-colour, light and dark, and
// degrades to weight/border/marker cues in monochrome.
package theme

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"charm.land/lipgloss/v2"
)

// Tokens are the semantic styling vocabulary shared by TUI and CLI.
type Tokens struct {
	Surface     color.Color // raised modal surface
	BG          color.Color // canonical terminal background the theme targets
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

	headerMix float64 // option-colour fraction in a column header band (A5)

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
	t := &Theme{Tokens: toks, Dark: dark, Mono: mono, headerMix: 0.30}
	if !dark {
		t.headerMix = 0.18
	}
	if mono {
		t.headerMix = 0
	}
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
		Background(toks.SelectionBG).
		Padding(0, 1)
	return t
}

// HeaderBand is the column-header style (A5): bold FG on a background
// mixed from the column's option colour toward the theme background.
// In monochrome the band is bold only.
func (t *Theme) HeaderBand(optionColor string) lipgloss.Style {
	if t.Mono || t.headerMix == 0 {
		return t.ColumnHeader
	}
	return t.ColumnHeader.Background(mix(t.OptionColor(optionColor), t.Tokens.BG, t.headerMix))
}

// HeaderBandBG exposes the derived band background for contrast tests.
func (t *Theme) HeaderBandBG(optionColor string) color.Color {
	if t.Mono || t.headerMix == 0 {
		return t.Tokens.BG
	}
	return mix(t.OptionColor(optionColor), t.Tokens.BG, t.headerMix)
}

// ColumnBorder is the unfocused card border for a column (A5): the
// neutral border mixed halfway toward the option colour.
func (t *Theme) ColumnBorder(optionColor string) color.Color {
	if t.Mono {
		return t.Tokens.Border
	}
	return mix(t.OptionColor(optionColor), t.Tokens.Border, 0.5)
}

// Stripe is the card accent-stripe style in the column's option colour.
func (t *Theme) Stripe(optionColor string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.OptionColor(optionColor))
}

// mix blends fg over bg with fg fraction t (0..1). A nil colour
// yields bg; callers guard the monochrome theme, whose tokens are
// empty colours, before mixing.
func mix(fg, bg color.Color, t float64) color.Color {
	f, okF := hexRGB(fg)
	b, okB := hexRGB(bg)
	if !okF || !okB {
		return bg
	}
	ch := func(x, y uint64) uint64 {
		return uint64(float64(x)*t + float64(y)*(1-t) + 0.5)
	}
	return lipgloss.Color("#" + hex2(ch(f[0], b[0])) + hex2(ch(f[1], b[1])) + hex2(ch(f[2], b[2])))
}

func hexRGB(c color.Color) ([3]uint64, bool) {
	if c == nil {
		return [3]uint64{}, false
	}
	r, g, b, _ := c.RGBA()
	return [3]uint64{uint64(r >> 8), uint64(g >> 8), uint64(b >> 8)}, true
}

func hex2(v uint64) string {
	const digits = "0123456789ABCDEF"
	return string([]byte{digits[v>>4&0xF], digits[v&0xF]})
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
		Surface:     lipgloss.Color("#161B22"),
		BG:          lipgloss.Color("#0D1117"),
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
		Surface:     lipgloss.Color("#F6F8FA"),
		BG:          lipgloss.Color("#FFFFFF"),
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
		Surface: none, BG: none, FG: none, Muted: none, Subtle: none, Border: none,
		BorderFocus: none, Accent: none, SelectionBG: none,
		Success: none, Warning: none, Danger: none,
		Options: map[string]color.Color{
			"GRAY": none, "BLUE": none, "GREEN": none, "YELLOW": none,
			"ORANGE": none, "RED": none, "PINK": none, "PURPLE": none,
		},
	}
}

// Surface fills a raised modal without allowing nested SGR resets to punch
// holes in its surface or leak its background into centering padding.
func (t *Theme) Surface(content string) string {
	return t.fill(content, t.Tokens.Surface)
}

func (t *Theme) fill(content string, bg color.Color) string {
	if t.Mono {
		return content
	}
	r, g, b, _ := bg.RGBA()
	fr, fg, fb, _ := t.Tokens.FG.RGBA()
	restore := fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm", fr>>8, fg>>8, fb>>8, r>>8, g>>8, b>>8)
	rows := strings.Split(content, "\n")
	for i, line := range rows {
		for _, reset := range []string{"\x1b[m", "\x1b[0m", "\x1b[49m"} {
			line = strings.ReplaceAll(line, reset, reset+restore)
		}
		rows[i] = restore + line + "\x1b[0m"
	}
	return strings.Join(rows, "\n")
}

// Selected renders a complete row; a caller-provided marker remains visible
// in monochrome. Selected values use the theme foreground for AA contrast.
func (t *Theme) Selected(content string, width int) string {
	plain := ansi.Truncate(ansi.Strip(content), max(0, width), "…")
	plain += strings.Repeat(" ", max(0, width-ansi.StringWidth(plain)))
	style := lipgloss.NewStyle().Bold(true)
	if !t.Mono {
		style = style.Foreground(t.Tokens.FG).Background(t.Tokens.SelectionBG)
	}
	return style.Render(plain)
}
