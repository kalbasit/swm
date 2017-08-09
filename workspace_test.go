package tmx

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestWorkspacePath(t *testing.T) {
	// create a new project
	p := &Workspace{
		Name:        "base",
		CodePath:    "/home/kalbasit/code",
		ProfileName: "personal",
	}
	// assert the Path
	assert.Equal(t, "/home/kalbasit/code/personal/base", p.Path())
}

func TestWorkspaceScan(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFs
	AppFs = afero.NewMemMapFs()
	defer func() { AppFs = oldAppFS }()
	// create the filesystem we want to scan
	prepareFilesystem(t.Name())
	// create a workspace
	w := &Workspace{
		Name:        "base",
		CodePath:    "/home/kalbasit/code",
		ProfileName: "TestWorkspaceScan",
	}
	// scan now
	w.Scan()
	// assert now
	expected := map[string]*Project{
		"github.com/kalbasit/tmx": &Project{
			ImportPath:    "github.com/kalbasit/tmx",
			CodePath:      "/home/kalbasit/code",
			ProfileName:   "TestWorkspaceScan",
			WorkspaceName: "base",
		},
		"github.com/kalbasit/dotfiles": &Project{
			ImportPath:    "github.com/kalbasit/dotfiles",
			CodePath:      "/home/kalbasit/code",
			ProfileName:   "TestWorkspaceScan",
			WorkspaceName: "base",
		},
	}
	assert.Equal(t, expected, w.Projects)
}
