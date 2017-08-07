package tmx

import (
	"regexp"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCodeScan(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFs
	AppFs = afero.NewMemMapFs()
	defer func() { AppFs = oldAppFS }()
	// create the filesystem we want to scan
	prepareFilesystem()
	// create a code
	c := &Code{
		Path:           "/home/kalbasit/code",
		ExcludePattern: regexp.MustCompile("^.snapshots$"),
	}
	// scan now
	c.Scan()
	// assert now
	expected := map[string]*Profile{
		"TestCodeScan": &Profile{
			Name:     "TestCodeScan",
			CodePath: "/home/kalbasit/code",
			Workspaces: map[string]*Workspace{
				"base": &Workspace{
					Name:        "base",
					CodePath:    "/home/kalbasit/code",
					ProfileName: "TestCodeScan",
					Projects: map[string]*Project{
						"github.com/kalbasit/tmx": &Project{
							ImportPath:    "github.com/kalbasit/tmx",
							CodePath:      "/home/kalbasit/code",
							ProfileName:   "TestCodeScan",
							WorkspaceName: "base",
						},
						"github.com/kalbasit/dotfiles": &Project{
							ImportPath:    "github.com/kalbasit/dotfiles",
							CodePath:      "/home/kalbasit/code",
							ProfileName:   "TestCodeScan",
							WorkspaceName: "base",
						},
					},
				},
				"STORY-123": &Workspace{
					Name:        "STORY-123",
					CodePath:    "/home/kalbasit/code",
					ProfileName: "TestCodeScan",
					Projects: map[string]*Project{
						"github.com/kalbasit/private": &Project{
							ImportPath:    "github.com/kalbasit/private",
							CodePath:      "/home/kalbasit/code",
							ProfileName:   "TestCodeScan",
							WorkspaceName: "STORY-123",
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected["TestCodeScan"].Workspaces["base"].Name, c.Profiles["TestCodeScan"].Workspaces["base"].Name)
	assert.Equal(t, expected["TestCodeScan"].Workspaces["base"].CodePath, c.Profiles["TestCodeScan"].Workspaces["base"].CodePath)
	assert.Equal(t, expected["TestCodeScan"].Workspaces["base"].ProfileName, c.Profiles["TestCodeScan"].Workspaces["base"].ProfileName)
	assert.Equal(t, expected["TestCodeScan"].Workspaces["base"].Projects["github.com/kalbasit/tmx"], c.Profiles["TestCodeScan"].Workspaces["base"].Projects["github.com/kalbasit/tmx"])
	assert.Equal(t, expected["TestCodeScan"].Workspaces["base"].Projects["github.com/kalbasit/dotfiles"], c.Profiles["TestCodeScan"].Workspaces["base"].Projects["github.com/kalbasit/dotfiles"])
	assert.Equal(t, expected["TestCodeScan"].Workspaces["STORY-123"].Name, c.Profiles["TestCodeScan"].Workspaces["STORY-123"].Name)
	assert.Equal(t, expected["TestCodeScan"].Workspaces["STORY-123"].CodePath, c.Profiles["TestCodeScan"].Workspaces["STORY-123"].CodePath)
	assert.Equal(t, expected["TestCodeScan"].Workspaces["STORY-123"].ProfileName, c.Profiles["TestCodeScan"].Workspaces["STORY-123"].ProfileName)
	assert.Equal(t, expected["TestCodeScan"].Workspaces["STORY-123"].Projects["github.com/kalbasit/private"], c.Profiles["TestCodeScan"].Workspaces["STORY-123"].Projects["github.com/kalbasit/private"])
}

func TestFindProjectBySessionName(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFs
	AppFs = afero.NewMemMapFs()
	defer func() { AppFs = oldAppFS }()
	// create the filesystem we want to scan
	prepareFilesystem()
	// create a code
	c := &Code{
		Path: "/home/kalbasit/code",
	}
	// scan now
	c.Scan()
	// assert now
	expected := &Project{
		ImportPath:    "github.com/kalbasit/tmx",
		CodePath:      "/home/kalbasit/code",
		ProfileName:   "TestFindProjectBySessionName",
		WorkspaceName: "base",
	}
	project, err := c.FindProjectBySessionName("TestFindProjectBySessionName@base=github" + dotChar + "com/kalbasit/tmx")
	if assert.NoError(t, err) {
		assert.Equal(t, expected, project)
	}
}
