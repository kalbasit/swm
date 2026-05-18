// Package layout computes on-disk paths from swm's domain model.
// The host — not plugins — owns all path composition. Plugins return
// ProjectID(host, segments[]) and the Resolver derives every filesystem path.
package layout

import (
	"path/filepath"
	"strings"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

// Resolver computes canonical clone and worktree paths from a code root.
type Resolver struct {
	codeRoot         string
	defaultStoryName string
}

// NewResolver returns a Resolver anchored at the given code root directory.
// defaultStoryName is the story that maps to the canonical repositories/ path
// rather than a git worktree (typically "_default").
func NewResolver(codeRoot, defaultStoryName string) *Resolver {
	return &Resolver{codeRoot: codeRoot, defaultStoryName: defaultStoryName}
}

// CanonicalPath returns <code_root>/repositories/<host>/<seg1>/.../<segN>.
func (r *Resolver) CanonicalPath(id *pluginv1.ProjectID) string {
	parts := append([]string{r.codeRoot, "repositories", id.Host}, id.Segments...)

	return filepath.Join(parts...)
}

// CodeRoot returns the code root directory this resolver is anchored to.
func (r *Resolver) CodeRoot() string {
	return r.codeRoot
}

// ProjectIDFromPath derives a ProjectID from a path under repositories/.
// For example: "/code/repositories/github.com/kalbasit/swm" -> {host:"github.com", segments:["kalbasit","swm"]}.
// Returns nil if the path is not under <code_root>/repositories/.
func (r *Resolver) ProjectIDFromPath(path string) *pluginv1.ProjectID {
	base := filepath.Join(r.codeRoot, "repositories") + string(filepath.Separator)

	if !strings.HasPrefix(path, base) {
		return nil
	}

	rel := strings.TrimPrefix(path, base)
	parts := strings.Split(rel, string(filepath.Separator))

	if len(parts) < 2 { //nolint:mnd // need at least host + one segment
		return nil
	}

	return &pluginv1.ProjectID{
		Host:     parts[0],
		Segments: parts[1:],
	}
}

// WorktreePath returns the filesystem path for a project within a story.
// For the default story the project lives at the canonical repositories/ path,
// not inside a git worktree under stories/.
func (r *Resolver) WorktreePath(storyName string, id *pluginv1.ProjectID) string {
	if id == nil {
		return ""
	}

	if storyName == r.defaultStoryName {
		return r.CanonicalPath(id)
	}

	parts := append([]string{r.codeRoot, "stories", storyName, id.Host}, id.Segments...)

	return filepath.Join(parts...)
}
