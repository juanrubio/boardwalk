// Package githubx defines Boardwalk's GitHub Projects v2 domain types
// and the Client interface the core service depends on. The concrete
// GraphQL implementation lives in client.go (go-gh, ADR 0002); tests
// substitute fixture clients with the same shapes.
package githubx

import "time"

// Project is a Projects v2 project as discovered from GitHub.
type Project struct {
	ID         string // node ID
	Number     int
	Title      string
	Owner      string // user or organisation login
	OwnerKind  string // "user" or "organization"
	URL        string
	Closed     bool
	ItemsTotal int
}

// Ref returns the canonical owner/number reference.
func (p Project) Ref() string {
	return p.Owner + "/" + itoa(p.Number)
}

// Option is one value of a single-select field.
type Option struct {
	ID          string
	Name        string
	Color       string // GitHub palette name: GRAY, BLUE, ...
	Description string
}

// Iteration is one value of an iteration field.
type Iteration struct {
	ID           string
	Title        string
	StartDate    time.Time
	DurationDays int
}

// Field is a project field discovered at runtime (PRD Section 9).
// DataType mirrors GitHub: SINGLE_SELECT, ITERATION, DATE, NUMBER,
// TEXT, ASSIGNEES, TITLE, etc.
type Field struct {
	ID         string
	Name       string
	DataType   string
	Options    []Option    // SINGLE_SELECT only
	Iterations []Iteration // ITERATION only (current + upcoming)
}

// FieldByID returns the field with this node ID, if present.
func FieldByID(fields []Field, id string) (Field, bool) {
	for _, f := range fields {
		if f.ID == id {
			return f, true
		}
	}
	return Field{}, false
}

// OptionByName returns the named option, if present.
func (f Field) OptionByName(name string) (Option, bool) {
	for _, o := range f.Options {
		if o.Name == name {
			return o, true
		}
	}
	return Option{}, false
}

// FieldByName returns the named field, if present.
func FieldByName(fields []Field, name string) (Field, bool) {
	for _, f := range fields {
		if f.Name == name {
			return f, true
		}
	}
	return Field{}, false
}

// FieldValue is one field's current value on an item.
type FieldValue struct {
	FieldID   string
	FieldName string
	DataType  string
	// Exactly one of these is meaningful per DataType.
	Text           string
	Number         *float64
	Date           *time.Time
	OptionID       string
	OptionName     string
	IterationID    string
	IterationTitle string
	Users          []string // logins for ASSIGNEES/people fields
}

// Kind distinguishes linked issues, pull requests, and drafts.
type Kind string

const (
	KindIssue Kind = "issue"
	KindPR    Kind = "pr"
	KindDraft Kind = "draft"
)

// Item is a project item with the fields cards and rows display.
// Bodies are never carried here (performance.md: no bodies in lists).
type Item struct {
	ID         string // node ID (PVTI_...)
	DatabaseID int64  // numeric ID used by draft:<id> references
	Kind       Kind
	Title      string
	Repo       string // owner/repo for linked items; "" for drafts
	Number     int    // issue/PR number; 0 for drafts
	URL        string
	Body       string // populated only by detail fetches
	Values     []FieldValue
	Archived   bool
}

// Ref returns the copyable command reference (cli.md Section 1):
// owner/repo#number for linked items, draft:<databaseId> for drafts.
func (it Item) Ref() string {
	if it.Kind == KindDraft {
		return "draft:" + itoa64(it.DatabaseID)
	}
	return it.Repo + "#" + itoa(it.Number)
}

// Value returns the value of the named field, if set.
func (it Item) Value(fieldName string) (FieldValue, bool) {
	for _, v := range it.Values {
		if v.FieldName == fieldName {
			return v, true
		}
	}
	return FieldValue{}, false
}

// Board is a fully loaded project board used by CLI list and tests.
type Board struct {
	Project Project
	Fields  []Field
	Items   []Item
}

// DefaultBoardField picks the single-select field named Status, else
// the first single-select field (tui.md Section 1 rules 2-3). It never
// depends on any remembered UI preference.
func DefaultBoardField(fields []Field) (Field, bool) {
	for _, f := range fields {
		if f.DataType == "SINGLE_SELECT" && f.Name == "Status" {
			return f, true
		}
	}
	for _, f := range fields {
		if f.DataType == "SINGLE_SELECT" {
			return f, true
		}
	}
	return Field{}, false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func itoa64(n int64) string { return itoa(int(n)) }
