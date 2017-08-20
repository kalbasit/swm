package code

import (
	"io/ioutil"
	"regexp"
	"testing"

	"github.com/kalbasit/swm/testhelper"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// discard logs
	log.Logger = zerolog.New(ioutil.Discard)
}

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

		// assert the base story
		assert.Equal(t, expected[t.Name()].getStories()["base"].name, c.getProfiles()[t.Name()].getStories()["base"].name)
		assert.Equal(t, expected[t.Name()].getStories()["base"].profile.name, c.getProfiles()[t.Name()].getStories()["base"].profile.name)
		assert.Equal(t, expected[t.Name()].getStories()["base"].getProjects()["github.com/kalbasit/swm"].importPath, c.getProfiles()[t.Name()].getStories()["base"].getProjects()["github.com/kalbasit/swm"].importPath)
		assert.Equal(t, expected[t.Name()].getStories()["base"].getProjects()["github.com/kalbasit/dotfiles"].importPath, c.getProfiles()[t.Name()].getStories()["base"].getProjects()["github.com/kalbasit/dotfiles"].importPath)
		assert.Equal(t, expected[t.Name()].getStories()["base"].getProjects()["github.com/kalbasit/workflow"].importPath, c.getProfiles()[t.Name()].getStories()["base"].getProjects()["github.com/kalbasit/workflow"].importPath)

		// assert the STORY-123 story
		assert.Equal(t, expected[t.Name()].getStories()["STORY-123"].name, c.getProfiles()[t.Name()].getStories()["STORY-123"].name)
		assert.Equal(t, expected[t.Name()].getStories()["STORY-123"].profile.name, c.getProfiles()[t.Name()].getStories()["STORY-123"].profile.name)
		assert.Equal(t, expected[t.Name()].getStories()["STORY-123"].getProjects()["github.com/kalbasit/dotfiles"].importPath, c.getProfiles()[t.Name()].getStories()["STORY-123"].getProjects()["github.com/kalbasit/dotfiles"].importPath)
		assert.Equal(t, expected[t.Name()].getStories()["STORY-123"].getProjects()["github.com/kalbasit/swm"].importPath, c.getProfiles()[t.Name()].getStories()["STORY-123"].getProjects()["github.com/kalbasit/swm"].importPath)
	}
	// scan now
	require.NoError(t, c.Scan())
	assertFn()
}

func TestCodeProfile(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFS
	AppFS = afero.NewMemMapFs()
	defer func() { AppFS = oldAppFS }()
	// create the filesystem we want to scan
	testhelper.CreateProjects(t, AppFS)
	// create a code
	c := New("/code", regexp.MustCompile("^.snapshots$"))
	// define the assertion function
	assertFn := func(pTest *profile) {
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

		// assert the base story
		assert.Equal(t, expected[t.Name()].getStories()["base"].name, pTest.getStories()["base"].name)
		assert.Equal(t, expected[t.Name()].getStories()["base"].profile.name, pTest.getStories()["base"].profile.name)
		assert.Equal(t, expected[t.Name()].getStories()["base"].getProjects()["github.com/kalbasit/swm"].importPath, pTest.getStories()["base"].getProjects()["github.com/kalbasit/swm"].importPath)
		assert.Equal(t, expected[t.Name()].getStories()["base"].getProjects()["github.com/kalbasit/dotfiles"].importPath, pTest.getStories()["base"].getProjects()["github.com/kalbasit/dotfiles"].importPath)
		assert.Equal(t, expected[t.Name()].getStories()["base"].getProjects()["github.com/kalbasit/workflow"].importPath, pTest.getStories()["base"].getProjects()["github.com/kalbasit/workflow"].importPath)

		// assert the STORY-123 story
		assert.Equal(t, expected[t.Name()].getStories()["STORY-123"].name, pTest.getStories()["STORY-123"].name)
		assert.Equal(t, expected[t.Name()].getStories()["STORY-123"].profile.name, pTest.getStories()["STORY-123"].profile.name)
		assert.Equal(t, expected[t.Name()].getStories()["STORY-123"].getProjects()["github.com/kalbasit/dotfiles"].importPath, pTest.getStories()["STORY-123"].getProjects()["github.com/kalbasit/dotfiles"].importPath)
		assert.Equal(t, expected[t.Name()].getStories()["STORY-123"].getProjects()["github.com/kalbasit/swm"].importPath, pTest.getStories()["STORY-123"].getProjects()["github.com/kalbasit/swm"].importPath)
	}
	// assert it throws an error before scanning
	_, err := c.Profile(t.Name())
	assert.EqualError(t, err, ErrCoderNotScanned.Error())
	// scan now
	require.NoError(t, c.Scan())
	// get the profile
	p, err := c.Profile(t.Name())
	require.NoError(t, c.Scan())
	assertFn(p.(*profile))
}

func TestPath(t *testing.T) {
	c := &code{path: "/code"}
	assert.Equal(t, "/code", c.Path())
}
