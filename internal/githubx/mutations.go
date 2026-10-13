package githubx

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	ghapi "github.com/cli/go-gh/v2/pkg/api"

	"github.com/juanrubio/gh-boardwalk/internal/exiterr"
)

// ---- Content resolution --------------------------------------------------

func parseIssueRef(ref string) (repo string, number int, isPR bool, err error) {
	s := ref
	if strings.HasPrefix(s, "http") {
		for _, marker := range []string{"/issues/", "/pull/"} {
			if idx := strings.LastIndex(s, marker); idx >= 0 {
				n, e := strconv.Atoi(strings.Trim(s[idx+len(marker):], "/"))
				if e != nil {
					break
				}
				parts := strings.Split(strings.Trim(s[:idx], "/"), "/")
				if len(parts) >= 2 {
					return parts[len(parts)-2] + "/" + parts[len(parts)-1], n, marker == "/pull/", nil
				}
			}
		}
		return "", 0, false, exiterr.Usagef("cannot parse issue/PR URL %q", ref)
	}
	hash := strings.LastIndex(s, "#")
	if hash < 0 {
		return "", 0, false, exiterr.Usagef("content reference %q must be owner/repo#number or an issue/PR URL", ref)
	}
	n, e := strconv.Atoi(s[hash+1:])
	if e != nil {
		return "", 0, false, exiterr.Usagef("content reference %q: number is not numeric", ref)
	}
	return s[:hash], n, false, nil
}

const resolveContentQuery = `
query BoardwalkContent($owner: String!, $name: String!, $num: Int!) {
  repository(owner: $owner, name: $name) {
    issueOrPullRequest(number: $num) {
      __typename
      ... on Issue { id number title url }
      ... on PullRequest { id number title url }
    }
  }
}`

func (c *gqlClient) ResolveContent(ctx context.Context, ref string) (string, Item, error) {
	if err := c.CheckAuth(ctx); err != nil {
		return "", Item{}, err
	}
	repo, number, _, err := parseIssueRef(ref)
	if err != nil {
		return "", Item{}, err
	}
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return "", Item{}, exiterr.Usagef("content reference %q: repository must be owner/repo", ref)
	}
	var resp struct {
		Repository *struct {
			IssueOrPullRequest *struct {
				Typename string `json:"__typename"`
				ID       string
				Number   int
				Title    string
				URL      string
			}
		}
	}
	vars := map[string]interface{}{"owner": parts[0], "name": parts[1], "num": number}
	if err := c.do(ctx, resolveContentQuery, vars, &resp); err != nil {
		return "", Item{}, err
	}
	if resp.Repository == nil || resp.Repository.IssueOrPullRequest == nil {
		return "", Item{}, exiterr.NotFoundf("no issue or pull request %s", ref)
	}
	n := resp.Repository.IssueOrPullRequest
	kind := KindIssue
	if n.Typename == "PullRequest" {
		kind = KindPR
	}
	return n.ID, Item{Kind: kind, Repo: repo, Number: n.Number, Title: n.Title, URL: n.URL}, nil
}

// ---- Add / draft ---------------------------------------------------------

const addItemMutation = `
mutation BoardwalkAdd($project: ID!, $content: ID!) {
  addProjectV2ItemById(input: {projectId: $project, contentId: $content}) {
    item { id databaseId type }
  }
}`

func (c *gqlClient) AddItem(ctx context.Context, projectID, contentID string) (Item, error) {
	if err := c.CheckAuth(ctx); err != nil {
		return Item{}, err
	}
	var resp struct {
		AddProjectV2ItemById struct {
			Item struct {
				ID         string
				DatabaseID int64
				Type       string
			}
		}
	}
	vars := map[string]interface{}{"project": projectID, "content": contentID}
	if err := c.do(ctx, addItemMutation, vars, &resp); err != nil {
		return Item{}, err
	}
	return c.LoadItemDetail(ctx, resp.AddProjectV2ItemById.Item.ID)
}

