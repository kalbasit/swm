package code

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/kalbasit/swm/ifaces"
	"github.com/kalbasit/swm/story"
	"github.com/kalbasit/swm/testhelper"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// discard logs
	log.Logger = zerolog.New(ioutil.Discard)
}

func TestScan(t *testing.T) {
	// create a temporary directory
	dir, err := ioutil.TempDir("", "swm-test-*")
	require.NoError(t, err)

	// delete it once we are done here
	defer func() { os.RemoveAll(dir) }()

	// create the filesystem we want to scan
	require.NoError(t, testhelper.CreateProjects(dir))

	// define the assertion function
	assertFn := func(c ifaces.Code, story_name string) {
		// assert the repositories
		for _, importPath := range []string{"github.com/owner1/repo1", "github.com/owner2/repo2", "github.com/owner3/repo3"} {
			prj, err := c.GetProjectByRelativePath(importPath)
			require.NoError(t, err)

			assert.Equal(t, importPath, prj.String())
			assert.Equal(t, path.Join(dir, "repositories", importPath), prj.Path(nil))

			s, err := story.New(story_name, "")
			require.NoError(t, err)

			require.NoError(t, prj.CreateStory(s))

			sn := prj.Path(s)
			assert.Equal(t, path.Join(dir, "stories", story_name, importPath), sn)
		}
	}

	sc := New(dir, regexp.MustCompile("^.snapshots$"))
	require.NoError(t, sc.Scan())
	assertFn(sc, t.Name())
}

func TestPath(t *testing.T) {
	c := &code{path: "/code"}
	assert.Equal(t, "/code", c.Path())
}

func TestGetProject(t *testing.T) {
	// create a temporary directory
	dir, err := ioutil.TempDir("", "swm-test-*")
	require.NoError(t, err)

	// delete it once we are done here
	defer func() { os.RemoveAll(dir) }()

	// create the filesystem we want to scan
	require.NoError(t, testhelper.CreateProjects(dir))

	testCases := []struct {
		story_name string
	}{
		{
			story_name: "",
		},
		{
			story_name: t.Name(),
		},
	}

	for _, testCase := range testCases {
		// create a code
		c := New(dir, regexp.MustCompile("^.snapshots$"))
		require.NoError(t, c.Scan())

		// create the story
		var s ifaces.Story
		if testCase.story_name != "" {
			s, err = story.New(testCase.story_name, "")
			require.NoError(t, err)
		}

		// get the project and assert things
		for _, importPath := range []string{"github.com/owner1/repo1", "github.com/owner2/repo2", "github.com/owner3/repo3"} {

			prj, err := c.GetProjectByRelativePath(importPath)
			require.NoError(t, err)
			assert.Equal(t, path.Join(dir, "repositories", importPath), prj.Path(nil))

			if testCase.story_name != "" {
				sp := prj.Path(s)
				assert.Equal(t, path.Join(dir, "stories", testCase.story_name, importPath), sp)
			}
		}
	}
}

func TestProjects(t *testing.T) {
	// create a temporary directory
	dir, err := ioutil.TempDir("", "swm-test-*")
	require.NoError(t, err)

	// delete it once we are done here
	defer func() { os.RemoveAll(dir) }()

	// create the filesystem we want to scan
	require.NoError(t, testhelper.CreateProjects(dir))

	// create a code
	c := New(dir, regexp.MustCompile("^.snapshots$"))
	require.NoError(t, c.Scan())

	// get all the projects, and collect their import paths, then compare those to the expected ones
	expectedImportPaths := []string{"github.com/owner1/repo1", "github.com/owner2/repo2", "github.com/owner3/repo3"}
	sort.Strings(expectedImportPaths)

	prjs := c.Projects()
	if assert.Len(t, prjs, len(expectedImportPaths)) {
		var gotImportPaths []string
		for _, prj := range prjs {
			gotImportPaths = append(gotImportPaths, prj.String())
		}

		sort.Strings(gotImportPaths)

		assert.EqualValues(t, expectedImportPaths, gotImportPaths)
	}
}

func TestClone(t *testing.T) {
	t.Run("non-exist repository should not create top-level directory of the repository", func(t *testing.T) {
		// create a temporary directory
		dir, err := ioutil.TempDir("", "swm-test-*")
		require.NoError(t, err)

		// delete it once we are done here
		defer func() { os.RemoveAll(dir) }()

		// create a code
		c := New(dir, regexp.MustCompile("^.snapshots$"))
		require.NoError(t, c.Scan())

		// assert that we get an error
		err = c.Clone("file://path/to/inexistent-repository")
		assert.EqualError(t, err, "exit status 128")

		// make sure that the parent of the repository does not exist
		assert.NoDirExists(t, path.Join(c.RepositoriesDir(), "path"))
		assert.NoDirExists(t, path.Join(c.RepositoriesDir(), "path", "to"))

		assert.DirExists(t, path.Join(dir, ".tmp-clone"))
	})

	t.Run("cloning a working URL should work", func(t *testing.T) {
		// create a temporary directory
		dir, err := ioutil.TempDir("", "swm-test-*")
		require.NoError(t, err)

		// delete it once we are done here
		defer func() { os.RemoveAll(dir) }()

		// create the filesystem we want to scan
		require.NoError(t, testhelper.CreateProjects(dir))

		// create a code
		c := New(dir, regexp.MustCompile("^.snapshots$"))
		require.NoError(t, c.Scan())

		// clone the repo4 from the ignored location, but first validate it does not exist in the scanned projects
		importPath := strings.TrimPrefix(path.Join(dir, ".snapshots", "github.com/owner4/repo4"), string(os.PathSeparator))
		_, err = c.GetProjectByRelativePath(importPath)
		require.True(t, errors.Is(err, ErrProjectNotFound))

		err = c.Clone(fmt.Sprintf("file://%s", path.Join(dir, ".snapshots", "github.com/owner4/repo4")))
		if assert.NoError(t, err) {
			prj, err := c.GetProjectByRelativePath(importPath)
			if assert.NoError(t, err) {
				assert.Equal(t, importPath, prj.String())
				assert.Equal(t, path.Join(c.RepositoriesDir(), importPath), prj.Path(nil))
			}
		}
	})
}

func TestGetProjectByAbsolutePath(t *testing.T) {
	// create a temporary directory
	dir, err := ioutil.TempDir("", "swm-test-*")
	require.NoError(t, err)

	// delete it once we are done here
	defer func() { os.RemoveAll(dir) }()

	// create the filesystem we want to scan
	require.NoError(t, testhelper.CreateProjects(dir))

	// create a code
	c := New(dir, regexp.MustCompile("^.snapshots$"))
	require.NoError(t, c.Scan())

	tests := map[string]string{
		dir + "/repositories/github.com/owner1/repo1": "github.com/owner1/repo1",

		dir + "/repositories/github.com/owner2/repo2": "github.com/owner2/repo2",
	}

	for p, ip := range tests {
		prj, err := c.GetProjectByAbsolutePath(p)
		if assert.NoError(t, err) {
			assert.Equal(t, ip, prj.String())
		}
	}

	_, err = c.GetProjectByAbsolutePath("/code/not-existing/base")
	assert.Error(t, err)
	_, err = c.GetProjectByAbsolutePath(dir + "/repositories/github.com/user/repo")
	assert.Error(t, err)
}
