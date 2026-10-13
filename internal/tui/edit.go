package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/juanrubio/gh-boardwalk/internal/core"
	"github.com/juanrubio/gh-boardwalk/internal/githubx"
)

// formField is one editable row in the detail-pane form.
type formField struct {
	name     string // Title, Body, or project field name
	kind     string // title|body|select|iteration|date|number|text|readonly
	oldDisp  string
	newVal   string // display/edit value
	options  []string
	optIDs   []string
	readonly bool
	reason   string
	err      string
}

type formState struct {
	itemID   string
	isDraft  bool // new draft form
	fields   []formField
	idx      int
	picker   bool // picker open for current select field
	pickIdx  int
	origBody string
}

func (f *formState) dirtyCount() int {
	n := 0
	for _, ff := range f.fields {
		if ff.newVal != ff.oldDisp {
			n++
		}
	}
	return n
}

type summaryState struct {
	changes      []core.Change
	results      []core.Result
	sent         bool
	err          string
	isDraft      bool
	title        string
	body         string
	pendingRetry []core.Change
}

func (m *model) startEdit(it githubx.Item) {
	m.pane = paneEdit
	f := &formState{itemID: it.ID, origBody: it.Body}
	f.fields = append(f.fields, formField{name: "Title", kind: "title", oldDisp: it.Title, newVal: it.Title})
	body := it.Body
	f.fields = append(f.fields, formField{name: "Body", kind: "body", oldDisp: body, newVal: body})
	if m.board != nil {
		for _, fld := range m.board.Fields {
			if fld.DataType == "TITLE" {
				continue
			}
			ff := formField{name: fld.Name}
			old := ""
			if v, ok := it.Value(fld.Name); ok {
				old = core.DisplayValue(v)
				if old == "-" {
					old = ""
				}
			}
			ff.oldDisp, ff.newVal = old, old
			switch fld.DataType {
			case "SINGLE_SELECT":
				ff.kind = "select"
				ff.options = append(ff.options, "(clear)")
				ff.optIDs = append(ff.optIDs, "")
				for _, o := range fld.Options {
					ff.options = append(ff.options, o.Name)
					ff.optIDs = append(ff.optIDs, o.ID)
				}
			case "ITERATION":
				ff.kind = "iteration"
				ff.options = append(ff.options, "(clear)")
				for _, it2 := range fld.Iterations {
					ff.options = append(ff.options, it2.Title)
				}
			case "DATE":
				ff.kind = "date"
			case "NUMBER":
				ff.kind = "number"
			case "TEXT":
				ff.kind = "text"
			default:
				ff.kind = "readonly"
				ff.readonly = true
				ff.reason = fld.DataType + " is edited with `item edit` in the MVP"
			}
			f.fields = append(f.fields, ff)
		}
	}
	m.form = f
	m.summary = nil
}

func (m *model) startNewDraft() {
	m.pane = paneNewDraft
	m.detail = nil
	m.form = &formState{isDraft: true, fields: []formField{
		{name: "Title", kind: "title", oldDisp: "", newVal: ""},
		{name: "Body", kind: "body", oldDisp: "", newVal: ""},
	}}
	m.summary = nil
}

func (m *model) formChanges() ([]core.Change, error) {
	if m.form == nil || m.board == nil {
		return nil, fmt.Errorf("no form")
	}
	// find base item
	var base githubx.Item
	if m.form.isDraft {
		base = githubx.Item{Kind: githubx.KindDraft}
	} else if m.detail != nil {
		base = *m.detail
	} else if it, ok := m.focusedItem(); ok {
		base = it
	}
	var changes []core.Change
	board := &githubx.Board{Project: m.project, Fields: m.board.Fields, Items: m.board.Items}
	for _, ff := range m.form.fields {
		if ff.readonly || ff.newVal == ff.oldDisp {
			continue
		}
		switch ff.name {
		case "Title":
			if strings.TrimSpace(ff.newVal) == "" {
				return nil, fmt.Errorf("title must not be empty")
			}
			changes = append(changes, core.Change{Field: "Title", Old: ff.oldDisp, New: ff.newVal, IsTitle: true, NewTitle: ff.newVal})
		case "Body":
			changes = append(changes, core.Change{Field: "Body", Old: truncateSummaryLocal(ff.oldDisp), New: truncateSummaryLocal(ff.newVal), IsBody: true, NewBody: ff.newVal})
		default:
			ch, err := core.ParseFieldAssignment(board, base, ff.name+"="+ff.newVal)
			if err != nil {
				return nil, err
			}
			changes = append(changes, ch)
		}
	}
	return changes, nil
}

