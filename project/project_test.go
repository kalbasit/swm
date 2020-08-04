package project

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/kalbasit/swm/ifaces"
	"github.com/kalbasit/swm/story"
	"github.com/kalbasit/swm/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type code struct {
	path string
}

func (c *code) Clone(url string) error                                  { return nil }
func (c *code) GetProjectByAbsolutePath(string) (ifaces.Project, error) { return nil, nil }
func (c *code) GetProjectByRelativePath(string) (ifaces.Project, error) { return nil, nil }
func (c *code) HookPath() string                                        { return "" }
func (c *code) Path() string                                            { return c.path }
func (c *code) Projects() []ifaces.Project                              { return nil }
func (c *code) RepositoriesDir() string                                 { return path.Join(c.path, "repositories") }
func (c *code) Scan() error                                             { return nil }
func (c *code) StoriesDir() string                                      { return path.Join(c.path, "stories") }

func TestPath(t *testing.T) {
	c := &code{path: "/code"}
	t.Run("no active story", func(t *testing.T) {
		prj := New(c, "hostname/path/to/repo")
		assert.Equal(t, "/code/repositories/hostname/path/to/repo", prj.Path(nil))
	})

	t.Run("active story", func(t *testing.T) {
		c := &code{path: "/code"}
		s, err := story.New("STORY-123", "")
		require.NoError(t, err)
		prj := New(c, "hostname/path/to/repo")
		assert.Equal(t, "/code/stories/STORY-123/hostname/path/to/repo", prj.Path(s))
	})
}

func TestCreateStory(t *testing.T) {
	// create a temporary directory
	dir, err := ioutil.TempDir("", "swm-test-*")
	require.NoError(t, err)

	// delete it once we are done here
	defer func() { os.RemoveAll(dir) }()

	// create the filesystem we want to scan
	require.NoError(t, testhelper.CreateProjects(dir))

	// create a code
	c := &code{path: dir}

	// create the story
	s, err := story.New(t.Name(), "")
	require.NoError(t, err)

	prj := New(c, "github.com/owner1/repo1")
	if assert.NoError(t, prj.CreateStory(s)) {
		sp := prj.Path(s)
		assert.DirExists(t, sp)
		assert.FileExists(t, path.Join(sp, ".git"))
	}
}

func TestString(t *testing.T) {
	assert.Equal(t, "github.com/kalbasit/swm", (&project{importPath: "github.com/kalbasit/swm"}).String())
}
