package code

import (
	"sort"
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
	expected := map[string]*Story{
		"base": &Story{
			Name:        "base",
			CodePath:    "/home/kalbasit/code",
			ProfileName: "TestProfileScan",
			Projects: map[string]*Project{
				"github.com/kalbasit/tmx": &Project{
					ImportPath:  "github.com/kalbasit/tmx",
					CodePath:    "/home/kalbasit/code",
					ProfileName: "TestProfileScan",
					StoryName:   "base",
				},
				"github.com/kalbasit/dotfiles": &Project{
					ImportPath:  "github.com/kalbasit/dotfiles",
					CodePath:    "/home/kalbasit/code",
					ProfileName: "TestProfileScan",
					StoryName:   "base",
				},
			},
		},
		"STORY-123": &Story{
			Name:        "STORY-123",
			CodePath:    "/home/kalbasit/code",
			ProfileName: "TestProfileScan",
			Projects: map[string]*Project{
				"github.com/kalbasit/private": &Project{
					ImportPath:  "github.com/kalbasit/private",
					CodePath:    "/home/kalbasit/code",
					ProfileName: "TestProfileScan",
					StoryName:   "STORY-123",
				},
			},
		},
	}
	assert.Equal(t, expected["base"].Name, p.Stories["base"].Name)
	assert.Equal(t, expected["base"].CodePath, p.Stories["base"].CodePath)
	assert.Equal(t, expected["base"].ProfileName, p.Stories["base"].ProfileName)
	assert.Equal(t, expected["base"].Projects["github.com/kalbasit/tmx"], p.Stories["base"].Projects["github.com/kalbasit/tmx"])
	assert.Equal(t, expected["base"].Projects["github.com/kalbasit/dotfiles"], p.Stories["base"].Projects["github.com/kalbasit/dotfiles"])
	assert.Equal(t, expected["STORY-123"].Name, p.Stories["STORY-123"].Name)
	assert.Equal(t, expected["STORY-123"].CodePath, p.Stories["STORY-123"].CodePath)
	assert.Equal(t, expected["STORY-123"].ProfileName, p.Stories["STORY-123"].ProfileName)
	assert.Equal(t, expected["STORY-123"].Projects["github.com/kalbasit/private"], p.Stories["STORY-123"].Projects["github.com/kalbasit/private"])
}

func TestProfileSessionNames(t *testing.T) {
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
		"TestProfileSessionNames@base=github" + dotChar + "com/kalbasit/tmx",
		"TestProfileSessionNames@base=github" + dotChar + "com/kalbasit/dotfiles",
		"TestProfileSessionNames@STORY-123=github" + dotChar + "com/kalbasit/private",
	}
	got := c.Profiles["TestProfileSessionNames"].SessionNames()
	sort.Strings(want)
	sort.Strings(got)
	assert.Equal(t, want, got)
}

func TestBaseWorkSpace(t *testing.T) {
	// create a new Code
	c := &Code{
		Profiles: map[string]*Profile{
			"personal": &Profile{
				Stories: map[string]*Story{
					"base": &Story{},
				},
			},
		},
	}
	// assert now
	assert.Exactly(t, c.Profiles["personal"].Stories[BaseStory], c.Profiles["personal"].BaseStory())
}
