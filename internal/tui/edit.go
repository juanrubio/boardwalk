package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
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

	input textinput.Model // title/date/number/text kinds (real cursor input, A4)
	area  textarea.Model  // body kind
}

// value returns the field's current edit value from its component.
func (ff *formField) value() string {
	switch ff.kind {
	case "body":
		return ff.area.Value()
	case "title", "date", "number", "text":
		return ff.input.Value()
	default:
		return ff.newVal
	}
}

// setValue replaces the edit value in the component and the mirror.
func (ff *formField) setValue(v string) {
	ff.newVal = v
	switch ff.kind {
	case "body":
		ff.area.SetValue(v)
	case "title", "date", "number", "text":
		ff.input.SetValue(v)
	}
}

func (ff *formField) blur() {
	switch ff.kind {
	case "body":
		ff.area.Blur()
	case "title", "date", "number", "text":
		ff.input.Blur()
	}
}

func (ff *formField) focus() tea.Cmd {
	switch ff.kind {
	case "body":
		return ff.area.Focus()
	case "title", "date", "number", "text":
		return ff.input.Focus()
	default:
		return nil
	}
}

func newTextInput(v string) textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.SetValue(v)
	ti.CursorEnd()
	return ti
}

func newTextArea(v string) textarea.Model {
	ta := textarea.New()
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	ta.SetValue(v)
	return ta
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
	for i := range f.fields {
		if f.fields[i].value() != f.fields[i].oldDisp {
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

func (m *model) startEdit(it githubx.Item) tea.Cmd {
	m.pane = paneEdit
	f := &formState{itemID: it.ID, origBody: it.Body}
	titleFF := formField{name: "Title", kind: "title", oldDisp: it.Title, newVal: it.Title}
	titleFF.input = newTextInput(it.Title)
	f.fields = append(f.fields, titleFF)
	body := it.Body
	bodyFF := formField{name: "Body", kind: "body", oldDisp: body, newVal: body}
	bodyFF.area = newTextArea(body)
	f.fields = append(f.fields, bodyFF)
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
				ff.input = newTextInput(old)
			case "NUMBER":
				ff.kind = "number"
				ff.input = newTextInput(old)
			case "TEXT":
				ff.kind = "text"
				ff.input = newTextInput(old)
			default:
				ff.kind = "readonly"
				ff.readonly = true
				ff.reason = fld.DataType
			}
			f.fields = append(f.fields, ff)
		}
	}
	m.form = f
	m.summary = nil
	m.syncFormLayout()
	return m.focusField(0)
}

func (m *model) startNewDraft() tea.Cmd {
	m.pane = paneNewDraft
	m.detail = nil
	titleFF := formField{name: "Title", kind: "title"}
	titleFF.input = newTextInput("")
	bodyFF := formField{name: "Body", kind: "body"}
	bodyFF.area = newTextArea("")
	m.form = &formState{isDraft: true, fields: []formField{titleFF, bodyFF}}
	m.summary = nil
	m.syncFormLayout()
	return m.focusField(0)
}

// modalSize is the centred editor modal geometry (A4): generous, but
// never closer than 4 cells to the terminal edges.
func (m *model) modalSize() (int, int) {
	w, h := m.width, m.height
	if w == 0 {
		w, h = 120, 40
	}
	mw := min(w-8, 110)
	if mw < 40 {
		mw = max(20, w-4)
	}
	mh := min(h-6, 34)
	if mh < 12 {
		mh = max(8, h-4)
	}
	return mw, mh
}

// syncFormLayout sizes the text components to the modal.
func (m *model) syncFormLayout() {
	if m.form == nil {
		return
	}
	mw, mh := m.modalSize()
	inputW := max(10, mw-24)
	bodyH := mh - len(m.form.fields) - 9
	if bodyH < 4 {
		bodyH = 4
	}
	if bodyH > 14 {
		bodyH = 14
	}
	for i := range m.form.fields {
		ff := &m.form.fields[i]
		switch ff.kind {
		case "body":
			ff.area.SetWidth(max(10, mw-6)) // indent (2) + area = inner width: zero trailing padding
			ff.area.SetHeight(bodyH)
		case "title", "date", "number", "text":
			ff.input.SetWidth(inputW)
		}
	}
}

