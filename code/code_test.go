package code

import (
	"regexp"
	"testing"

	"github.com/kalbasit/swm/testhelper"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodeScan(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFS
	AppFS = afero.NewMemMapFs()
	defer func() { AppFS = oldAppFS }()
	// create the filesystem we want to scan
	testhelper.CreateProjects(t, AppFS)
	// create a code
	c := New("/code", regexp.MustCompile("^.snapshots$"))
	// define the assertion function
	assertFn := func() {
		// create the expected structs
		p := newProfile(c, t.Name())
		p.setStories(map[string]*story{
			"base":      newStory(p, "base"),
			"STORY-123": newStory(p, "STORY-123"),
		})
		p.getStories()["base"].setProjects(map[string]*project{
			"github.com/kalbasit/swm":      newProject(p.getStories()["base"], "github.com/kalbasit/swm"),
			"github.com/kalbasit/dotfiles": newProject(p.getStories()["base"], "github.com/kalbasit/dotfiles"),
			"github.com/kalbasit/workflow": newProject(p.getStories()["base"], "github.com/kalbasit/workflow"),
		})
		p.getStories()["STORY-123"].setProjects(map[string]*project{
			"github.com/kalbasit/swm":      newProject(p.getStories()["STORY-123"], "github.com/kalbasit/swm"),
			"github.com/kalbasit/dotfiles": newProject(p.getStories()["STORY-123"], "github.com/kalbasit/dotfiles"),
		})
		expected := map[string]*profile{t.Name(): p}

		assert.Equal(t, expected[t.Name()].getStories()["base"].name, c.getProfiles()[t.Name()].getStories()["base"].name)
		assert.Equal(t, expected[t.Name()].getStories()["base"].profile, c.getProfiles()[t.Name()].getStories()["base"].profile)
		assert.Equal(t, expected[t.Name()].getStories()["base"].getProjects()["github.com/kalbasit/swm"], c.getProfiles()[t.Name()].getStories()["base"].getProjects()["github.com/kalbasit/swm"])
		assert.Equal(t, expected[t.Name()].getStories()["base"].getProjects()["github.com/kalbasit/dotfiles"], c.getProfiles()[t.Name()].getStories()["base"].getProjects()["github.com/kalbasit/dotfiles"])
		assert.Equal(t, expected[t.Name()].getStories()["base"].getProjects()["github.com/kalbasit/workflow"], c.getProfiles()[t.Name()].getStories()["base"].getProjects()["github.com/kalbasit/workflow"])
		assert.Equal(t, expected[t.Name()].getStories()["STORY-123"].name, c.getProfiles()[t.Name()].getStories()["STORY-123"].name)
		assert.Equal(t, expected[t.Name()].getStories()["STORY-123"].profile, c.getProfiles()[t.Name()].getStories()["STORY-123"].profile)
		assert.Equal(t, expected[t.Name()].getStories()["STORY-123"].getProjects()["github.com/kalbasit/dotfiles"], c.getProfiles()[t.Name()].getStories()["STORY-123"].getProjects()["github.com/kalbasit/dotfiles"])
		assert.Equal(t, expected[t.Name()].getStories()["STORY-123"].getProjects()["github.com/kalbasit/swm"], c.getProfiles()[t.Name()].getStories()["STORY-123"].getProjects()["github.com/kalbasit/swm"])
	}
	// scan now
	require.NoError(t, c.Scan())
	assertFn()
}

func TestPath(t *testing.T) {
	c := &code{path: "/code"}
	assert.Equal(t, "/code", c.Path())
}
