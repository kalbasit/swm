package code

import (
	"sort"
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

	// test with the non base workspace
	w = &Workspace{
		Name:        "STORY-123",
		CodePath:    "/home/kalbasit/code",
		ProfileName: "TestWorkspaceScan",
	}
	// scan now
	w.Scan()
	// assert now
	expected = map[string]*Project{
		"github.com/kalbasit/dotfiles": &Project{
			ImportPath:    "github.com/kalbasit/dotfiles",
			CodePath:      "/home/kalbasit/code",
			ProfileName:   "TestWorkspaceScan",
			WorkspaceName: "STORY-123",
		},
	}
	assert.Equal(t, expected, w.Projects)
}

func TestWorkspaceSessionNames(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFs
	AppFs = afero.NewMemMapFs()
	defer func() { AppFs = oldAppFS }()
	// create the filesystem we want to scan
	prepareFilesystem(t.Name())
	// create a code
	c := &Code{
		Path: "/home/kalbasit/code",
	}
	// scan now
	c.Scan()
	// assert now
	want := []string{
		"TestWorkspaceSessionNames@base=github" + dotChar + "com/kalbasit/tmx",
		"TestWorkspaceSessionNames@base=github" + dotChar + "com/kalbasit/dotfiles",
	}
	got := c.Profiles["TestWorkspaceSessionNames"].Workspaces["base"].SessionNames()
	sort.Strings(want)
	sort.Strings(got)
	assert.Equal(t, want, got)
}

func TestWorkspaceFindProjectBySessionName(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFs
	AppFs = afero.NewMemMapFs()
	defer func() { AppFs = oldAppFS }()
	// create the filesystem we want to scan
	prepareFilesystem(t.Name())
	// create a code
	c := &Code{
		Path: "/home/kalbasit/code",
	}
	// scan now
	c.Scan()
	// assert it now
	expected := &Project{
		ImportPath:    "github.com/kalbasit/tmx",
		CodePath:      "/home/kalbasit/code",
		ProfileName:   "TestWorkspaceFindProjectBySessionName",
		WorkspaceName: "base",
	}
	project, err := c.Profiles["TestWorkspaceFindProjectBySessionName"].Workspaces["base"].FindProjectBySessionName("github" + dotChar + "com/kalbasit/tmx")
	if assert.NoError(t, err) {
		assert.Equal(t, expected, project)
	}
}
