package tui

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/juanrubio/boardwalk/internal/core"
	"github.com/juanrubio/boardwalk/internal/exiterr"
	"github.com/juanrubio/boardwalk/internal/githubx"
	"github.com/juanrubio/boardwalk/internal/theme"
)

var updateGolden = flag.Bool("update-golden", false, "update golden frames")

// Golden frames (tui.md §7): every named state at its size/profile is
// compared byte-for-byte; a diff fails until deliberately updated with
// `go test ./internal/tui/ -run TestGoldenFrames -update-golden`.
func TestGoldenFrames(t *testing.T) {
	type frame struct {
		name  string
		w, h  int
		setup func(m *model)
	}
	frames := []frame{
		{"board-120x40-mono", 120, 40, nil},
		{"board-80x24-mono", 80, 24, nil},
		{"board-200x50-dark", 200, 50, func(m *model) { m.theme = theme.Dark() }},
		{"board-120x40-light", 120, 40, func(m *model) { m.theme = theme.Light() }},
		{"compact-collapsed-80x24", 80, 24, func(m *model) {
			m.density = "compact"
			m.collapsed["Done"] = true
			m.collapsed["Done "] = true
			// collapse actual last column label if present
			cols := m.columns()
			if len(cols) > 0 {
				m.collapsed[cols[len(cols)-1].label] = true
			}
		}},
		{"list-120x40", 120, 40, func(m *model) { m.view = viewList }},
		{"split-detail-120x40", 120, 40, func(m *model) {
			it, _ := m.focusedItem()
			m.pane = paneDetail
			m.detail = &it
			m.detailBody = "Fixture body rendered for the golden frame."
		}},
		{"edit-dirty-120x40", 120, 40, func(m *model) {
			it, _ := m.focusedItem()
			m.detail = &it
			m.startEdit(it)
			m.form.fields[0].newVal = it.Title + " edited"
		}},
		{"summary-partial-120x40", 120, 40, func(m *model) {
			it, _ := m.focusedItem()
			m.detail = &it
			m.pane = paneSummary
			m.summary = &summaryState{
				sent: true,
				changes: []core.Change{
					{Field: "Status", Old: "Ready", New: "In Progress"},
					{Field: "Estimate", Old: "3", New: "5"},
				},
				results: []core.Result{
					{Change: core.Change{Field: "Status", Old: "Ready", New: "In Progress"}},
					{Change: core.Change{Field: "Estimate", Old: "3", New: "5"}, Err: exiterr.Failuref("GitHub rejected the request (FORBIDDEN)")},
				},
			}
		}},
		{"help-80x24", 80, 24, func(m *model) { m.overlay = overlayHelp }},
		{"search-80x24", 80, 24, func(m *model) {
			m.overlay = overlaySearch
			m.searchText = "Fixture"
			m.filter = "Fixture"
		}},
		{"stale-120x40", 120, 40, func(m *model) {
			m.stale = true
			m.loading = true
			m.snapAt = time.Now().Add(-4 * time.Minute)
		}},
		{"archive-80x24", 80, 24, func(m *model) { m.overlay = overlayArchive }},
		{"quit-80x24-dark", 80, 24, func(m *model) { m.theme = theme.Dark(); m.requestQuit() }},
		{"quit-120x40-light", 120, 40, func(m *model) { m.theme = theme.Light(); m.requestQuit(); m.quitChoice = true }},
		{"quit-60x15-mono", 60, 15, func(m *model) { m.requestQuit() }},
		{"delete-stage2-80x24", 80, 24, func(m *model) { m.overlay = overlayDelete; m.deleteStage = 1 }},
		{"group-picker-120x40", 120, 40, func(m *model) { m.overlay = overlayGroup }},
		{"board-picker-120x40", 120, 40, func(m *model) {
			m.boards = []githubx.Project{
				{ID: "o7", Owner: "fixture-org", Number: 7, Title: "Org Roadmap"},
				{ID: "u1", Owner: "fixture-user", Number: 1, Title: "SnazzyEmail (fixture)"},
				{ID: "u3", Owner: "fixture-user", Number: 3, Title: "Life Board"},
			}
			// testModel's project is a distinct instance; mark by ref match.
			m.boards[1].ID = m.project.ID
			m.overlay = overlayBoards
			m.boardIdx = 1
		}},
		{"too-small-50x12", 50, 12, nil},
		{"new-draft-120x40", 120, 40, func(m *model) { m.startNewDraft() }},
	}
	for _, fr := range frames {
		t.Run(fr.name, func(t *testing.T) {
			m, _ := testModel(t, "fixture-user/1")
			m.width, m.height = fr.w, fr.h
			m.loadedAt = time.Now().Add(-4 * time.Minute)
			if fr.setup != nil {
				fr.setup(m)
			}
			got := m.View().Content + "\n"
			path := filepath.Join("testdata", "golden", fr.name+".golden")
			if *updateGolden {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("missing golden %s (run with -update): %v", path, err)
			}
			if got != string(want) {
				t.Fatalf("golden mismatch for %s:\n--- got ---\n%s\n--- want ---\n%s", fr.name, got, want)
			}
		})
	}
}