func truncateSummaryLocal(s string) string {
	r := []rune(s)
	if len(r) > 40 {
		return string(r[:39]) + "…"
	}
	return s
}

func (m *model) onFormKey(msg tea.KeyPressMsg, s, ks string) (tea.Model, tea.Cmd) {
	f := m.form
	// Picker open for select/iteration
	if f.picker {
		cur := f.fields[f.idx]
		switch ks {
		case "esc":
			f.picker = false
		case "enter":
			if f.pickIdx >= 0 && f.pickIdx < len(cur.options) {
				f.fields[f.idx].newVal = cur.options[f.pickIdx]
				if cur.options[f.pickIdx] == "(clear)" {
					f.fields[f.idx].newVal = ""
				}
			}
			f.picker = false
		default:
			switch s {
			case "j", "down":
				if f.pickIdx < len(cur.options)-1 {
					f.pickIdx++
				}
			case "k", "up":
				if f.pickIdx > 0 {
					f.pickIdx--
				}
			}
		}
		return m, nil
	}

	switch ks {
	case "ctrl+s":
		if f.isDraft {
			title := ""
			var body string
			for _, ff := range f.fields {
				if ff.name == "Title" {
					title = ff.newVal
				}
				if ff.name == "Body" {
					body = ff.newVal
				}
			}
			if strings.TrimSpace(title) == "" {
				m.footer = "Title is required."
				return m, nil
			}
			m.summary = &summaryState{
				isDraft: true, title: title, body: body,
				changes: []core.Change{{Field: "Draft", Old: "-", New: title}},
			}
			m.pane = paneSummary
			return m, nil
		}
		changes, err := m.formChanges()
		if err != nil {
			m.footer = err.Error()
			return m, nil
		}
		if len(changes) == 0 {
			m.footer = "No changes."
			return m, nil
		}
		m.summary = &summaryState{changes: changes}
		m.pane = paneSummary
		return m, nil
	case "ctrl+e":
		// $VISUAL/$EDITOR handoff for the Body field (tui.md §4): the
		// TUI suspends while the editor runs; saved text returns dirty.
		editor := os.Getenv("VISUAL")
		if editor == "" {
			editor = os.Getenv("EDITOR")
		}
		if editor == "" {
			m.footer = "No $VISUAL/$EDITOR set; edit the Body field here."
			return m, nil
		}
		bodyIdx := -1
		for i, ff := range f.fields {
			if ff.name == "Body" {
				bodyIdx = i
			}
		}
		if bodyIdx < 0 {
			return m, nil
		}
		tmp, err := os.CreateTemp("", "boardwalk-body-*.md")
		if err != nil {
			m.footer = "Editor handoff failed: " + err.Error()
			return m, nil
		}
		path := tmp.Name()
		_, _ = tmp.WriteString(f.fields[bodyIdx].newVal)
		_ = tmp.Close()
		parts := strings.Fields(editor)
		args := append(append([]string{}, parts[1:]...), path)
		cmd := exec.Command(parts[0], args...) //nolint:gosec // user's configured editor
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			data, rerr := os.ReadFile(path) //nolint:gosec // our own temp file
			_ = os.Remove(path)
			if err != nil {
				return editorDoneMsg{err: err}
			}
			if rerr != nil {
				return editorDoneMsg{err: rerr}
			}
			return editorDoneMsg{body: string(data), fieldIdx: bodyIdx}
		})
	case "tab", "down":
		if f.idx < len(f.fields)-1 {
			f.idx++
		}
		return m, nil
	case "shift+tab", "up":
		if f.idx > 0 {
			f.idx--
		}
		return m, nil
	case "esc":
		if f.dirtyCount() > 0 {
			m.overlay = overlayDiscard
			return m, nil
		}
		m.pane, m.form = paneNone, nil
		return m, nil
	case "enter":
		cur := f.fields[f.idx]
		if cur.kind == "select" || cur.kind == "iteration" {
			f.picker = true
			f.pickIdx = 0
			for i, o := range cur.options {
				if o == cur.newVal {
					f.pickIdx = i
				}
			}
		}
		return m, nil
	case "backspace":
		cur := &f.fields[f.idx]
		if cur.readonly {
			return m, nil
		}
		if cur.kind == "select" || cur.kind == "iteration" {
			return m, nil
		}
		r := []rune(cur.newVal)
		if len(r) > 0 {
			cur.newVal = string(r[:len(r)-1])
			cur.err = validateField(cur.kind, cur.newVal)
		}
		return m, nil
	}
	// Text capture: focused input captures printable keys (§2 A1)
	if msg.Key().Text != "" {
		cur := &f.fields[f.idx]
		if cur.readonly || cur.kind == "select" || cur.kind == "iteration" {
			return m, nil
		}
		cur.newVal += msg.Key().Text
		cur.err = validateField(cur.kind, cur.newVal)
		return m, nil
	}
	// Space for text fields (Key.Text is " " usually handled above)
	if s == " " {
		cur := &f.fields[f.idx]
		if !cur.readonly && cur.kind != "select" && cur.kind != "iteration" {
			cur.newVal += " "
		}
		return m, nil
	}
	return m, nil
}

