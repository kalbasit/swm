package tmx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectPath(t *testing.T) {
	// create a new project
	p := &Project{
		ImportPath:    "github.com/kalbasit/tmx",
		CodePath:      "/home/kalbasit/code",
		ProfileName:   "personal",
		WorkspaceName: "base",
	}
	// assert the Path
	assert.Equal(t, "/home/kalbasit/code/personal/base/src/github.com/kalbasit/tmx", p.Path())
}

func TestProjectSessionName(t *testing.T) {
	// create a new project
	p := &Project{
		ImportPath:    "github.com/kalbasit/tmx",
		CodePath:      "/home/kalbasit/code",
		ProfileName:   "personal",
		WorkspaceName: "base",
	}
	// assert the Path
	assert.Equal(t, "personal@base=github\u2022com/kalbasit/tmx", p.SessionName())
}