const createDraftMutation = `
mutation BoardwalkDraft($project: ID!, $title: String!, $body: String) {
  addProjectV2DraftIssue(input: {projectId: $project, title: $title, body: $body}) {
    projectItem { id databaseId type }
  }
}`

func (c *gqlClient) CreateDraft(ctx context.Context, projectID, title, body string) (Item, error) {
	if err := c.CheckAuth(ctx); err != nil {
		return Item{}, err
	}
	var resp struct {
		AddProjectV2DraftIssue struct {
			ProjectItem struct {
				ID         string
				DatabaseID int64
				Type       string
			}
		}
	}
	vars := map[string]interface{}{"project": projectID, "title": title}
	if body != "" {
		vars["body"] = body
	}
	if err := c.do(ctx, createDraftMutation, vars, &resp); err != nil {
		return Item{}, err
	}
	return Item{
		ID:         resp.AddProjectV2DraftIssue.ProjectItem.ID,
		DatabaseID: resp.AddProjectV2DraftIssue.ProjectItem.DatabaseID,
		Kind:       KindDraft, Title: title, Body: body,
	}, nil
}

// ---- Batched field save (one request) -------------------------------------

func valueInput(v MutationValue) map[string]interface{} {
	switch {
	case v.Clear:
		return map[string]interface{}{} // handled by caller via clear semantics below
	case v.OptionID != nil:
		return map[string]interface{}{"singleSelectOptionId": *v.OptionID}
	case v.Text != nil:
		return map[string]interface{}{"text": *v.Text}
	case v.Number != nil:
		return map[string]interface{}{"number": *v.Number}
	case v.Date != nil:
		return map[string]interface{}{"date": *v.Date}
	case v.IterationID != nil:
		return map[string]interface{}{"iterationId": *v.IterationID}
	default:
		return map[string]interface{}{}
	}
}

func (c *gqlClient) SetFieldValues(ctx context.Context, projectID, itemID string, assigns []FieldAssign) []FieldResult {
	results := make([]FieldResult, len(assigns))
	if err := c.CheckAuth(ctx); err != nil {
		for i, a := range assigns {
			results[i] = FieldResult{FieldName: a.FieldName, Err: err}
		}
		return results
	}
	if len(assigns) == 0 {
		return results
	}
	var b strings.Builder
	b.WriteString("mutation BoardwalkSave($project: ID!, $item: ID!")
	vars := map[string]interface{}{"project": projectID, "item": itemID}
	for i, a := range assigns {
		fmt.Fprintf(&b, ", $f%d: ID!, $v%d: ProjectV2FieldValue!", i, i)
		vars[fmt.Sprintf("f%d", i)] = a.FieldID
		vars[fmt.Sprintf("v%d", i)] = valueInput(a.Value)
	}
	b.WriteString(") {\n")
	for i := range assigns {
		fmt.Fprintf(&b, "  f%d: updateProjectV2ItemFieldValue(input: {projectId: $project, itemId: $item, fieldId: $f%d, value: $v%d}) { projectV2Item { id } }\n", i, i, i)
	}
	b.WriteString("}")
	resp := map[string]struct {
		ProjectV2Item struct{ ID string }
	}{}
	gql, err := c.gql()
	if err != nil {
		for i, a := range assigns {
			results[i] = FieldResult{FieldName: a.FieldName, Err: err}
		}
		return results
	}
	err = gql.DoWithContext(ctx, b.String(), vars, &resp)
	if err == nil {
		for i, a := range assigns {
			results[i] = FieldResult{FieldName: a.FieldName}
		}
		return results
	}
	// Partial failure: map typed GraphQL errors by alias path.
	var gqlErr *ghapi.GraphQLError
	if errors.As(err, &gqlErr) {
		failed := map[string]error{}
		for _, item := range gqlErr.Errors {
			if len(item.Path) > 0 {
				if alias, ok := item.Path[0].(string); ok {
					failed[alias] = classify(errors.New(item.Message))
				}
			}
		}
		if len(failed) > 0 {
			for i, a := range assigns {
				alias := fmt.Sprintf("f%d", i)
				results[i] = FieldResult{FieldName: a.FieldName, Err: failed[alias]}
			}
			return results
		}
	}
	classified := classify(err)
	for i, a := range assigns {
		results[i] = FieldResult{FieldName: a.FieldName, Err: classified}
	}
	return results
}

