package tmux

import (
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"testing"

	"github.com/kalbasit/swm/code"
	"github.com/kalbasit/swm/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSessionProjectsNoStory(t *testing.T) {
	// create a temporary directory
	dir, err := ioutil.TempDir("", "swm-test-*")
	require.NoError(t, err)

	// delete it once we are done here
	defer func() { os.RemoveAll(dir) }()

	// create the filesystem we want to scan
	testhelper.CreateProjects(t, dir)

	// create a code
	c := code.New(nil, dir, "", regexp.MustCompile("^.snapshots$"))
	require.NoError(t, c.Scan())

	// create the tmux client
	tmx := &tmux{options: &Options{Code: c}}

	// get the session map
	sessionNameProjects, err := tmx.getSessionNameProjects()
	require.NoError(t, err)
	assert.NotEmpty(t, sessionNameProjects)
	// assert the keys in the map
	var keys []string
	for k, _ := range sessionNameProjects {
		keys = append(keys, k)
	}
	expectedKeys := []string{
		"github" + dotChar + "com/owner1/repo1",
		"github" + dotChar + "com/owner2/repo2",
		"github" + dotChar + "com/owner3/repo3",
	}
	sort.Strings(keys)
	sort.Strings(expectedKeys)
	require.Equal(t, expectedKeys, keys)
	// assert the correct project
	for name, prj := range sessionNameProjects {
		switch name {
		case "github" + dotChar + "com/owner1/repo1":
			assert.Equal(t, "github.com/owner1/repo1", prj.String())
		case "github" + dotChar + "com/owner2/repo2":
			assert.Equal(t, "github.com/owner2/repo2", prj.String())
		case "github" + dotChar + "com/owner3/repo3":
			assert.Equal(t, "github.com/owner3/repo3", prj.String())
		}
	}
}

func TestGetSessionProjectsStory123(t *testing.T) {
	// create a temporary directory
	dir, err := ioutil.TempDir("", "swm-test-*")
	require.NoError(t, err)

	// delete it once we are done here
	defer func() { os.RemoveAll(dir) }()

	// create the filesystem we want to scan
	testhelper.CreateProjects(t, dir)

	// create a code
	c := code.New(nil, dir, "STORY-123", regexp.MustCompile("^.snapshots$"))
	require.NoError(t, c.Scan())

	// create the tmux client
	tmx := &tmux{options: &Options{
		Code:      c,
		StoryName: "STORY-123",
	}}

	// get the session map
	sessionNameProjects, err := tmx.getSessionNameProjects()
	require.NoError(t, err)
	assert.NotEmpty(t, sessionNameProjects)

	// assert the keys in the map
	var keys []string
	for k, _ := range sessionNameProjects {
		keys = append(keys, k)
	}
	expectedKeys := []string{
		"github" + dotChar + "com/owner1/repo1",
		"github" + dotChar + "com/owner2/repo2",
		"github" + dotChar + "com/owner3/repo3",
	}
	sort.Strings(keys)
	sort.Strings(expectedKeys)
	require.Equal(t, expectedKeys, keys)

	// assert the correct project
	for name, prj := range sessionNameProjects {
		switch name {
		case "github" + dotChar + "com/owner1/repo1":
			assert.Equal(t, "github.com/owner1/repo1", prj.String())
		case "github" + dotChar + "com/owner2/repo2":
			assert.Equal(t, "github.com/owner2/repo2", prj.String())
		case "github" + dotChar + "com/owner3/repo3":
			assert.Equal(t, "github.com/owner3/repo3", prj.String())
		}
	}
}

func TestSanitize(t *testing.T) {
	t.Run(".", func(t *testing.T) {
		assert.Equal(t, "github"+dotChar+"com/owner1/repo1", sanitize("github.com/owner1/repo1"))
	})

	t.Run(":", func(t *testing.T) {
		assert.Equal(t, "github"+colonChar+"com/owner1/repo1", sanitize("github:com/owner1/repo1"))
	})
}
