package code

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectPath(t *testing.T) {
	// create a new project
	p := &project{
		story: &story{
			name: "base",
			profile: &profile{
				name: "personal",
				code: &code{
					path: "/home/kalbasit/code",
				},
			},
		},

		importPath: "github.com/kalbasit/swm",
	}
	// assert the Path
	assert.Equal(t, "/home/kalbasit/code/personal/base/src/github.com/kalbasit/swm", p.Path())
}

func TestProjectSessionName(t *testing.T) {
	// create a new project
	p := &project{
		story: &story{
			name: "base",
			profile: &profile{
				name: "personal",
				code: &code{
					path: "/home/kalbasit/code",
				},
			},
		},

		importPath: "github.com/kalbasit/swm",
	}
	// assert the Path
	assert.Equal(t, "personal@base=github\u2022com/kalbasit/swm", p.SessionName())
}

func TestBaseProject(t *testing.T) {
	assert.True(t, (&project{story: &story{name: baseStoryName}}).Base())
	assert.False(t, (&project{story: &story{name: "STORY-123"}}).Base())
}
