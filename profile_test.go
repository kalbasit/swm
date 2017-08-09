package tmx

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestProfilePath(t *testing.T) {
	// create a new project
	p := &Profile{
		Name:     "personal",
		CodePath: "/home/kalbasit/code",
	}
	// assert the Path
	assert.Equal(t, "/home/kalbasit/code/personal", p.Path())
}

func TestProfileScan(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFs
	AppFs = afero.NewMemMapFs()
	defer func() { AppFs = oldAppFS }()
	// create the filesystem we want to scan
	prepareFilesystem(t.Name())
	// create a workspace
	p := &Profile{
		Name:     "TestProfileScan",
		CodePath: "/home/kalbasit/code",
	}
	// scan now
	p.Scan()
	// assert now
	expected := map[string]*Workspace{
		"base": &Workspace{
			Name:        "base",
			CodePath:    "/home/kalbasit/code",
			ProfileName: "TestProfileScan",
			Projects: map[string]*Project{
				"github.com/kalbasit/tmx": &Project{
					ImportPath:    "github.com/kalbasit/tmx",
					CodePath:      "/home/kalbasit/code",
					ProfileName:   "TestProfileScan",
					WorkspaceName: "base",
				},
				"github.com/kalbasit/dotfiles": &Project{
					ImportPath:    "github.com/kalbasit/dotfiles",
					CodePath:      "/home/kalbasit/code",
					ProfileName:   "TestProfileScan",
					WorkspaceName: "base",
				},
			},
		},
		"STORY-123": &Workspace{
			Name:        "STORY-123",
			CodePath:    "/home/kalbasit/code",
			ProfileName: "TestProfileScan",
			Projects: map[string]*Project{
				"github.com/kalbasit/private": &Project{
					ImportPath:    "github.com/kalbasit/private",
					CodePath:      "/home/kalbasit/code",
					ProfileName:   "TestProfileScan",
					WorkspaceName: "STORY-123",
				},
			},
		},
	}
	assert.Equal(t, expected["base"].Name, p.Workspaces["base"].Name)
	assert.Equal(t, expected["base"].CodePath, p.Workspaces["base"].CodePath)
	assert.Equal(t, expected["base"].ProfileName, p.Workspaces["base"].ProfileName)
	assert.Equal(t, expected["base"].Projects["github.com/kalbasit/tmx"], p.Workspaces["base"].Projects["github.com/kalbasit/tmx"])
	assert.Equal(t, expected["base"].Projects["github.com/kalbasit/dotfiles"], p.Workspaces["base"].Projects["github.com/kalbasit/dotfiles"])
	assert.Equal(t, expected["STORY-123"].Name, p.Workspaces["STORY-123"].Name)
	assert.Equal(t, expected["STORY-123"].CodePath, p.Workspaces["STORY-123"].CodePath)
	assert.Equal(t, expected["STORY-123"].ProfileName, p.Workspaces["STORY-123"].ProfileName)
	assert.Equal(t, expected["STORY-123"].Projects["github.com/kalbasit/private"], p.Workspaces["STORY-123"].Projects["github.com/kalbasit/private"])
}
