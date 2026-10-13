package githubx

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/juanrubio/gh-boardwalk/internal/exiterr"
)

// ---- GraphQL response shapes -------------------------------------------

type gqlProject struct {
	ID     string
	Number int
	Title  string
	URL    string
	Closed bool
	Owner  struct {
		Login string
		// Typename distinguishes user vs organisation owners.
		Typename string `json:"__typename"`
	}
	Items struct{ TotalCount int }
}

func (p gqlProject) toProject() Project {
	kind := "user"
	if p.Owner.Typename == "Organization" {
		kind = "organization"
	}
	return Project{
		ID: p.ID, Number: p.Number, Title: p.Title, Owner: p.Owner.Login,
		OwnerKind: kind, URL: p.URL, Closed: p.Closed, ItemsTotal: p.Items.TotalCount,
	}
}

type gqlOption struct {
	ID          string
	Name        string
	Color       string
	Description string
}

type gqlIteration struct {
	ID        string
	Title     string
	StartDate string
	Duration  int
}

type gqlField struct {
	ID            string
	Name          string
	DataType      string
	Options       []gqlOption
	Configuration struct {
		Iterations          []gqlIteration
		CompletedIterations []gqlIteration
	}
}

func (f gqlField) toField() Field {
	out := Field{ID: f.ID, Name: f.Name, DataType: f.DataType}
	for _, o := range f.Options {
		out.Options = append(out.Options, Option(o))
	}
	// Current and upcoming first: completed iterations come last and
	// are typically not pickable targets.
	for _, it := range append(f.Configuration.Iterations, f.Configuration.CompletedIterations...) {
		start, _ := time.Parse("2006-01-02", it.StartDate)
		out.Iterations = append(out.Iterations, Iteration{ID: it.ID, Title: it.Title, StartDate: start, DurationDays: it.Duration})
	}
	return out
}

type gqlFieldValue struct {
	Typename    string `json:"__typename"`
	Text        string
	Number      *float64
	Date        string
	Name        string
	OptionID    string
	IterationID string
	Title       string
	Field       struct{ ID string }
	Users       struct{ Nodes []struct{ Login string } }
	Labels      struct{ Nodes []struct{ Name string } }
}

type gqlItem struct {
	ID         string
	DatabaseID int64
	Type       string
	IsArchived bool
	Content    struct {
		Typename   string `json:"__typename"`
		ID         string
		Number     int
		Title      string
		URL        string
		Body       string
		Assignees  struct{ Nodes []struct{ Login string } }
		Repository struct{ NameWithOwner string }
	}
	FieldValues struct{ Nodes []gqlFieldValue }
}

func (g gqlItem) toItem(fields []Field) Item {
	kind := KindDraft
	switch g.Content.Typename {
	case "Issue":
		kind = KindIssue
	case "PullRequest":
		kind = KindPR
	}
	it := Item{
		ID: g.ID, ContentID: g.Content.ID, DatabaseID: g.DatabaseID, Kind: kind,
		Title: g.Content.Title, URL: g.Content.URL, Body: g.Content.Body,
		Archived: g.IsArchived,
	}
	if kind != KindDraft {
		it.Repo = g.Content.Repository.NameWithOwner
		it.Number = g.Content.Number
	}
	byID := make(map[string]Field, len(fields))
	for _, f := range fields {
		byID[f.ID] = f
	}
	for _, v := range g.FieldValues.Nodes {
		f, ok := byID[v.Field.ID]
		if !ok {
			continue
		}
		fv := FieldValue{FieldID: f.ID, FieldName: f.Name, DataType: f.DataType}
		switch v.Typename {
		case "ProjectV2ItemFieldTextValue", "ProjectV2ItemFieldLabelValue":
			if v.Text != "" {
				fv.Text = v.Text
			} else {
				var names []string
				for _, l := range v.Labels.Nodes {
					names = append(names, l.Name)
				}
				fv.Text = strings.Join(names, ", ")
			}
		case "ProjectV2ItemFieldNumberValue":
			fv.Number = v.Number
		case "ProjectV2ItemFieldDateValue":
			if d, err := time.Parse("2006-01-02", v.Date); err == nil {
				fv.Date = &d
			}
		case "ProjectV2ItemFieldSingleSelectValue":
			fv.OptionID, fv.OptionName = v.OptionID, v.Name
		case "ProjectV2ItemFieldIterationValue":
			fv.IterationID, fv.IterationTitle = v.IterationID, v.Title
		case "ProjectV2ItemFieldUserValue":
			for _, u := range v.Users.Nodes {
				fv.Users = append(fv.Users, u.Login)
			}
		default:
			continue
		}
		it.Values = append(it.Values, fv)
	}
	// Assignees come from the content for linked items and drafts.
	if len(g.Content.Assignees.Nodes) > 0 {
		var logins []string
		for _, u := range g.Content.Assignees.Nodes {
			logins = append(logins, u.Login)
		}
		found := false
		for i, v := range it.Values {
			if v.DataType == "ASSIGNEES" {
				it.Values[i].Users = logins
				found = true
			}
		}
		if !found {
			it.Values = append(it.Values, FieldValue{FieldName: "Assignees", DataType: "ASSIGNEES", Users: logins})
		}
	}
	return it
}

