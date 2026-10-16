package theme

import (
	"image/color"
	"math"
	"strconv"
	"strings"
	"testing"
)

// WCAG AA contrast (tui.md §7): every text token pair used meets 4.5:1
// in truecolour light and dark themes. Backgrounds are the canonical
// terminal backgrounds the themes are designed for.
func TestWCAGAAContrast(t *testing.T) {
	cases := []struct {
		name string
		th   *Theme
		bg   string
	}{
		{"dark", Dark(), "#0D1117"},
		{"light", Light(), "#FFFFFF"},
	}
	for _, tc := range cases {
		bg := parseHex(t, tc.bg)
		toks := map[string]color.Color{
			"FG": tc.th.Tokens.FG, "Muted": tc.th.Tokens.Muted, "Accent": tc.th.Tokens.Accent,
			"Success": tc.th.Tokens.Success, "Warning": tc.th.Tokens.Warning, "Danger": tc.th.Tokens.Danger,
			"BorderFocus": tc.th.Tokens.BorderFocus,
		}
		for _, o := range []string{"GRAY", "BLUE", "GREEN", "YELLOW", "ORANGE", "RED", "PINK", "PURPLE"} {
			toks["Option_"+o] = tc.th.Tokens.Options[o]
		}
		for name, fg := range toks {
			ratio := contrastRatio(parseColor(t, fg), bg)
			if ratio < 4.5 {
				t.Errorf("%s %s contrast %.2f < 4.5", tc.name, name, ratio)
			}
		}
		// (A5) FG on every derived header band, and Meta on the
		// focused card's selection background, must also meet AA.
		for _, o := range []string{"GRAY", "BLUE", "GREEN", "YELLOW", "ORANGE", "RED", "PINK", "PURPLE"} {
			band := parseColor(t, tc.th.HeaderBandBG(o))
			if band == bg {
				t.Errorf("%s %s header band was not tinted (equals terminal bg)", tc.name, o)
			}
			if ratio := contrastRatio(parseColor(t, tc.th.Tokens.FG), band); ratio < 4.5 {
				t.Errorf("%s FG on %s header band contrast %.2f < 4.5", tc.name, o, ratio)
			}
		}
		sel := parseColor(t, tc.th.Tokens.SelectionBG)
		if ratio := contrastRatio(parseColor(t, tc.th.Tokens.Muted), sel); ratio < 4.5 {
			t.Errorf("%s Meta on SelectionBG contrast %.2f < 4.5", tc.name, ratio)
		}
	}
}

func parseHex(t *testing.T, s string) [3]float64 {
	t.Helper()
	s = strings.TrimPrefix(s, "#")
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		t.Fatal(err)
	}
	return [3]float64{float64(n>>16&0xFF) / 255, float64(n>>8&0xFF) / 255, float64(n&0xFF) / 255}
}

func parseColor(t *testing.T, c color.Color) [3]float64 {
	t.Helper()
	// Tokens are lipgloss.Color hex strings.
	type stringer interface{ String() string }
	if s, ok := c.(stringer); ok {
		if strings.HasPrefix(s.String(), "#") {
			return parseHex(t, s.String())
		}
	}
	r, g, b, _ := c.RGBA()
	return [3]float64{float64(r>>8) / 255, float64(g>>8) / 255, float64(b>>8) / 255}
}

func luminance(c [3]float64) float64 {
	f := func(v float64) float64 {
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*f(c[0]) + 0.7152*f(c[1]) + 0.0722*f(c[2])
}

func contrastRatio(a, b [3]float64) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func TestRaisedAndSelectedTextContrast(t *testing.T) {
	for _, th := range []*Theme{Dark(), Light()} {
		for name, fg := range map[string]color.Color{"text": th.Tokens.FG, "muted": th.Tokens.Muted, "accent": th.Tokens.Accent, "warning": th.Tokens.Warning, "danger": th.Tokens.Danger} {
			if ratio := contrastRatio(parseColor(t, fg), parseColor(t, th.Tokens.Surface)); ratio < 4.5 {
				t.Errorf("dark=%v surface %s contrast %.2f", th.Dark, name, ratio)
			}
		}
		for name, fg := range th.Tokens.Options {
			if ratio := contrastRatio(parseColor(t, fg), parseColor(t, th.Tokens.Surface)); ratio < 4.5 {
				t.Errorf("dark=%v surface option %s contrast %.2f", th.Dark, name, ratio)
			}
		}
		if ratio := contrastRatio(parseColor(t, th.Tokens.FG), parseColor(t, th.Tokens.SelectionBG)); ratio < 4.5 {
			t.Errorf("selected text contrast %.2f", ratio)
		}
	}
}
