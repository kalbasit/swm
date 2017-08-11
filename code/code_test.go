package code

import (
	"encoding/json"
	"path"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFs
	AppFs = afero.NewMemMapFs()
	defer func() { AppFs = oldAppFS }()
	// create the filesystem we want to scan
	prepareFilesystem(t.Name())
	// create a code
	c := New("/home/kalbasit/code", regexp.MustCompile("^.snapshots$"))
	// define the assertion function
	assertFn := func() {
		// assert that it was loaded correctly
		expected := map[string]*Profile{
			"TestLoad": &Profile{
				Name:     "TestLoad",
				CodePath: "/home/kalbasit/code",
				Stories: map[string]*Story{
					"base": &Story{
						Name:        "base",
						CodePath:    "/home/kalbasit/code",
						ProfileName: "TestLoad",
						Projects: map[string]*Project{
							"github.com/kalbasit/tmx": &Project{
								ImportPath:  "github.com/kalbasit/tmx",
								CodePath:    "/home/kalbasit/code",
								ProfileName: "TestLoad",
								StoryName:   "base",
							},
							"github.com/kalbasit/dotfiles": &Project{
								ImportPath:  "github.com/kalbasit/dotfiles",
								CodePath:    "/home/kalbasit/code",
								ProfileName: "TestLoad",
								StoryName:   "base",
							},
						},
					},
					"STORY-123": &Story{
						Name:        "STORY-123",
						CodePath:    "/home/kalbasit/code",
						ProfileName: "TestLoad",
						Projects: map[string]*Project{
							"github.com/kalbasit/private": &Project{
								ImportPath:  "github.com/kalbasit/private",
								CodePath:    "/home/kalbasit/code",
								ProfileName: "TestLoad",
								StoryName:   "STORY-123",
							},
						},
					},
				},
			},
		}
		assert.Equal(t, expected["TestLoad"].Stories["base"].Name, c.Profiles["TestLoad"].Stories["base"].Name)
		assert.Equal(t, expected["TestLoad"].Stories["base"].CodePath, c.Profiles["TestLoad"].Stories["base"].CodePath)
		assert.Equal(t, expected["TestLoad"].Stories["base"].ProfileName, c.Profiles["TestLoad"].Stories["base"].ProfileName)
		assert.Equal(t, expected["TestLoad"].Stories["base"].Projects["github.com/kalbasit/tmx"], c.Profiles["TestLoad"].Stories["base"].Projects["github.com/kalbasit/tmx"])
		assert.Equal(t, expected["TestLoad"].Stories["base"].Projects["github.com/kalbasit/dotfiles"], c.Profiles["TestLoad"].Stories["base"].Projects["github.com/kalbasit/dotfiles"])
		assert.Equal(t, expected["TestLoad"].Stories["STORY-123"].Name, c.Profiles["TestLoad"].Stories["STORY-123"].Name)
		assert.Equal(t, expected["TestLoad"].Stories["STORY-123"].CodePath, c.Profiles["TestLoad"].Stories["STORY-123"].CodePath)
		assert.Equal(t, expected["TestLoad"].Stories["STORY-123"].ProfileName, c.Profiles["TestLoad"].Stories["STORY-123"].ProfileName)
		assert.Equal(t, expected["TestLoad"].Stories["STORY-123"].Projects["github.com/kalbasit/private"], c.Profiles["TestLoad"].Stories["STORY-123"].Projects["github.com/kalbasit/private"])
	}
	// scan now
	c.Scan()
	assertFn()
	// save it manually
	f, err := AppFs.Create(path.Join(CachePath, strings.Replace(c.Path, "/", "-", -1)+".json"))
	if assert.NoError(t, err) && assert.NoError(t, json.NewEncoder(f).Encode(c)) {
		f.Close()
		// re-create the code so we can try loading
		c := New("/home/kalbasit/code", regexp.MustCompile("^.snapshots$"))
		// load it now
		if assert.NoError(t, c.Load()) {
			assertFn()
		}
	}
}