// ---- Read operations -----------------------------------------------------

const viewerProjectsQuery = `
query BoardwalkViewerProjects {
  viewer {
    login
    projectsV2(first: 100) { nodes { ...ProjFields } }
    organizations(first: 20) { nodes { login projectsV2(first: 100) { nodes { ...ProjFields } } } }
  }
}` + "\n" + projectFieldsFragment

type projectsResponse struct {
	Viewer struct {
		Login         string
		ProjectsV2    struct{ Nodes []gqlProject }
		Organizations struct {
			Nodes []struct {
				Login      string
				ProjectsV2 struct{ Nodes []gqlProject }
			}
		}
	}
	User struct {
		ProjectsV2 struct{ Nodes []gqlProject }
	}
}

func (c *gqlClient) ListProjects(ctx context.Context, ownerLogin string) ([]Project, error) {
	if err := c.CheckAuth(ctx); err != nil {
		return nil, err
	}
	var resp projectsResponse
	if ownerLogin != "" {
		// Try user first, then organisation.
		var u struct {
			User *struct {
				ProjectsV2 struct{ Nodes []gqlProject }
			}
		}
		q := `query($l:String!){ user(login:$l){ projectsV2(first:100){ nodes { ...ProjFields } } } }` + "\n" + projectFieldsFragment
		if err := c.do(ctx, q, map[string]interface{}{"l": ownerLogin}, &u); err == nil && u.User != nil {
			return projects(u.User.ProjectsV2.Nodes), nil
		}
		var o struct {
			Organization *struct {
				ProjectsV2 struct{ Nodes []gqlProject }
			}
		}
		qo := `query($l:String!){ organization(login:$l){ projectsV2(first:100){ nodes { ...ProjFields } } } }` + "\n" + projectFieldsFragment
		if err := c.do(ctx, qo, map[string]interface{}{"l": ownerLogin}, &o); err != nil {
			return nil, err
		} else if o.Organization != nil {
			return projects(o.Organization.ProjectsV2.Nodes), nil
		}
		return nil, exiterr.NotFoundf("no user or organisation named %q", ownerLogin)
	}
	if err := c.do(ctx, viewerProjectsQuery, map[string]interface{}{}, &resp); err != nil {
		return nil, err
	}
	out := projects(resp.Viewer.ProjectsV2.Nodes)
	for _, org := range resp.Viewer.Organizations.Nodes {
		out = append(out, projects(org.ProjectsV2.Nodes)...)
	}
	return out, nil
}

func projects(nodes []gqlProject) []Project {
	out := make([]Project, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.toProject())
	}
	return out
}

func parseProjectRef(ref string) (owner string, number int, err error) {
	// Accept https://github.com/users/<login>/projects/N and /orgs/<login>/projects/N.
	if strings.HasPrefix(ref, "http") {
		parts := strings.Split(strings.Trim(ref, "/"), "/")
		// .../users|orgs/<login>/projects/N
		for i, p := range parts {
			if p == "projects" && i >= 1 && i+1 < len(parts) {
				n, e := strconvAtoi(parts[i+1])
				if e != nil {
					break
				}
				return parts[i-1], n, nil
			}
		}
		return "", 0, exiterr.Usagef("cannot parse project URL %q (want .../projects/<number>)", ref)
	}
	parts := strings.Split(ref, "/")
	if len(parts) != 2 {
		return "", 0, exiterr.Usagef("project reference %q must be owner/number or a project URL", ref)
	}
	n, e := strconvAtoi(parts[1])
	if e != nil {
		return "", 0, exiterr.Usagef("project reference %q: number %q is not numeric", ref, parts[1])
	}
	return parts[0], n, nil
}

func strconvAtoi(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return 0, err
	}
	// Reject trailing junk like "3abc".
	if fmt.Sprint(n) != strings.TrimSpace(s) {
		return 0, fmt.Errorf("not a number: %q", s)
	}
	return n, nil
}

