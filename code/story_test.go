package code

import (
	"sort"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestWorkspacePath(t *testing.T) {
	// create a new project
	p := &Story{
		Name:        "base",
		CodePath:    "/home/kalbasit/code",
		ProfileName: "personal",
	}
	// assert the Path
	assert.Equal(t, "/home/kalbasit/code/personal/base", p.GoPath())
}

func TestWorkspaceScan(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFs
	AppFs = afero.NewMemMapFs()
	defer func() { AppFs = oldAppFS }()
	// create the filesystem we want to scan
	prepareFilesystem(t.Name())
	// create a workspace
	w := &Story{
		Name:        "base",
		CodePath:    "/home/kalbasit/code",
		ProfileName: "TestWorkspaceScan",
	}
	// scan now
	w.Scan()
	// assert now
	expected := map[string]*Project{
		"github.com/kalbasit/swm": &Project{
			ImportPath:  "github.com/kalbasit/swm",
			CodePath:    "/home/kalbasit/code",
			ProfileName: "TestWorkspaceScan",
			StoryName:   "base",
		},
		"github.com/kalbasit/dotfiles": &Project{
			ImportPath:  "github.com/kalbasit/dotfiles",
			CodePath:    "/home/kalbasit/code",
			ProfileName: "TestWorkspaceScan",
			StoryName:   "base",
		},
	}
	assert.Equal(t, expected, w.Projects)

	// test with the non base workspace
	w = &Story{
		Name:        "STORY-123",
		CodePath:    "/home/kalbasit/code",
		ProfileName: "TestWorkspaceScan",
	}
	// scan now
	w.Scan()
	// assert now
	expected = map[string]*Project{
		"github.com/kalbasit/dotfiles": &Project{
			ImportPath:  "github.com/kalbasit/dotfiles",
			CodePath:    "/home/kalbasit/code",
			ProfileName: "TestWorkspaceScan",
			StoryName:   "STORY-123",
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
		"TestWorkspaceSessionNames@base=github" + dotChar + "com/kalbasit/swm",
		"TestWorkspaceSessionNames@base=github" + dotChar + "com/kalbasit/dotfiles",
	}
	got := c.Profiles["TestWorkspaceSessionNames"].Stories["base"].SessionNames()
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
		ImportPath:  "github.com/kalbasit/swm",
		CodePath:    "/home/kalbasit/code",
		ProfileName: "TestWorkspaceFindProjectBySessionName",
		StoryName:   "base",
	}
	project, err := c.Profiles["TestWorkspaceFindProjectBySessionName"].Stories["base"].FindProjectBySessionName("github" + dotChar + "com/kalbasit/swm")
	if assert.NoError(t, err) {
		assert.Equal(t, expected, project)
	}
}
