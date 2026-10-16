package core

import (
	"strconv"
	"strings"

	"github.com/juanrubio/boardwalk/internal/exiterr"
	"github.com/juanrubio/boardwalk/internal/githubx"
)

// ResolveItemRef resolves a cli.md Section 1 item reference within a
// loaded board. Not found exits 4; ambiguity exits 2 and changes
// nothing. Short IDs (draft:<databaseId>) are exact and cannot be
// ambiguous.
func ResolveItemRef(board *githubx.Board, ref string) (githubx.Item, error) {
	if ref == "" {
		return githubx.Item{}, exiterr.Usagef("empty item reference")
	}
	// draft:<databaseId>
	if strings.HasPrefix(ref, "draft:") {
		idStr := strings.TrimPrefix(ref, "draft:")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err == nil {
			for _, it := range board.Items {
				if it.DatabaseID == id {
					return it, nil
				}
			}
		}
		return githubx.Item{}, exiterr.NotFoundf("no item %q on project %s", ref, board.Project.Ref())
	}
	// Full URL: match by suffix number or contained reference.
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		if it, ok := matchURL(board, ref); ok {
			return it, nil
		}
		return githubx.Item{}, exiterr.NotFoundf("no item for URL %q on project %s", ref, board.Project.Ref())
	}
	// owner/repo#number
	if strings.Contains(ref, "#") {
		for _, it := range board.Items {
			if it.Kind != githubx.KindDraft && it.Ref() == ref {
				return it, nil
			}
		}
		return githubx.Item{}, exiterr.NotFoundf("no item %q on project %s (it may exist on GitHub without being on this project)", ref, board.Project.Ref())
	}
	// Full node ID
	for _, it := range board.Items {
		if it.ID == ref {
			return it, nil
		}
	}
	// Exact title (convenience; must be unique)
	var matches []githubx.Item
	for _, it := range board.Items {
		if it.Title == ref {
			matches = append(matches, it)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return githubx.Item{}, exiterr.NotFoundf("no item %q on project %s", ref, board.Project.Ref())
	default:
		var refs []string
		for _, m := range matches {
			refs = append(refs, m.Ref()+" ("+string(m.Kind)+")")
		}
		return githubx.Item{}, exiterr.Usagef("item title %q is ambiguous on project %s: %s — use a reference instead",
			ref, board.Project.Ref(), strings.Join(refs, ", "))
	}
}

func matchURL(board *githubx.Board, url string) (githubx.Item, bool) {
	// Accept .../issues/N or .../pull/N URLs.
	for _, kind := range []string{"/issues/", "/pull/"} {
		if idx := strings.LastIndex(url, kind); idx >= 0 {
			numStr := strings.Trim(url[idx+len(kind):], "/")
			if n, err := strconv.Atoi(numStr); err == nil {
				repo := repoFromURL(url[:idx])
				for _, it := range board.Items {
					if it.Number == n && it.Repo == repo {
						return it, true
					}
				}
			}
		}
	}
	for _, it := range board.Items {
		if it.URL != "" && it.URL == url {
			return it, true
		}
	}
	return githubx.Item{}, false
}

func repoFromURL(prefix string) string {
	// https://github.com/owner/repo
	parts := strings.Split(strings.Trim(prefix, "/"), "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return ""
}
