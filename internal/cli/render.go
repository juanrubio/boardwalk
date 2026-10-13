package cli

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"

	"github.com/juanrubio/gh-boardwalk/internal/githubx"
	"github.com/juanrubio/gh-boardwalk/internal/theme"
)

// colorMode resolves cli.md Section 5 colour behaviour:
// --color flag > NO_COLOR / TERM=dumb > CLICOLOR_FORCE > TTY detection.
func colorMode(opts *Options) (enabled bool) {
	switch opts.Color {
	case "always":
		return true
	case "never":
		return false
	}
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	if v := os.Getenv("CLICOLOR_FORCE"); v != "" && v != "0" {
		return true
	}
	return isTTY(opts.Out)
}

// themeFor returns the CLI theme consistent with the colour mode.
func themeFor(opts *Options) *theme.Theme {
	if !colorMode(opts) {
		return theme.Monochrome()
	}
	return theme.Dark()
}

// termWidth returns the stdout terminal width, or 80 when unknown.
func termWidth(opts *Options) int {
	if f, ok := opts.Out.(*os.File); ok {
		if w, _, err := term.GetSize(int(f.Fd())); err == nil && w > 0 {
			return w
		}
	}
	if v := os.Getenv("COLUMNS"); v != "" {
		var w int
		if _, err := fmt.Sscanf(v, "%d", &w); err == nil && w > 0 {
			return w
		}
	}
	return 80
}

// osc8 wraps text in an OSC 8 hyperlink when enabled.
func osc8(url, text string, enabled bool) string {
	if !enabled || url == "" {
		return text
	}
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// spinner is a debounced stderr progress indicator (cli.md Section 5):
// it renders only after 150 ms so fast operations never flicker.
type spinner struct {
	opts   *Options
	mu     sync.Mutex
	active bool
	done   chan struct{}
	wg     sync.WaitGroup
}

func newSpinner(opts *Options) *spinner {
	return &spinner{opts: opts, done: make(chan struct{})}
}

func (s *spinner) Start(label string) {
	if !isTTY(s.opts.Err) {
		return
	}
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		select {
		case <-s.done:
			return
		case <-time.After(150 * time.Millisecond):
		}
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			_, _ = fmt.Fprintf(s.opts.Err, "\r%s %s", frames[i%len(frames)], label)
			i++
			select {
			case <-s.done:
				_, _ = fmt.Fprint(s.opts.Err, "\r\x1b[K")
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()
	close(s.done)
	s.wg.Wait()
}

// withSpinner runs fn with the debounced spinner active.
func withSpinner(opts *Options, label string, fn func() error) error {
	sp := newSpinner(opts)
	sp.Start(label)
	defer sp.Stop()
	return fn()
}

// statusChip renders a single-select value with its GitHub option tint
// (a coloured dot plus the text name; never colour alone).
func statusChip(opts *Options, board *githubx.Board, fieldName, value string) string {
	if value == "" || value == "-" {
		return "-"
	}
	if !colorMode(opts) {
		return value
	}
	if board != nil {
		if f, ok := githubx.FieldByName(board.Fields, fieldName); ok {
			if opt, ok2 := f.OptionByName(value); ok2 {
				th := themeFor(opts)
				style := chipStyle(th, opt.Color)
				return style + "● " + value + "\x1b[0m"
			}
		}
	}
	return value
}

// chipStyle returns an ANSI foreground sequence for a GitHub colour.
func chipStyle(th *theme.Theme, gitHubColor string) string {
	c := th.OptionColor(gitHubColor)
	if c == nil {
		return ""
	}
	// Lip Gloss colors render as hex strings in our token maps.
	s := fmt.Sprintf("%v", c)
	if strings.HasPrefix(s, "#") && len(s) == 7 {
		var r, g, b int
		_, _ = fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
	}
	return ""
}
