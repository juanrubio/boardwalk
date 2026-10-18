// Package fixture provides an offline, in-memory githubx.Client fed
// by the captured/synthetic board JSON in testdata/fixtures
// (ADR 0004). It implements the full Client surface, including
// mutations, so CLI/core/TUI tests never touch the network.
package fixture

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/juanrubio/boardwalk/internal/exiterr"
	"github.com/juanrubio/boardwalk/internal/githubx"
)

// Client is a fixture-backed githubx.Client.
type Client struct {
	mu          sync.Mutex
	Boards      []*githubx.Board
	Contents    map[string]githubx.Item // contentID -> item template for AddItem
	FailField   map[string]error        // field name -> error for SetFieldValue (partial-failure tests)
	PositionErr error                   // when set, UpdateItemPosition returns it without mutating
	NextID      int64
	AuthErr     error // when set, CheckAuth returns it
}

// LoadDir reads every *.board.json in dir into a Client.
func LoadDir(dir string) (*Client, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.board.json"))
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no *.board.json fixtures in %s", dir)
	}
	c := &Client{Contents: map[string]githubx.Item{}, FailField: map[string]error{}, NextID: 990000}
	for _, path := range entries {
		data, err := os.ReadFile(path) //nolint:gosec // test fixture path
		if err != nil {
			return nil, err
		}
		var b githubx.Board
		if err := json.Unmarshal(data, &b); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		board := b
		c.Boards = append(c.Boards, &board)
	}
	return c, nil
}

// Board returns the fixture board for a project ref (test helper).
func (c *Client) Board(ref string) *githubx.Board {
	for _, b := range c.Boards {
		if b.Project.Ref() == ref {
			return b
		}
	}
	return nil
}

func (c *Client) CheckAuth(_ context.Context) error { return c.AuthErr }

func (c *Client) ListProjects(_ context.Context, owner string) ([]githubx.Project, error) {
	var out []githubx.Project
	for _, b := range c.Boards {
		if owner == "" || b.Project.Owner == owner {
			out = append(out, b.Project)
		}
	}
	return out, nil
}

func (c *Client) ResolveProject(_ context.Context, ref string) (githubx.Project, error) {
	if b := c.Board(ref); b != nil {
		return b.Project, nil
	}
	// URLs: .../projects/N with owner second-to-last segment group.
	if strings.HasPrefix(ref, "http") {
		for _, b := range c.Boards {
			if b.Project.URL == ref {
				return b.Project, nil
			}
		}
	}
	return githubx.Project{}, exiterr.NotFoundf("no fixture project %q", ref)
}

func (c *Client) boardByID(id string) *githubx.Board {
	for _, b := range c.Boards {
		if b.Project.ID == id {
			return b
		}
	}
	return nil
}

func (c *Client) LoadBoardPage(_ context.Context, projectID, cursor string, pageSize int, _ string) ([]githubx.Field, []githubx.Item, string, error) {
	b := c.boardByID(projectID)
	if b == nil {
		return nil, nil, "", exiterr.NotFoundf("no fixture project id %q", projectID)
	}
	start := 0
	if cursor != "" {
		n, err := strconv.Atoi(cursor)
		if err != nil {
			return nil, nil, "", exiterr.Usagef("bad fixture cursor %q", cursor)
		}
		start = n
	}
	if start > len(b.Items) {
		start = len(b.Items)
	}
	end := start + pageSize
	if end > len(b.Items) {
		end = len(b.Items)
	}
	next := ""
	if end < len(b.Items) {
		next = strconv.Itoa(end)
	}
	items := make([]githubx.Item, end-start)
	copy(items, b.Items[start:end])
	return b.Fields, items, next, nil
}

func (c *Client) findItem(itemID string) (*githubx.Board, *githubx.Item) {
	for _, b := range c.Boards {
		for i := range b.Items {
			if b.Items[i].ID == itemID {
				return b, &b.Items[i]
			}
		}
	}
	return nil, nil
}

