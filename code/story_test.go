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
		for _, prj2 := range s.(*story).projects {
			assert.Contains(t, prjs, Project(prj2))
		}
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