func TestSave(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFs
	AppFs = afero.NewMemMapFs()
	defer func() { AppFs = oldAppFS }()
	// create the filesystem we want to scan
	prepareFilesystem(t.Name())
	// create a code
	c := New("/home/kalbasit/code", regexp.MustCompile("^.snapshots$"))
	// define the assertion function
	assertFn := func() {
		// assert that it was loaded correctly
		expected := map[string]*Profile{
			"TestSave": &Profile{
				Name:     "TestSave",
				CodePath: "/home/kalbasit/code",
				Stories: map[string]*Story{
					"base": &Story{
						Name:        "base",
						CodePath:    "/home/kalbasit/code",
						ProfileName: "TestSave",
						Projects: map[string]*Project{
							"github.com/kalbasit/tmx": &Project{
								ImportPath:  "github.com/kalbasit/tmx",
								CodePath:    "/home/kalbasit/code",
								ProfileName: "TestSave",
								StoryName:   "base",
							},
							"github.com/kalbasit/dotfiles": &Project{
								ImportPath:  "github.com/kalbasit/dotfiles",
								CodePath:    "/home/kalbasit/code",
								ProfileName: "TestSave",
								StoryName:   "base",
							},
						},
					},
					"STORY-123": &Story{
						Name:        "STORY-123",
						CodePath:    "/home/kalbasit/code",
						ProfileName: "TestSave",
						Projects: map[string]*Project{
							"github.com/kalbasit/private": &Project{
								ImportPath:  "github.com/kalbasit/private",
								CodePath:    "/home/kalbasit/code",
								ProfileName: "TestSave",
								StoryName:   "STORY-123",
							},
						},
					},
				},
			},
		}
		assert.Equal(t, expected["TestSave"].Stories["base"].Name, c.Profiles["TestSave"].Stories["base"].Name)
		assert.Equal(t, expected["TestSave"].Stories["base"].CodePath, c.Profiles["TestSave"].Stories["base"].CodePath)
		assert.Equal(t, expected["TestSave"].Stories["base"].ProfileName, c.Profiles["TestSave"].Stories["base"].ProfileName)
		assert.Equal(t, expected["TestSave"].Stories["base"].Projects["github.com/kalbasit/tmx"], c.Profiles["TestSave"].Stories["base"].Projects["github.com/kalbasit/tmx"])
		assert.Equal(t, expected["TestSave"].Stories["base"].Projects["github.com/kalbasit/dotfiles"], c.Profiles["TestSave"].Stories["base"].Projects["github.com/kalbasit/dotfiles"])
		assert.Equal(t, expected["TestSave"].Stories["STORY-123"].Name, c.Profiles["TestSave"].Stories["STORY-123"].Name)
		assert.Equal(t, expected["TestSave"].Stories["STORY-123"].CodePath, c.Profiles["TestSave"].Stories["STORY-123"].CodePath)
		assert.Equal(t, expected["TestSave"].Stories["STORY-123"].ProfileName, c.Profiles["TestSave"].Stories["STORY-123"].ProfileName)
		assert.Equal(t, expected["TestSave"].Stories["STORY-123"].Projects["github.com/kalbasit/private"], c.Profiles["TestSave"].Stories["STORY-123"].Projects["github.com/kalbasit/private"])
	}
	// scan now
	c.Scan()
	assertFn()
	if assert.NoError(t, c.Save()) {
		// assert the cache file exists
		if stat, err := AppFs.Stat(path.Join(CachePath, strings.Replace(c.Path, "/", "-", -1)+".json")); assert.NoError(t, err) {
			assert.Equal(t, int64(793), stat.Size())
		}
		// re-create the code so we can try loading
		c := New("/home/kalbasit/code", regexp.MustCompile("^.snapshots$"))
		// load it now
		if assert.NoError(t, c.Load()) {
			assertFn()
		}
	}
}