// Benchmark/performance assertions (performance.md B6/B7): keypress to
// frame on synthetic 1,000- and 10,000-item boards.
func TestFrameBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	for _, tc := range []struct {
		n      int
		budget time.Duration
	}{{1000, 16 * time.Millisecond}, {10000, 50 * time.Millisecond}} {
		m := syntheticModel(tc.n)
		// Warm up.
		_ = m.View()
		const steps = 30
		var worst time.Duration
		var total time.Duration
		for i := 0; i < steps; i++ {
			start := time.Now()
			mm, _ := m.Update(key("j"))
			m = mm.(*model)
			_ = m.View()
			d := time.Since(start)
			total += d
			if d > worst {
				worst = d
			}
		}
		avg := total / steps
		t.Logf("%d items: avg frame %v worst %v budget %v", tc.n, avg, worst, tc.budget)
		if avg > tc.budget {
			t.Errorf("%d items: avg frame %v exceeds budget %v", tc.n, avg, tc.budget)
		}
	}
}

func syntheticModel(n int) *model {
	fields := []githubx.Field{
		{ID: "FS", Name: "Status", DataType: "SINGLE_SELECT", Options: []githubx.Option{
			{ID: "O1", Name: "Backlog", Color: "GRAY"}, {ID: "O2", Name: "In Progress", Color: "BLUE"}, {ID: "O3", Name: "Done", Color: "GREEN"},
		}},
		{ID: "FP", Name: "Priority", DataType: "SINGLE_SELECT", Options: []githubx.Option{{ID: "P0", Name: "P0", Color: "RED"}, {ID: "P1", Name: "P1", Color: "ORANGE"}}},
	}
	items := make([]githubx.Item, n)
	statuses := []string{"Backlog", "In Progress", "Done"}
	for i := 0; i < n; i++ {
		st := statuses[i%3]
		items[i] = githubx.Item{
			ID: fmt.Sprintf("PVTI_synth_%d", i), DatabaseID: int64(1000 + i), Kind: githubx.KindIssue,
			Title: fmt.Sprintf("Synthetic item %d", i), Repo: "fixture-org/repo", Number: i + 1,
			Values: []githubx.FieldValue{
				{FieldID: "FS", FieldName: "Status", DataType: "SINGLE_SELECT", OptionName: st, OptionID: "O1"},
			},
		}
	}
	m := &model{
		project: githubx.Project{ID: "PS", Number: 7, Title: "Synthetic", Owner: "fixture-org"},
		theme:   theme.Monochrome(), density: "comfortable",
		collapsed: map[string]bool{}, pending: map[string]*pendingMove{}, pendingPos: map[string]*pendingReorder{}, failed: map[string]string{},
		board: &githubx.Board{Project: githubx.Project{ID: "PS", Number: 7, Title: "Synthetic", Owner: "fixture-org"}, Fields: fields, Items: items},
		width: 120, height: 40,
	}
	m.rebuildFocus()
	return m
}

var _ = strings.Contains
