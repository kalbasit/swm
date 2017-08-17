package code

import (
	"regexp"
	"testing"

	"github.com/kalbasit/swm/testhelper"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestProfileCoder(t *testing.T) {
	// create a new story
	p := &profile{
		name: "TestStoryGoPath",
		code: &code{
			path: "/code",
		},
	}
	assert.Equal(t, Coder(p.code), p.Coder())
}

func TestProfileName(t *testing.T) {
	// create a new story
	p := &profile{
		name: "TestStoryGoPath",
		code: &code{
			path: "/code",
		},
	}
	assert.Equal(t, p.name, p.Name())
}

func TestProfileBase(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFs
	AppFs = afero.NewMemMapFs()
	defer func() { AppFs = oldAppFS }()
	// create the filesystem we want to scan
	testhelper.CreateProjects(t, AppFs)
	// create a new code
	c := New("/code", regexp.MustCompile("^.snapshots$"))
	// scan now
	if err := c.Scan(); err != nil {
		t.Fatalf("code scan failed: %s", err)
	}
	// get a profile
	p, err := c.Profile(t.Name())
	if assert.NoError(t, err) {
		assert.Equal(t, Story(p.(*profile).stories[baseStoryName]), p.Base())
	}
}

/*
func TestProfilePath(t *testing.T) {
	// create a new project
	p := &profile{
		name:     "personal",
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
	p := &profile{
		name:     "TestProfileScan",
		CodePath: "/home/kalbasit/code",
	}
	// scan now
	p.Scan()
	// assert now
	expected := map[string]*story{
		"base": &story{
			name:        "base",
			codePath:    "/home/kalbasit/code",
			profileName: "TestProfileScan",
			projects: map[string]*project{
				"github.com/kalbasit/swm": &project{
					ImportPath:  "github.com/kalbasit/swm",
					CodePath:    "/home/kalbasit/code",
					ProfileName: "TestProfileScan",
					StoryName:   "base",
				},
				"github.com/kalbasit/dotfiles": &project{
					ImportPath:  "github.com/kalbasit/dotfiles",
					CodePath:    "/home/kalbasit/code",
					ProfileName: "TestProfileScan",
					StoryName:   "base",
				},
			},
		},
		"STORY-123": &story{
			name:        "STORY-123",
			codePath:    "/home/kalbasit/code",
			profileName: "TestProfileScan",
			projects: map[string]*project{
				"github.com/kalbasit/private": &project{
					ImportPath:  "github.com/kalbasit/private",
					CodePath:    "/home/kalbasit/code",
					ProfileName: "TestProfileScan",
					StoryName:   "STORY-123",
				},
			},
		},
	}
	assert.Equal(t, expected["base"].name, p.stories["base"].name)
	assert.Equal(t, expected["base"].codePath, p.stories["base"].codePath)
	assert.Equal(t, expected["base"].profileName, p.stories["base"].profileName)
	assert.Equal(t, expected["base"].projects["github.com/kalbasit/swm"], p.stories["base"].projects["github.com/kalbasit/swm"])
	assert.Equal(t, expected["base"].projects["github.com/kalbasit/dotfiles"], p.stories["base"].projects["github.com/kalbasit/dotfiles"])
	assert.Equal(t, expected["STORY-123"].name, p.stories["STORY-123"].name)
	assert.Equal(t, expected["STORY-123"].codePath, p.stories["STORY-123"].codePath)
	assert.Equal(t, expected["STORY-123"].profileName, p.stories["STORY-123"].profileName)
	assert.Equal(t, expected["STORY-123"].projects["github.com/kalbasit/private"], p.stories["STORY-123"].projects["github.com/kalbasit/private"])
}

func TestProfileSessionNames(t *testing.T) {
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
		"TestProfileSessionNames@base=github" + dotChar + "com/kalbasit/swm",
		"TestProfileSessionNames@base=github" + dotChar + "com/kalbasit/dotfiles",
		"TestProfileSessionNames@STORY-123=github" + dotChar + "com/kalbasit/private",
	}
	got := c.profiles["TestProfileSessionNames"].SessionNames()
	sort.Strings(want)
	sort.Strings(got)
	assert.Equal(t, want, got)
}

func TestBaseWorkSpace(t *testing.T) {
	// create a new Code
	c := &code{
		profiles: map[string]*profile{
			"personal": &profile{
				stories: map[string]*story{
					"base": &story{},
				},
			},
		},
	}
	// assert now
	assert.Exactly(t, c.profiles["personal"].stories[BaseStory], c.profiles["personal"].BaseStory())
}
*/