func TestLoadOrScan(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFs
	AppFs = afero.NewMemMapFs()
	defer func() { AppFs = oldAppFS }()
	// create the filesystem we want to scan
	prepareFilesystem(t.Name())
	// create a code
	c := New("/home/kalbasit/code", regexp.MustCompile("^.snapshots$"))
	// define the assertion function
	assertFn := func() {
		// assert that it was loaded correctly
		expected := map[string]*Profile{
			"TestLoadOrScan": &Profile{
				Name:     "TestLoadOrScan",
				CodePath: "/home/kalbasit/code",
				Stories: map[string]*Story{
					"base": &Story{
						Name:        "base",
						CodePath:    "/home/kalbasit/code",
						ProfileName: "TestLoadOrScan",
						Projects: map[string]*Project{
							"github.com/kalbasit/tmx": &Project{
								ImportPath:  "github.com/kalbasit/tmx",
								CodePath:    "/home/kalbasit/code",
								ProfileName: "TestLoadOrScan",
								StoryName:   "base",
							},
							"github.com/kalbasit/dotfiles": &Project{
								ImportPath:  "github.com/kalbasit/dotfiles",
								CodePath:    "/home/kalbasit/code",
								ProfileName: "TestLoadOrScan",
								StoryName:   "base",
							},
						},
					},
					"STORY-123": &Story{
						Name:        "STORY-123",
						CodePath:    "/home/kalbasit/code",
						ProfileName: "TestLoadOrScan",
						Projects: map[string]*Project{
							"github.com/kalbasit/private": &Project{
								ImportPath:  "github.com/kalbasit/private",
								CodePath:    "/home/kalbasit/code",
								ProfileName: "TestLoadOrScan",
								StoryName:   "STORY-123",
							},
						},
					},
				},
			},
		}
		assert.Equal(t, expected["TestLoadOrScan"].Stories["base"].Name, c.Profiles["TestLoadOrScan"].Stories["base"].Name)
		assert.Equal(t, expected["TestLoadOrScan"].Stories["base"].CodePath, c.Profiles["TestLoadOrScan"].Stories["base"].CodePath)
		assert.Equal(t, expected["TestLoadOrScan"].Stories["base"].ProfileName, c.Profiles["TestLoadOrScan"].Stories["base"].ProfileName)
		assert.Equal(t, expected["TestLoadOrScan"].Stories["base"].Projects["github.com/kalbasit/tmx"], c.Profiles["TestLoadOrScan"].Stories["base"].Projects["github.com/kalbasit/tmx"])
		assert.Equal(t, expected["TestLoadOrScan"].Stories["base"].Projects["github.com/kalbasit/dotfiles"], c.Profiles["TestLoadOrScan"].Stories["base"].Projects["github.com/kalbasit/dotfiles"])
		assert.Equal(t, expected["TestLoadOrScan"].Stories["STORY-123"].Name, c.Profiles["TestLoadOrScan"].Stories["STORY-123"].Name)
		assert.Equal(t, expected["TestLoadOrScan"].Stories["STORY-123"].CodePath, c.Profiles["TestLoadOrScan"].Stories["STORY-123"].CodePath)
		assert.Equal(t, expected["TestLoadOrScan"].Stories["STORY-123"].ProfileName, c.Profiles["TestLoadOrScan"].Stories["STORY-123"].ProfileName)
		assert.Equal(t, expected["TestLoadOrScan"].Stories["STORY-123"].Projects["github.com/kalbasit/private"], c.Profiles["TestLoadOrScan"].Stories["STORY-123"].Projects["github.com/kalbasit/private"])
	}
	// load it now
	if assert.NoError(t, c.LoadOrScan()) {
		assertFn()
		// assert the cache file exists
		if stat, err := AppFs.Stat(path.Join(CachePath, strings.Replace(c.Path, "/", "-", -1)+".json")); assert.NoError(t, err) {
			assert.Equal(t, int64(835), stat.Size())
			// load it again
			c.Load()
			assertFn()
		}
	}
}

