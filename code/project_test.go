package code

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectPath(t *testing.T) {
	// create a new project
	p := &Project{
		ImportPath:  "github.com/kalbasit/swm",
		CodePath:    "/home/kalbasit/code",
		ProfileName: "personal",
		StoryName:   "base",
	}
	// assert the Path
	assert.Equal(t, "/home/kalbasit/code/personal/base/src/github.com/kalbasit/swm", p.Path())
}

func TestProjectSessionName(t *testing.T) {
	// create a new project
	p := &Project{
		ImportPath:  "github.com/kalbasit/swm",
		CodePath:    "/home/kalbasit/code",
		ProfileName: "personal",
		StoryName:   "base",
	}
	// assert the Path
	assert.Equal(t, "personal@base=github\u2022com/kalbasit/swm", p.SessionName())
}

func TestBaseProject(t *testing.T) {
	assert.True(t, (&Project{StoryName: BaseStory}).Base())
	assert.False(t, (&Project{StoryName: "STORY-123"}).Base())
}
