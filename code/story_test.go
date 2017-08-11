package code

/*

func TestWorkspacePath(t *testing.T) {
	// create a new project
	p := &story{
		name:        "base",
		codePath:    "/home/kalbasit/code",
		profileName: "personal",
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
	w := &story{
		name:        "base",
		codePath:    "/home/kalbasit/code",
		profileName: "TestWorkspaceScan",
	}
	// scan now
	w.scan()
	// assert now
	expected := map[string]*project{
		"github.com/kalbasit/swm": &project{
			ImportPath:  "github.com/kalbasit/swm",
			CodePath:    "/home/kalbasit/code",
			ProfileName: "TestWorkspaceScan",
			StoryName:   "base",
		},
		"github.com/kalbasit/dotfiles": &project{
			ImportPath:  "github.com/kalbasit/dotfiles",
			CodePath:    "/home/kalbasit/code",
			ProfileName: "TestWorkspaceScan",
			StoryName:   "base",
		},
	}
	assert.Equal(t, expected, w.projects)

	// test with the non base workspace
	w = &story{
		name:        "STORY-123",
		codePath:    "/home/kalbasit/code",
		profileName: "TestWorkspaceScan",
	}
	// scan now
	w.scan()
	// assert now
	expected = map[string]*project{
		"github.com/kalbasit/dotfiles": &project{
			ImportPath:  "github.com/kalbasit/dotfiles",
			CodePath:    "/home/kalbasit/code",
			ProfileName: "TestWorkspaceScan",
			StoryName:   "STORY-123",
		},
	}
	assert.Equal(t, expected, w.projects)
}

func TestWorkspaceSessionNames(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFs
	AppFs = afero.NewMemMapFs()
	defer func() { AppFs = oldAppFS }()
	// create the filesystem we want to scan
	prepareFilesystem(t.Name())
	// create a code
	c := &code{
		path: "/home/kalbasit/code",
	}
	// scan now
	c.scan()
	// assert now
	want := []string{
		"TestWorkspaceSessionNames@base=github" + dotChar + "com/kalbasit/swm",
		"TestWorkspaceSessionNames@base=github" + dotChar + "com/kalbasit/dotfiles",
	}
	got := c.profiles["TestWorkspaceSessionNames"].stories["base"].SessionNames()
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
	c := &code{
		path: "/home/kalbasit/code",
	}
	// scan now
	c.scan()
	// assert it now
	expected := &project{
		ImportPath:  "github.com/kalbasit/swm",
		CodePath:    "/home/kalbasit/code",
		ProfileName: "TestWorkspaceFindProjectBySessionName",
		StoryName:   "base",
	}
	project, err := c.profiles["TestWorkspaceFindProjectBySessionName"].stories["base"].FindProjectBySessionName("github" + dotChar + "com/kalbasit/swm")
	if assert.NoError(t, err) {
		assert.Equal(t, expected, project)
	}
}
*/