func TestCodeScan(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFs
	AppFs = afero.NewMemMapFs()
	defer func() { AppFs = oldAppFS }()
	// create the filesystem we want to scan
	prepareFilesystem(t.Name())
	// create a code
	c := &Code{
		Path:           "/home/kalbasit/code",
		ExcludePattern: regexp.MustCompile("^.snapshots$"),
	}
	// scan now
	c.Scan()
	// assert now
	expected := map[string]*Profile{
		"TestCodeScan": &Profile{
			Name:     "TestCodeScan",
			CodePath: "/home/kalbasit/code",
			Stories: map[string]*Story{
				"base": &Story{
					Name:        "base",
					CodePath:    "/home/kalbasit/code",
					ProfileName: "TestCodeScan",
					Projects: map[string]*Project{
						"github.com/kalbasit/tmx": &Project{
							ImportPath:  "github.com/kalbasit/tmx",
							CodePath:    "/home/kalbasit/code",
							ProfileName: "TestCodeScan",
							StoryName:   "base",
						},
						"github.com/kalbasit/dotfiles": &Project{
							ImportPath:  "github.com/kalbasit/dotfiles",
							CodePath:    "/home/kalbasit/code",
							ProfileName: "TestCodeScan",
							StoryName:   "base",
						},
					},
				},
				"STORY-123": &Story{
					Name:        "STORY-123",
					CodePath:    "/home/kalbasit/code",
					ProfileName: "TestCodeScan",
					Projects: map[string]*Project{
						"github.com/kalbasit/private": &Project{
							ImportPath:  "github.com/kalbasit/private",
							CodePath:    "/home/kalbasit/code",
							ProfileName: "TestCodeScan",
							StoryName:   "STORY-123",
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected["TestCodeScan"].Stories["base"].Name, c.Profiles["TestCodeScan"].Stories["base"].Name)
	assert.Equal(t, expected["TestCodeScan"].Stories["base"].CodePath, c.Profiles["TestCodeScan"].Stories["base"].CodePath)
	assert.Equal(t, expected["TestCodeScan"].Stories["base"].ProfileName, c.Profiles["TestCodeScan"].Stories["base"].ProfileName)
	assert.Equal(t, expected["TestCodeScan"].Stories["base"].Projects["github.com/kalbasit/tmx"], c.Profiles["TestCodeScan"].Stories["base"].Projects["github.com/kalbasit/tmx"])
	assert.Equal(t, expected["TestCodeScan"].Stories["base"].Projects["github.com/kalbasit/dotfiles"], c.Profiles["TestCodeScan"].Stories["base"].Projects["github.com/kalbasit/dotfiles"])
	assert.Equal(t, expected["TestCodeScan"].Stories["STORY-123"].Name, c.Profiles["TestCodeScan"].Stories["STORY-123"].Name)
	assert.Equal(t, expected["TestCodeScan"].Stories["STORY-123"].CodePath, c.Profiles["TestCodeScan"].Stories["STORY-123"].CodePath)
	assert.Equal(t, expected["TestCodeScan"].Stories["STORY-123"].ProfileName, c.Profiles["TestCodeScan"].Stories["STORY-123"].ProfileName)
	assert.Equal(t, expected["TestCodeScan"].Stories["STORY-123"].Projects["github.com/kalbasit/private"], c.Profiles["TestCodeScan"].Stories["STORY-123"].Projects["github.com/kalbasit/private"])
}

func TestFindProjectBySessionName(t *testing.T) {
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
	// what do we expect to get back
	expected := map[string]*Project{
		"base": {
			ImportPath:  "github.com/kalbasit/tmx",
			CodePath:    "/home/kalbasit/code",
			ProfileName: "TestFindProjectBySessionName",
			StoryName:   "base",
		},
		"STORY-123": {
			ImportPath:  "github.com/kalbasit/tmx",
			CodePath:    "/home/kalbasit/code",
			ProfileName: "TestFindProjectBySessionName",
			StoryName:   "STORY-123",
		},
	}
	// assert we can find it by using the base workspace
	project, err := c.FindProjectBySessionName("TestFindProjectBySessionName@base=github" + dotChar + "com/kalbasit/tmx")
	if assert.NoError(t, err) {
		assert.Equal(t, expected["base"], project)
	}
	// assert we can find it by using the STORY-123 workspace
	project, err = c.FindProjectBySessionName("TestFindProjectBySessionName@STORY-123=github" + dotChar + "com/kalbasit/tmx")
	if assert.NoError(t, err) {
		assert.Equal(t, expected["STORY-123"], project)
	}
	// assert we can find the same one if the workspace does not exist
	project, err = c.FindProjectBySessionName("TestFindProjectBySessionName@notexistant=github" + dotChar + "com/kalbasit/tmx")
	if assert.NoError(t, err) {
		assert.Equal(t, expected["base"], project)
	}
}

func TestCodeSessionNames(t *testing.T) {
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
		"TestCodeSessionNames@base=github" + dotChar + "com/kalbasit/tmx",
		"TestCodeSessionNames@base=github" + dotChar + "com/kalbasit/dotfiles",
		"TestCodeSessionNames@STORY-123=github" + dotChar + "com/kalbasit/private",
	}
	got := c.SessionNames()
	sort.Strings(want)
	sort.Strings(got)
	assert.Equal(t, want, got)
}