// focusField moves form focus, blurring the old component and
// returning the new component's focus (blink) command.
func (m *model) focusField(idx int) tea.Cmd {
	f := m.form
	if f == nil || len(f.fields) == 0 {
		return nil
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(f.fields) {
		idx = len(f.fields) - 1
	}
	if f.idx >= 0 && f.idx < len(f.fields) {
		f.fields[f.idx].blur()
	}
	f.idx = idx
	f.picker = false
	return f.fields[idx].focus()
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
	for i := range m.form.fields {
		ff := &m.form.fields[i]
		v := ff.value()
		if ff.readonly || v == ff.oldDisp {
			continue
		}
		switch ff.name {
		case "Title":
			if strings.TrimSpace(v) == "" {
				return nil, fmt.Errorf("title must not be empty")
			}
			changes = append(changes, core.Change{Field: "Title", Old: ff.oldDisp, New: v, IsTitle: true, NewTitle: v})
		case "Body":
			changes = append(changes, core.Change{Field: "Body", Old: truncateSummaryLocal(ff.oldDisp), New: truncateSummaryLocal(v), IsBody: true, NewBody: v})
		default:
			ch, err := core.ParseFieldAssignment(board, base, ff.name+"="+v)
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
				v := cur.options[f.pickIdx]
				if v == "(clear)" {
					v = ""
				}
				f.fields[f.idx].setValue(v)
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
			for i := range f.fields {
				if f.fields[i].name == "Title" {
					title = f.fields[i].value()
				}
				if f.fields[i].name == "Body" {
					body = f.fields[i].value()
				}
			}
			if strings.TrimSpace(title) == "" {
				m.setFooter(statusError, "Title is required.")
				return m, nil
			}
			m.summary = &summaryState{
				isDraft: true, title: title, body: body,
				changes: []core.Change{{Field: "Draft", Old: "-", New: title}},
			}
			m.pane = paneSummary
			return m, repaintCmd()
		}
		changes, err := m.formChanges()
		if err != nil {
			m.setFooter(statusError, err.Error())
			return m, nil
		}
		if len(changes) == 0 {
			m.setFooter(statusInfo, "No changes.")
			return m, nil
		}
		m.summary = &summaryState{changes: changes}
		m.pane = paneSummary
		return m, repaintCmd()
	case "ctrl+e":
		// $VISUAL/$EDITOR handoff for the Body field (tui.md §4): the
		// TUI suspends while the editor runs; saved text returns dirty.
		editor := os.Getenv("VISUAL")
		if editor == "" {
			editor = os.Getenv("EDITOR")
		}
		if editor == "" {
			m.setFooter(statusWarning, "No $VISUAL/$EDITOR set; edit the Body field here.")
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
			m.setFooter(statusError, "Editor handoff failed: "+err.Error())
			return m, nil
		}
		path := tmp.Name()
		_, _ = tmp.WriteString(f.fields[bodyIdx].value())
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
	case "tab":
		return m, m.focusField(f.idx + 1)
	case "shift+tab":
		return m, m.focusField(f.idx - 1)
	case "down":
		if f.fields[f.idx].kind == "body" {
			return m.routeFormKey(msg)
		}
		return m, m.focusField(f.idx + 1)
	case "up":
		if f.fields[f.idx].kind == "body" {
			return m.routeFormKey(msg)
		}
		return m, m.focusField(f.idx - 1)
	case "esc":
		if f.dirtyCount() > 0 {
			m.overlay = overlayDiscard
			return m, nil
		}
		m.pane, m.form = paneNone, nil
		return m, repaintCmd()
	case "enter":
		cur := f.fields[f.idx]
		if cur.kind == "select" || cur.kind == "iteration" {
			f.picker = true
			f.pickIdx = 0
			for i, o := range cur.options {
				if o == cur.value() {
					f.pickIdx = i
				}
			}
			return m, nil
		}
		if cur.kind == "body" {
			return m.routeFormKey(msg) // newline in the text area
		}
		// Single-line input: Enter moves to the next field.
		return m, m.focusField(f.idx + 1)
	}
	// Everything else (printable keys, cursor keys, backspace, Home/
	// End) goes to the focused component: it owns the cursor (A4).
	return m.routeFormKey(msg)
}

// routeFormKey delivers a key to the focused text component and
// syncs the field mirror + validation. Cursor keys, Home/End, and
// text all land here: the component owns the cursor (A4).
func (m *model) routeFormKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	return m, m.routeToComponent(msg)
}

// routeFormMsg delivers non-key messages (cursor blink) likewise.
func (m *model) routeFormMsg(msg tea.Msg) tea.Cmd {
	return m.routeToComponent(msg)
}

func (m *model) routeToComponent(msg tea.Msg) tea.Cmd {
	f := m.form
	if f == nil || f.idx < 0 || f.idx >= len(f.fields) {
		return nil
	}
	ff := &f.fields[f.idx]
	var cmd tea.Cmd
	switch ff.kind {
	case "body":
		var nm textarea.Model
		nm, cmd = ff.area.Update(msg)
		ff.area = nm
	case "title", "date", "number", "text":
		var nm textinput.Model
		nm, cmd = ff.input.Update(msg)
		ff.input = nm
	default:
		return nil
	}
	ff.newVal = ff.value()
	ff.err = validateField(ff.kind, ff.newVal)
	return cmd
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