func (c *Client) LoadItemDetail(_ context.Context, itemID string) (githubx.Item, error) {
	_, it := c.findItem(itemID)
	if it == nil {
		return githubx.Item{}, exiterr.NotFoundf("no fixture item %q", itemID)
	}
	cp := *it
	if cp.Body == "" {
		cp.Body = "Fixture body for " + it.Title + "."
	}
	return cp, nil
}

func (c *Client) ResolveContent(_ context.Context, ref string) (string, githubx.Item, error) {
	// Accept owner/repo#number or a URL containing /issues/N or /pull/N.
	repo, number, kind := parseContentRef(ref)
	if repo == "" {
		return "", githubx.Item{}, exiterr.Usagef("cannot parse content reference %q", ref)
	}
	id := fmt.Sprintf("content:%s#%d", repo, number)
	if it, ok := c.Contents[id]; ok {
		return id, it, nil
	}
	it := githubx.Item{
		Kind: kind, Repo: repo, Number: number, Title: fmt.Sprintf("Fixture %s %s#%d", kind, repo, number),
		URL: fmt.Sprintf("https://github.com/%s/issues/%d", repo, number),
	}
	c.Contents[id] = it
	return id, it, nil
}

func parseContentRef(ref string) (repo string, number int, kind githubx.Kind) {
	kind = githubx.KindIssue
	s := ref
	if strings.Contains(s, "/pull/") {
		kind = githubx.KindPR
	}
	if i := strings.Index(s, "#"); i >= 0 {
		n, err := strconv.Atoi(s[i+1:])
		if err != nil {
			return "", 0, kind
		}
		return s[:i], n, kind
	}
	parts := strings.Split(strings.Trim(s, "/"), "/")
	if len(parts) >= 4 {
		n, err := strconv.Atoi(parts[len(parts)-1])
		if err == nil {
			return parts[len(parts)-4] + "/" + parts[len(parts)-3], n, kind
		}
	}
	return "", 0, kind
}

func (c *Client) AddItem(_ context.Context, projectID, contentID string) (githubx.Item, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b := c.boardByID(projectID)
	if b == nil {
		return githubx.Item{}, exiterr.NotFoundf("no fixture project id %q", projectID)
	}
	tpl, ok := c.Contents[contentID]
	if !ok {
		return githubx.Item{}, exiterr.NotFoundf("no fixture content %q", contentID)
	}
	for _, it := range b.Items {
		if it.Repo == tpl.Repo && it.Number == tpl.Number && it.Kind == tpl.Kind {
			return it, nil // already present; core reports it
		}
	}
	c.NextID++
	tpl.ID = fmt.Sprintf("PVTI_fixture_new_%d", c.NextID)
	tpl.DatabaseID = c.NextID
	b.Items = append(b.Items, tpl)
	return tpl, nil
}

func (c *Client) CreateDraft(_ context.Context, projectID, title, body string) (githubx.Item, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b := c.boardByID(projectID)
	if b == nil {
		return githubx.Item{}, exiterr.NotFoundf("no fixture project id %q", projectID)
	}
	c.NextID++
	it := githubx.Item{
		ID: fmt.Sprintf("PVTI_fixture_new_%d", c.NextID), DatabaseID: c.NextID,
		Kind: githubx.KindDraft, Title: title, Body: body,
	}
	b.Items = append(b.Items, it)
	return it, nil
}

func (c *Client) SetFieldValues(ctx context.Context, projectID, itemID string, assigns []githubx.FieldAssign) []githubx.FieldResult {
	out := make([]githubx.FieldResult, len(assigns))
	for i, a := range assigns {
		_, err := c.setFieldValue(ctx, projectID, itemID, a.FieldID, a.Value)
		out[i] = githubx.FieldResult{FieldName: a.FieldName, Err: err}
	}
	return out
}