func validateField(kind, v string) string {
	if v == "" {
		return ""
	}
	switch kind {
	case "date":
		if _, err := parseDate(v); err != nil {
			return "want YYYY-MM-DD"
		}
	case "number":
		if _, err := parseNumber(v); err != nil {
			return "want a number"
		}
	}
	return ""
}

func parseDate(s string) (interface{}, error) {
	t := parseDateStr(s)
	if t == nil {
		return nil, fmt.Errorf("invalid date %q", s)
	}
	return t, nil
}

func parseNumber(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscan(s, &f)
	return f, err
}

func (m *model) onSummaryKey(s, ks string) (tea.Model, tea.Cmd) {
	sum := m.summary
	switch ks {
	case "enter":
		if sum.isDraft {
			svc, proj := m.svc, m.project
			title, body := sum.title, sum.body
			m.savingMsg = "Creating…"
			return m, func() tea.Msg {
				if svc == nil || svc.Client == nil {
					return draftResultMsg{err: fmt.Errorf("no GitHub client configured")}
				}
				it, err := svc.Client.CreateDraft(m.ctx, proj.ID, title, body)
				return draftResultMsg{item: it, err: err}
			}
		}
		// Determine changes to send: all if not sent, failed only if sent
		var toSend []core.Change
		if !sum.sent {
			toSend = sum.changes
		} else {
			for _, r := range sum.results {
				if r.Err != nil {
					toSend = append(toSend, r.Change)
				}
			}
			if len(toSend) == 0 {
				m.pane, m.form, m.summary = paneDetail, nil, nil
				return m, nil
			}
		}
		if m.detail == nil {
			return m, nil
		}
		svc, proj := m.svc, m.project
		it := *m.detail
		m.savingMsg = "Saving…"
		// Keep summary changes as the full set; results replace on return.
		// For retry, we need summary to map results back; store pending set.
		sum.pendingRetry = toSend
		return m, func() tea.Msg {
			results := svc.ApplyChanges(m.ctx, proj, it, toSend)
			return saveResultMsg{results: results}
		}
	case "esc":
		if sum.sent && anyFailed(sum.results) {
			m.overlay = overlayDiscard
			return m, nil
		}
		m.pane, m.summary = paneDetail, nil
		if m.form != nil && !m.form.isDraft {
			m.pane = paneEdit
		} else if m.form != nil && m.form.isDraft {
			m.pane = paneNewDraft
		}
		return m, nil
	default:
		if s == "e" {
			if m.form != nil && m.form.isDraft {
				m.pane = paneNewDraft
			} else {
				m.pane = paneEdit
			}
			m.summary = nil
			return m, nil
		}
	}
	return m, nil
}

func anyFailed(results []core.Result) bool {
	for _, r := range results {
		if r.Err != nil {
			return true
		}
	}
	return false
}