func (c *gqlClient) ResolveProject(ctx context.Context, ref string) (Project, error) {
	if err := c.CheckAuth(ctx); err != nil {
		return Project{}, err
	}
	owner, number, err := parseProjectRef(ref)
	if err != nil {
		return Project{}, err
	}
	vars := map[string]interface{}{"l": owner, "n": number}
	var u struct {
		User *struct{ ProjectV2 *gqlProject }
	}
	q := `query($l:String!,$n:Int!){ user(login:$l){ projectV2(number:$n){ ...ProjFields } } }` + "\n" + projectFieldsFragment
	errUser := c.do(ctx, q, vars, &u)
	if errUser == nil && u.User != nil && u.User.ProjectV2 != nil {
		return u.User.ProjectV2.toProject(), nil
	}
	if errUser != nil && strings.Contains(errUser.Error(), "ProjectV2") {
		// Partial-data NOT_FOUND: the owner exists, the project does not.
		return Project{}, exiterr.NotFoundf("no project %s/%d is accessible to your gh account", owner, number)
	}
	var o struct {
		Organization *struct{ ProjectV2 *gqlProject }
	}
	qo := `query($l:String!,$n:Int!){ organization(login:$l){ projectV2(number:$n){ ...ProjFields } } }` + "\n" + projectFieldsFragment
	if err := c.do(ctx, qo, vars, &o); err != nil {
		return Project{}, err
	} else if o.Organization != nil && o.Organization.ProjectV2 != nil {
		return o.Organization.ProjectV2.toProject(), nil
	}
	return Project{}, exiterr.NotFoundf("no project %s/%d is accessible to your gh account", owner, number)
}

const boardPageQuery = `
query BoardwalkBoard($id: ID!, $size: Int!, $cursor: String) {
  node(id: $id) {
    ... on ProjectV2 {
      ...BoardFields
      items(first: $size, after: $cursor) {
        pageInfo { hasNextPage endCursor }
        nodes { ...BoardItem }
      }
    }
  }
}` + "\n" + fieldsFragment + "\n" + itemFragment

type boardPageResponse struct {
	Node *struct {
		Fields struct{ Nodes []gqlField }
		Items  struct {
			PageInfo struct {
				HasNextPage bool
				EndCursor   string
			}
			Nodes []gqlItem
		}
	}
}

func (c *gqlClient) LoadBoardPage(ctx context.Context, projectID, cursor string, pageSize int, _ string) ([]Field, []Item, string, error) {
	if err := c.CheckAuth(ctx); err != nil {
		return nil, nil, "", err
	}
	vars := map[string]interface{}{"id": projectID, "size": pageSize}
	if cursor != "" {
		vars["cursor"] = cursor
	}
	var resp boardPageResponse
	if err := c.do(ctx, boardPageQuery, vars, &resp); err != nil {
		return nil, nil, "", err
	}
	if resp.Node == nil {
		return nil, nil, "", exiterr.NotFoundf("project %s not found or not accessible", projectID)
	}
	fields := make([]Field, 0, len(resp.Node.Fields.Nodes))
	for _, f := range resp.Node.Fields.Nodes {
		fields = append(fields, f.toField())
	}
	items := make([]Item, 0, len(resp.Node.Items.Nodes))
	for _, g := range resp.Node.Items.Nodes {
		items = append(items, g.toItem(fields))
	}
	next := ""
	if resp.Node.Items.PageInfo.HasNextPage {
		next = resp.Node.Items.PageInfo.EndCursor
	}
	return fields, items, next, nil
}

const itemDetailQuery = `
query BoardwalkItem($id: ID!) {
  node(id: $id) { ...BoardItem }
}` + "\n" + itemFragment

func (c *gqlClient) LoadItemDetail(ctx context.Context, itemID string) (Item, error) {
	if err := c.CheckAuth(ctx); err != nil {
		return Item{}, err
	}
	var resp struct{ Node *gqlItem }
	if err := c.do(ctx, itemDetailQuery, map[string]interface{}{"id": itemID}, &resp); err != nil {
		return Item{}, err
	}
	if resp.Node == nil {
		return Item{}, exiterr.NotFoundf("item %s not found", itemID)
	}
	// Fields metadata is not part of the item payload; values keep IDs
	// and names resolved by callers against the board schema when
	// needed. Detail callers fetch the board schema via LoadBoardPage.
	return resp.Node.toItem(nil), nil
}

// ---- Mutations: implemented in step 4 (mutations.go) --------------------
