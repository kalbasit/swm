package project

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/google/go-github/github"
	"github.com/kalbasit/swm/ifaces"
	"github.com/kalbasit/swm/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type code struct {
	path              string
	story_name        string
	story_branch_name string
}

func (c *code) Clone(url string) error                                               { return nil }
func (c *code) CreateStory() error                                                   { return nil }
func (c *code) GetProjectByAbsolutePath(absolutePath string) (ifaces.Project, error) { return nil, nil }
func (c *code) GetProjectByRelativePath(string) (ifaces.Project, error)              { return nil, nil }
func (c *code) GithubClient() *github.Client                                         { return nil }
func (c *code) SetGithubClient(*github.Client)                                       {}
func (c *code) HookPath() string                                                     { return "" }
func (c *code) Path() string                                                         { return c.path }
func (c *code) Projects() []ifaces.Project                                           { return nil }
func (c *code) RepositoriesDir() string                                              { return path.Join(c.path, "repositories") }
func (c *code) Scan() error                                                          { return nil }
func (c *code) StoriesDir() string                                                   { return path.Join(c.path, "stories") }
func (c *code) SetStoryName(string)                                                  {}
func (c *code) SetStoryBranchName(string)                                            {}
func (c *code) StoryName() string                                                    { return c.story_name }
func (c *code) StoryBranchName() string {
	if c.story_branch_name == "" {
		return c.story_name
	}
	return c.story_branch_name
}

func TestStoryPath(t *testing.T) {
	t.Run("no active story", func(t *testing.T) {
		c := &code{path: "/code"}
		prj := New(c, "hostname/path/to/repo")

		_, err := prj.StoryPath()
		assert.EqualError(t, err, ErrNoActiveStory.Error())
	})

	t.Run("active story", func(t *testing.T) {
		c := &code{path: "/code", story_name: "STORY-123"}
		prj := New(c, "hostname/path/to/repo")

		sp, err := prj.StoryPath()
		if assert.NoError(t, err) {
			assert.Equal(t, "/code/stories/STORY-123/hostname/path/to/repo", sp)
		}
	})
}

func TestRepositoryPath(t *testing.T) {
	c := &code{path: "/code"}
	prj := New(c, "hostname/path/to/repo")

	assert.Equal(t, "/code/repositories/hostname/path/to/repo", prj.RepositoryPath())
}

func TestEnsure(t *testing.T) {
	// create a temporary directory
	dir, err := ioutil.TempDir("", "swm-test-*")
	require.NoError(t, err)

	// delete it once we are done here
	defer func() { os.RemoveAll(dir) }()

	// create the filesystem we want to scan
	testhelper.CreateProjects(t, dir)

	// create a code
	c := &code{path: dir, story_name: "STORY-123"}

	prj := New(c, "github.com/owner1/repo1")
	if assert.NoError(t, prj.Ensure()) {
		sp, err := prj.StoryPath()
		if assert.NoError(t, err) {
			assert.DirExists(t, sp)
			assert.FileExists(t, path.Join(sp, ".git"))
		}
	}
}

func TestString(t *testing.T) {
	assert.Equal(t, "github.com/kalbasit/swm", (&project{importPath: "github.com/kalbasit/swm"}).String())
}

func TestPath(t *testing.T) {
	t.Run("no active story", func(t *testing.T) {
		c := &code{path: "/code"}
		prj := New(c, "hostname/path/to/repo")

		assert.Equal(t, "/code/repositories/hostname/path/to/repo", prj.Path())
	})

	t.Run("active story", func(t *testing.T) {
		c := &code{path: "/code", story_name: "STORY-123"}
		prj := New(c, "hostname/path/to/repo")

		assert.Equal(t, "/code/stories/STORY-123/hostname/path/to/repo", prj.Path())
	})
}
