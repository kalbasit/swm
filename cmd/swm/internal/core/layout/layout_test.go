package layout_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
)

func TestCanonicalPath(t *testing.T) {
	t.Parallel()

	githubSWM := &pluginv1.ProjectID{Host: "github.com", Segments: []string{"kalbasit", "swm"}}
	gitlabDeep := &pluginv1.ProjectID{Host: "gitlab.com", Segments: []string{"foo", "bar", "baz"}}
	keybaseProject := &pluginv1.ProjectID{Host: "keybase", Segments: []string{"team", "stowix.infra", "fly-secrets"}}

	r := layout.NewResolver("/home/user/code", "_default")

	tests := []struct {
		name     string
		project  *pluginv1.ProjectID
		expected string
	}{
		{
			name:     "github two-segment",
			project:  githubSWM,
			expected: "/home/user/code/repositories/github.com/kalbasit/swm",
		},
		{
			name:     "gitlab three-segment",
			project:  gitlabDeep,
			expected: "/home/user/code/repositories/gitlab.com/foo/bar/baz",
		},
		{
			name:     "keybase multi-segment",
			project:  keybaseProject,
			expected: "/home/user/code/repositories/keybase/team/stowix.infra/fly-secrets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, r.CanonicalPath(tt.project))
		})
	}
}

func TestWorktreePath(t *testing.T) {
	t.Parallel()

	githubSWM := &pluginv1.ProjectID{Host: "github.com", Segments: []string{"kalbasit", "swm"}}
	gitlabDeep := &pluginv1.ProjectID{Host: "gitlab.com", Segments: []string{"foo", "bar", "baz"}}

	r := layout.NewResolver("/home/user/code", "_default")

	tests := []struct {
		name      string
		storyName string
		project   *pluginv1.ProjectID
		expected  string
	}{
		{
			name:      "github story",
			storyName: "feat-x",
			project:   githubSWM,
			expected:  "/home/user/code/stories/feat-x/github.com/kalbasit/swm",
		},
		{
			name:      "gitlab deep story",
			storyName: "fix-y",
			project:   gitlabDeep,
			expected:  "/home/user/code/stories/fix-y/gitlab.com/foo/bar/baz",
		},
		{
			name:      "default story uses canonical path",
			storyName: "_default",
			project:   githubSWM,
			expected:  "/home/user/code/repositories/github.com/kalbasit/swm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, r.WorktreePath(tt.storyName, tt.project))
		})
	}
}
