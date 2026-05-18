package story_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/story"
	"github.com/kalbasit/swm/cmd/swm/internal/config"
)

func TestBranchFromTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tpl       string
		storyName string
		want      string
		wantErr   bool
	}{
		{
			name:      "default template",
			tpl:       config.DefaultBranchNameTemplate,
			storyName: testStoryName,
			want:      "feat/" + testStoryName,
		},
		{
			name:      "custom prefix template",
			tpl:       "fix/{{.Name}}",
			storyName: testBugName,
			want:      "fix/" + testBugName,
		},
		{
			name:      "template with no prefix",
			tpl:       "{{.Name}}",
			storyName: testStoryName,
			want:      testStoryName,
		},
		{
			name:      "template with suffix",
			tpl:       "{{.Name}}/wip",
			storyName: "my-story",
			want:      "my-story/wip",
		},
		{
			name:      "invalid template syntax",
			tpl:       "{{.Name",
			storyName: testStoryName,
			wantErr:   true,
		},
		{
			name:      "empty template uses default feat prefix",
			tpl:       "",
			storyName: testStoryName,
			want:      "feat/" + testStoryName, // falls back to config.DefaultBranchNameTemplate
		},
		{
			name:      "template that evaluates to empty string",
			tpl:       "{{if false}}{{.Name}}{{end}}",
			storyName: testStoryName,
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := story.BranchFromTemplate(tc.tpl, tc.storyName)
			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