// ---- Title/body, archive ---------------------------------------------------

func (c *gqlClient) UpdateTitleBody(ctx context.Context, item Item, title, body *string) (Item, error) {
	if err := c.CheckAuth(ctx); err != nil {
		return Item{}, err
	}
	out := item
	if item.Kind == KindDraft {
		var sb strings.Builder
		sb.WriteString("mutation BoardwalkDraftEdit($id: ID!")
		vars := map[string]interface{}{"id": item.ID}
		if title != nil {
			sb.WriteString(", $title: String!")
			vars["title"] = *title
		}
		if body != nil {
			sb.WriteString(", $body: String!")
			vars["body"] = *body
		}
		sb.WriteString(") { updateProjectV2DraftIssue(input: {draftIssueId: $id")
		if title != nil {
			sb.WriteString(", title: $title")
		}
		if body != nil {
			sb.WriteString(", body: $body")
		}
		sb.WriteString("}) { draftIssue { id } } }")
		var resp struct {
			UpdateProjectV2DraftIssue struct {
				DraftIssue struct{ ID string }
			}
		}
		if err := c.do(ctx, sb.String(), vars, &resp); err != nil {
			return Item{}, err
		}
	} else {
		// Linked issue/PR title/body edits go through their own mutations.
		kind := "updateIssue"
		if item.Kind == KindPR {
			kind = "updatePullRequest"
		}
		// The content node ID is not the item ID; resolve via the item's
		// project membership is ambiguous, so use the URL-derived ref.
		contentID, _, err := c.ResolveContent(ctx, item.Ref())
		if err != nil {
			return Item{}, err
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "mutation BoardwalkContentEdit($id: ID!")
		vars := map[string]interface{}{"id": contentID}
		if title != nil {
			sb.WriteString(", $title: String!")
			vars["title"] = *title
		}
		if body != nil {
			sb.WriteString(", $body: String!")
			vars["body"] = *body
		}
		fmt.Fprintf(&sb, ") { %s(input: {id: $id", kind)
		if title != nil {
			sb.WriteString(", title: $title")
		}
		if body != nil {
			sb.WriteString(", body: $body")
		}
		if item.Kind == KindPR {
			sb.WriteString("}) { pullRequest { id } } }")
		} else {
			sb.WriteString("}) { issue { id } } }")
		}
		var resp map[string]any
		if err := c.do(ctx, sb.String(), vars, &resp); err != nil {
			return Item{}, err
		}
	}
	if title != nil {
		out.Title = *title
	}
	if body != nil {
		out.Body = *body
	}
	return out, nil
}

const archiveMutation = `
mutation BoardwalkArchive($project: ID!, $item: ID!) {
  archiveProjectV2Item(input: {projectId: $project, itemId: $item}) {
    item { id isArchived }
  }
}`

const deleteMutation = `
mutation BoardwalkDelete($project: ID!, $item: ID!) {
  deleteProjectV2Item(input: {projectId: $project, itemId: $item}) {
    deletedItemId
  }
}`

func (c *gqlClient) DeleteItem(ctx context.Context, projectID, itemID string) error {
	if err := c.CheckAuth(ctx); err != nil {
		return err
	}
	var resp struct {
		DeleteProjectV2Item struct {
			DeletedItemID string
		}
	}
	return c.do(ctx, deleteMutation, map[string]interface{}{"project": projectID, "item": itemID}, &resp)
}

func (c *gqlClient) ArchiveItem(ctx context.Context, projectID, itemID string) error {
	if err := c.CheckAuth(ctx); err != nil {
		return err
	}
	var resp struct {
		ArchiveProjectV2Item struct {
			Item struct {
				ID         string
				IsArchived bool
			}
		}
	}
	return c.do(ctx, archiveMutation, map[string]interface{}{"project": projectID, "item": itemID}, &resp)
}
