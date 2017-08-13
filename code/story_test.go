package code

import (
	"os"
	"regexp"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestStoryProfile(t *testing.T) {
	// create a new story
	s := &story{
		name: "base",
		profile: &profile{
			name: "TestStoryGoPath",
			code: &code{
				path: "/code",
			},
		},
	}
	assert.Equal(t, s.profile, s.Profile())
}

func TestStoryBase(t *testing.T) {
	// test with a base story

	// create a new story
	s := &story{
		name: "base",
		profile: &profile{
			name: "TestStoryGoPath",
			code: &code{
				path: "/code",
			},
		},
	}
	assert.True(t, s.Base())

	// test with a real story

	// create a new story
	s = &story{
		name: "STORY-123",
		profile: &profile{
			name: "TestStoryGoPath",
			code: &code{
				path: "/code",
			},
		},
	}
	assert.False(t, s.Base())
}

func TestStoryGoPath(t *testing.T) {
	// testing the case of base

	// create a new story
	s := &story{
		name: "base",
		profile: &profile{
			name: "TestStoryGoPath",
			code: &code{
				path: "/code",
			},
		},
	}
	// assert the Path
	assert.Equal(t, "/code/TestStoryGoPath/base", s.GoPath())

	// testing the case of a story

	// create a new story
	s = &story{
		name: "STORY-123",
		profile: &profile{
			name: "TestStoryGoPath",
			code: &code{
				path: "/code",
			},
		},
	}
	// assert the Path
	assert.Equal(t, "/code/TestStoryGoPath/stories/STORY-123", s.GoPath())
}

func TestStoryProjects(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFs
	AppFs = afero.NewMemMapFs()
	defer func() { AppFs = oldAppFS }()
	// create the filesystem we want to scan
	prepareFilesystem(t)
	// create a new code
	c := New("/code", regexp.MustCompile("^.snapshots$"))
	// scan now
	if err := c.Scan(); err != nil {
		t.Fatalf("code scan failed: %s", err)
	}
	// get a profile
	p, err := c.Profile(t.Name())
	if assert.NoError(t, err) {
		// get a story
		s := p.Base()
		// get the projects
		prjs := s.Projects()
		// assert they are the same as the projects
		var expectedPrjs []Project
		{
			for _, prj2 := range s.(*story).projects {
				expectedPrjs = append(expectedPrjs, Project(prj2))
			}
		}
		assert.Equal(t, expectedPrjs, prjs)
	}
}

func TestStoryProject(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFs
	AppFs = afero.NewMemMapFs()
	defer func() { AppFs = oldAppFS }()
	// create the filesystem we want to scan
	prepareFilesystem(t)
	// create a new code
	c := New("/code", regexp.MustCompile("^.snapshots$"))
	// scan now
	if err := c.Scan(); err != nil {
		t.Fatalf("code scan failed: %s", err)
	}
	// get a profile
	p, err := c.Profile(t.Name())
	if assert.NoError(t, err) {

		// testing with a base story

		s := p.Base()
		for importPath, expectedPrj := range s.(*story).projects {
			prj, err := s.Project(importPath)
			if assert.NoError(t, err) {
				assert.Equal(t, Project(expectedPrj), prj)
				_, err := AppFs.Stat(prj.Path())
				assert.NoError(t, err)
			}
		}

		// testing with story that does exist

		s, err = p.Story("STORY-123")
		if assert.NoError(t, err) {
			for importPath, expectedPrj := range s.(*story).projects {
				prj, err := s.Project(importPath)
				if assert.NoError(t, err) {
					assert.Equal(t, Project(expectedPrj), prj)
					_, err := AppFs.Stat(prj.Path())
					assert.NoError(t, err)
				}
			}
		}

		// testing with a story that does not exist (no test for ensure here)

		s, err = p.Story("STORY-123")
		if assert.NoError(t, err) {
			// prepare the expected
			expectedPrj, err := p.Base().Project("github.com/kalbasit/workflow")
			if assert.NoError(t, err) {
				expectedPrj.(*project).story = s.(*story)
				prj, err := s.Project("github.com/kalbasit/workflow")
				if assert.NoError(t, err) {
					assert.Equal(t, Project(expectedPrj), prj)
					_, err := AppFs.Stat(prj.Path())
					assert.True(t, os.IsNotExist(err))
				}
			}
		}
	}
}

/*
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