func (c *Client) setFieldValue(_ context.Context, projectID, itemID, fieldID string, value githubx.MutationValue) (githubx.Item, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, it := c.findItem(itemID)
	if it == nil || b.Project.ID != projectID {
		return githubx.Item{}, exiterr.NotFoundf("no fixture item %q", itemID)
	}
	field, ok := githubx.FieldByID(b.Fields, fieldID)
	if !ok {
		return githubx.Item{}, exiterr.NotFoundf("no fixture field %q", fieldID)
	}
	if err, fail := c.FailField[field.Name]; fail {
		return githubx.Item{}, err
	}
	nv := githubx.FieldValue{FieldID: field.ID, FieldName: field.Name, DataType: field.DataType}
	switch {
	case value.Clear:
		// remove below
	case value.OptionID != nil:
		nv.OptionID = *value.OptionID
		for _, o := range field.Options {
			if o.ID == *value.OptionID {
				nv.OptionName = o.Name
			}
		}
	case value.Text != nil:
		nv.Text = *value.Text
	case value.Number != nil:
		nv.Number = value.Number
	case value.Date != nil:
		nv.Date = parseDate(*value.Date)
	case value.IterationID != nil:
		nv.IterationID = *value.IterationID
		for _, it2 := range field.Iterations {
			if it2.ID == *value.IterationID {
				nv.IterationTitle = it2.Title
			}
		}
	}
	for i, v := range it.Values {
		if v.FieldID == fieldID {
			if value.Clear {
				it.Values = append(it.Values[:i], it.Values[i+1:]...)
			} else {
				it.Values[i] = nv
			}
			return *it, nil
		}
	}
	if !value.Clear {
		it.Values = append(it.Values, nv)
	}
	return *it, nil
}

func parseDate(s string) *time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

func (c *Client) UpdateTitleBody(_ context.Context, item githubx.Item, title, body *string) (githubx.Item, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, it := c.findItem(item.ID)
	if it == nil {
		return githubx.Item{}, exiterr.NotFoundf("no fixture item %q", item.ID)
	}
	if title != nil {
		it.Title = *title
	}
	if body != nil {
		it.Body = *body
	}
	return *it, nil
}

func (c *Client) ConvertDraftToIssue(_ context.Context, _, itemID, repo string) (githubx.Item, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, it := c.findItem(itemID)
	if it == nil {
		return githubx.Item{}, exiterr.NotFoundf("no fixture item %q", itemID)
	}
	if it.Kind != githubx.KindDraft {
		return githubx.Item{}, exiterr.Usagef("only drafts can be converted")
	}
	it.Kind = githubx.KindIssue
	it.Repo = repo
	it.Number = 900 + int(it.DatabaseID%97)
	it.URL = "https://github.com/" + repo + "/issues/" + strconv.Itoa(it.Number)
	return *it, nil
}

func (c *Client) DeleteItem(_ context.Context, projectID, itemID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	b := c.boardByID(projectID)
	if b == nil {
		return exiterr.NotFoundf("no fixture project id %q", projectID)
	}
	for i := range b.Items {
		if b.Items[i].ID == itemID {
			b.Items = append(b.Items[:i], b.Items[i+1:]...)
			return nil
		}
	}
	return exiterr.NotFoundf("no fixture item %q", itemID)
}

func (c *Client) UpdateItemPosition(_ context.Context, projectID, itemID, afterID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.PositionErr != nil {
		return c.PositionErr
	}
	b := c.boardByID(projectID)
	if b == nil {
		return exiterr.NotFoundf("no fixture project id %q", projectID)
	}
	foundItem, foundAfter := false, afterID == ""
	for i := range b.Items {
		if b.Items[i].ID == itemID {
			foundItem = true
		}
		if b.Items[i].ID == afterID {
			foundAfter = true
		}
	}
	if !foundItem {
		return exiterr.NotFoundf("no fixture item %q", itemID)
	}
	if !foundAfter {
		return exiterr.NotFoundf("no fixture item %q", afterID)
	}
	b.Items = githubx.ApplyReorder(b.Items, itemID, afterID)
	return nil
}

func (c *Client) ArchiveItem(_ context.Context, _, itemID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, it := c.findItem(itemID)
	if it == nil {
		return exiterr.NotFoundf("no fixture item %q", itemID)
	}
	it.Archived = true
	return nil
}
