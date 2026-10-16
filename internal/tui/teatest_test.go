package tui

import (
	"bytes"
	"io"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"
)

// Program-level acceptance (tui.md §7 names teatest): the real Bubble
// Tea program paints the fixture board, navigates, and quits cleanly.
func TestTeatestProgramPaintsAndQuits(t *testing.T) {
	m, _ := testModel(t, "fixture-user/1")
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
	t.Cleanup(func() { _ = tm.Quit() })

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Grouped by Status"))
	}, teatest.WithCheckInterval(50*time.Millisecond), teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyPressMsg{Code: 'j', Text: "j"})
	tm.Send(tea.KeyPressMsg{Code: '?', Text: "?"})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Keys"))
	}, teatest.WithCheckInterval(50*time.Millisecond), teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyPressMsg{Code: 'q', Text: "q"}) // closes help
	tm.Send(tea.KeyPressMsg{Code: 'q', Text: "q"}) // opens quit confirmation
	tm.Send(tea.KeyPressMsg{Code: 'y', Text: "y"}) // confirms quit
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))

	out, err := io.ReadAll(tm.FinalOutput(t))
	if err != nil {
		t.Fatal(err)
	}
	_ = out // final frame is the restored screen; paint is proven above
	final := tm.FinalModel(t).(*model)
	if final == nil || final.board == nil || len(final.board.Items) != 14 {
		t.Fatalf("final model board = %+v", final)
	}
	if final.overlay != overlayNone {
		t.Fatalf("help should be closed, overlay=%v", final.overlay)
	}
}
