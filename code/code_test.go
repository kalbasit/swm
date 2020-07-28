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
	testhelper.CreateProjects(t, dir)

	// define the assertion function
	assertFn := func(c ifaces.Code, story_name string) {
		// assert the repositories
		for _, importPath := range []string{"github.com/owner1/repo1", "github.com/owner2/repo2", "github.com/owner3/repo3"} {
			prj, err := c.GetProject(importPath)
			require.NoError(t, err)

			assert.Equal(t, importPath, prj.String())
			assert.Equal(t, path.Join(dir, "repositories", importPath), prj.RepositoryPath())
			if story_name != "" {
				assert.NoError(t, prj.Ensure())

				sn, err := prj.StoryPath()
				if assert.NoError(t, err) {
					assert.Equal(t, path.Join(dir, "stories", story_name, importPath), sn)
				}
			}
		}
	}

	// create a code without a story
	c := New(nil, dir, "", regexp.MustCompile("^.snapshots$"))
	require.NoError(t, c.Scan())
	assertFn(c, "")

	// create a new code with a story
	sc := New(nil, dir, t.Name(), regexp.MustCompile("^.snapshots$"))
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
	testhelper.CreateProjects(t, dir)

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
		c := New(nil, dir, testCase.story_name, regexp.MustCompile("^.snapshots$"))
		require.NoError(t, c.Scan())

		// get the project and assert things
		for _, importPath := range []string{"github.com/owner1/repo1", "github.com/owner2/repo2", "github.com/owner3/repo3"} {
			prj, err := c.GetProject(importPath)
			require.NoError(t, err)
			assert.Equal(t, path.Join(dir, "repositories", importPath), prj.RepositoryPath())

			if testCase.story_name != "" {
				sp, err := prj.StoryPath()
				if assert.NoError(t, err) {
					assert.Equal(t, path.Join(dir, "stories", testCase.story_name, importPath), sp)
				}
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
	testhelper.CreateProjects(t, dir)

	// create a code
	c := New(nil, dir, "", regexp.MustCompile("^.snapshots$"))
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
	// create a temporary directory
	dir, err := ioutil.TempDir("", "swm-test-*")
	require.NoError(t, err)

	// delete it once we are done here
	defer func() { os.RemoveAll(dir) }()

	// create the filesystem we want to scan
	testhelper.CreateProjects(t, dir)

	// create a code
	c := New(nil, dir, "", regexp.MustCompile("^.snapshots$"))
	require.NoError(t, c.Scan())

	// clone the repo4 from the ignored location, but first validate it does not exist in the scanned projects
	importPath := strings.TrimPrefix(path.Join(dir, ".snapshots", "github.com/owner4/repo4"), string(os.PathSeparator))
	_, err = c.GetProject(importPath)
	require.True(t, errors.Is(err, ErrProjectNotFound))

	err = c.Clone(fmt.Sprintf("file://%s", path.Join(dir, ".snapshots", "github.com/owner4/repo4")))
	if assert.NoError(t, err) {
		prj, err := c.GetProject(importPath)
		if assert.NoError(t, err) {
			assert.Equal(t, importPath, prj.String())
			assert.Equal(t, path.Join(c.RepositoriesDir(), importPath), prj.RepositoryPath())
		}
	}
}

// func TestProjectByAbsolutePath(t *testing.T) {
// 	// compute the test name
// 	testName := t.Name()
// 	// swap the filesystem
// 	oldAppFS := AppFS
// 	AppFS = afero.NewMemMapFs()
// 	defer func() { AppFS = oldAppFS }()
// 	// create the filesystem we want to scan
// 	testhelper.CreateProjects(t, AppFS)
// 	// create a code
// 	c := New("/code", regexp.MustCompile("^.snapshots$"))
// 	// scan now
// 	require.NoError(t, c.Scan())
//
// 	type desc struct {
// 		profile    string
// 		story      string
// 		importPath string
// 	}
// 	tests := map[string]desc{
// 		"/code/" + testName + "/base/src/github.com/kalbasit/swm": desc{
// 			profile:    testName,
// 			story:      baseStoryName,
// 			importPath: "github.com/kalbasit/swm",
// 		},
//
// 		"/code/" + testName + "/base/src/github.com/kalbasit/swm/cmd": desc{
// 			profile:    testName,
// 			story:      baseStoryName,
// 			importPath: "github.com/kalbasit/swm",
// 		},
//
// 		"/code/" + testName + "/stories/STORY-123/src/github.com/kalbasit/swm": desc{
// 			profile:    testName,
// 			story:      "STORY-123",
// 			importPath: "github.com/kalbasit/swm",
// 		},
//
// 		"/code/" + testName + "/stories/STORY-123/src/github.com/kalbasit/swm/cmd": desc{
// 			profile:    testName,
// 			story:      "STORY-123",
// 			importPath: "github.com/kalbasit/swm",
// 		},
// 	}
//
// 	for p, expdec := range tests {
// 		prj, err := c.ProjectByAbsolutePath(p)
// 		if assert.NoError(t, err) {
// 			assert.Equal(t, expdec.profile, prj.Story().Profile().Name())
// 			assert.Equal(t, expdec.story, prj.Story().Name())
// 			assert.Equal(t, expdec.importPath, prj.ImportPath())
// 		}
// 	}
//
// 	var err error
// 	_, err = c.ProjectByAbsolutePath("/code/not-existing/base")
// 	assert.EqualError(t, err, ErrPathIsInvalid.Error())
// 	_, err = c.ProjectByAbsolutePath("/code/not-existing/base/src/github.com/kalbasit/swm")
// 	assert.EqualError(t, err, ErrProfileNoFound.Error())
// 	_, err = c.ProjectByAbsolutePath("/code/" + testName + "/stories/NOT_HERE/src/github.com/kalbasit/swm")
// 	assert.EqualError(t, err, ErrStoryNotFound.Error())
// 	_, err = c.ProjectByAbsolutePath("/code/" + testName + "/base/src/github.com/user/repo")
// 	assert.EqualError(t, err, ErrProjectNotFound.Error())
// }
