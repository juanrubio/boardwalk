package tui

import (
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/juanrubio/gh-boardwalk/internal/theme"
)

func TestSemanticPaneAndFormRendering(t *testing.T) {
	for _, th := range []*theme.Theme{theme.Dark(), theme.Light(), theme.Monochrome()} {
		for _, size := range [][2]int{{60, 15}, {80, 24}, {120, 40}} {
			m, _ := testModel(t, "fixture-user/1")
			m.theme = th
			m.width, m.height = size[0], size[1]
			it, ok := m.focusedItem()
			if !ok {
				t.Fatal("fixture empty")
			}
			m.detail = &it
			detail := m.renderDetail(35, 10)
			if !strings.Contains(ansi.Strip(detail), "Kind:") {
				t.Fatal("detail metadata missing")
			}
			m.startNewDraft()
			m.form.fields[0].setValue("My draft")
			form := m.renderFormModal()
			plain := ansi.Strip(form)
			for _, want := range []string{"New draft item", "▸ Title:", "My draft", "● changed", "Ctrl-S review", "Esc back"} {
				if !strings.Contains(plain, want) {
					t.Fatalf("size %v missing %q: %s", size, want, plain)
				}
			}
			for _, line := range strings.Split(form, "\n") {
				if lipgloss.Width(line) > size[0] {
					t.Fatalf("modal overflow at %v", size)
				}
			}
			if th.Mono {
				for _, s := range []string{detail, form} {
					if strings.Contains(s, ";2;") || strings.Contains(s, ";5;") {
						t.Fatal("monochrome emitted colors")
					}
				}
			} else if !strings.Contains(detail, "\x1b[") || !strings.Contains(form, "\x1b[") {
				t.Fatal("semantic colors absent")
			}
			m.focusField(1)
			if !strings.Contains(ansi.Strip(m.renderForm(48, 12)), "▸ Body:") {
				t.Fatal("field focus lost")
			}
		}
	}
}

func TestMarkdownUsesResolvedTheme(t *testing.T) {
	dark := renderBody("# Heading\n\n**Body**", 40, theme.Dark())
	light := renderBody("# Heading\n\n**Body**", 40, theme.Light())
	mono := renderBody("# Heading\n\n**Body**", 40, theme.Monochrome())
	if dark == light {
		t.Fatal("light markdown retained dark styles")
	}
	if strings.Contains(mono, ";2;") || strings.Contains(mono, ";5;") {
		t.Fatal("monochrome markdown emitted colors")
	}
	for _, v := range []string{dark, light, mono} {
		if !strings.Contains(ansi.Strip(v), "Heading") {
			t.Fatal("markdown content missing")
		}
	}
}

type semanticScreen struct{ *uv.RenderBuffer }

func (semanticScreen) WidthMethod() uv.WidthMethod { return ansi.GraphemeWidth }

func TestFocusedFormAndProjectSelectionCells(t *testing.T) {
	for _, th := range []*theme.Theme{theme.Dark(), theme.Light()} {
		m, fc := testModel(t, "fixture-user/1")
		m.theme = th
		m.startNewDraft()
		content := m.renderForm(48, 12)
		screen := uv.NewRenderBuffer(48, 12)
		uv.NewStyledString(content).Draw(semanticScreen{screen}, uv.Rect(0, 0, 48, 12))
		cell := screen.CellAt(0, 2)
		same := func(a, b color.Color) bool {
			if a == nil || b == nil {
				return false
			}
			r, g, bl, _ := a.RGBA()
			rr, gg, bb, _ := b.RGBA()
			return r == rr && g == gg && bl == bb
		}
		if cell == nil || !same(cell.Style.Fg, th.Tokens.Accent) {
			t.Fatal("focused field marker lacks accent")
		}
		projects, err := fc.ListProjects(t.Context(), "")
		if err != nil {
			t.Fatal(err)
		}
		m.boards = projects
		m.boardIdx = 0
		m.boards[0].Title = strings.Repeat("long title ", 20)
		picker := m.renderBoardPicker(80)
		lines := strings.Split(ansi.Strip(picker), "\n")
		selectedRow := -1
		for y, line := range lines {
			if strings.Contains(line, "▸") {
				selectedRow = y
			}
		}
		if selectedRow < 0 || !strings.Contains(lines[selectedRow], "…") || !strings.Contains(ansi.Strip(picker), "[current]") {
			t.Fatal("project selection/current missing")
		}
		screen = uv.NewRenderBuffer(56, len(lines))
		uv.NewStyledString(picker).Draw(semanticScreen{screen}, uv.Rect(0, 0, 56, len(lines)))
		border := screen.CellAt(0, 0)
		if border == nil || !same(border.Style.Fg, th.Tokens.Accent) {
			t.Fatal("project border lacks accent")
		}
		for _, x := range []int{2, 51} {
			cell = screen.CellAt(x, selectedRow)
			if cell == nil || !same(cell.Style.Bg, th.Tokens.SelectionBG) {
				t.Fatalf("selected project row not filled at %d", x)
			}
		}
	}
}
