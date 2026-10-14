package githubx

import "context"

// Client is the seam between Boardwalk's core and GitHub. The real
// implementation performs GraphQL over go-gh credentials (ADR 0002);
// tests drive the same interface from captured fixtures (ADR 0004).
// All failures are classified with internal/exiterr codes.
type Client interface {
	// CheckAuth verifies gh presence, authentication, and the upfront
	// project+repo scopes (ADR 0010). It is called once per process
	// before the first GitHub-backed operation.
	CheckAuth(ctx context.Context) error

	// ListProjects returns Projects v2 owned by ownerLogin (or the
	// viewer when empty), including organisations the viewer belongs to.
	ListProjects(ctx context.Context, ownerLogin string) ([]Project, error)

	// ResolveProject resolves owner/number or a project URL to a project.
	ResolveProject(ctx context.Context, ref string) (Project, error)

	// LoadBoardPage returns the project schema plus one page of items
	// in a single round trip (performance.md: schema + first page
	// combined). Pass cursor "" for the first page. query is an optional
	// GitHub project filter expression (server-side filtering, B9).
	LoadBoardPage(ctx context.Context, projectID string, cursor string, pageSize int, query string) (fields []Field, items []Item, nextCursor string, err error)

	// ItemByRef resolves a reference within a project board already
	// loaded in items, per cli.md Section 1 semantics implemented by
	// the core package (this client only fetches).
	LoadItemDetail(ctx context.Context, itemID string) (Item, error)

	// AddItem adds an existing issue/PR (by its node ID) to a project.
	AddItem(ctx context.Context, projectID, contentID string) (Item, error)

	// CreateDraft creates a draft item and returns it as displayed.
	CreateDraft(ctx context.Context, projectID, title, body string) (Item, error)

	// ResolveContent resolves owner/repo#number or an issue/PR URL to
	// its node ID and display data, from the issue side (O(1), never a
	// board scan; cli.md Section 1).
	ResolveContent(ctx context.Context, ref string) (contentID string, item Item, err error)

	// SetFieldValues applies several field assignments to one item as
	// ONE GraphQL request (performance.md: a save is one round trip)
	// and reports per-field results; a save is not atomic (tui.md).
	SetFieldValues(ctx context.Context, projectID, itemID string, assigns []FieldAssign) []FieldResult

	// UpdateTitleBody edits a linked issue/PR or draft title/body where
	// GitHub permits it, returning the updated item.
	UpdateTitleBody(ctx context.Context, item Item, title, body *string) (Item, error)

	// ArchiveItem archives a project item.
	ArchiveItem(ctx context.Context, projectID, itemID string) error

	// DeleteItem permanently deletes a project item (ADR 0012). A
	// linked issue/PR is untouched; a draft's content is gone.
	DeleteItem(ctx context.Context, projectID, itemID string) error

	// ConvertDraftToIssue converts a draft item into a real issue in
	// repo (owner/name), keeping board position and field values (A7).
	ConvertDraftToIssue(ctx context.Context, projectID, itemID, repo string) (Item, error)
}

// FieldAssign is one named field assignment in a batch save.
type FieldAssign struct {
	FieldID   string
	FieldName string
	Value     MutationValue
}

// FieldResult is the per-field outcome of a batch save.
type FieldResult struct {
	FieldName string
	Err       error // nil when GitHub applied the change
}

// MutationValue is one typed field assignment.
type MutationValue struct {
	Text        *string
	Number      *float64
	Date        *string // YYYY-MM-DD
	OptionID    *string
	IterationID *string
	Clear       bool
}
