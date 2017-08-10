package tmx

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
				Workspaces: map[string]*Workspace{
					"base": &Workspace{
						Name:        "base",
						CodePath:    "/home/kalbasit/code",
						ProfileName: "TestLoad",
						Projects: map[string]*Project{
							"github.com/kalbasit/tmx": &Project{
								ImportPath:    "github.com/kalbasit/tmx",
								CodePath:      "/home/kalbasit/code",
								ProfileName:   "TestLoad",
								WorkspaceName: "base",
							},
							"github.com/kalbasit/dotfiles": &Project{
								ImportPath:    "github.com/kalbasit/dotfiles",
								CodePath:      "/home/kalbasit/code",
								ProfileName:   "TestLoad",
								WorkspaceName: "base",
							},
						},
					},
					"STORY-123": &Workspace{
						Name:        "STORY-123",
						CodePath:    "/home/kalbasit/code",
						ProfileName: "TestLoad",
						Projects: map[string]*Project{
							"github.com/kalbasit/private": &Project{
								ImportPath:    "github.com/kalbasit/private",
								CodePath:      "/home/kalbasit/code",
								ProfileName:   "TestLoad",
								WorkspaceName: "STORY-123",
							},
						},
					},
				},
			},
		}
		assert.Equal(t, expected["TestLoad"].Workspaces["base"].Name, c.Profiles["TestLoad"].Workspaces["base"].Name)
		assert.Equal(t, expected["TestLoad"].Workspaces["base"].CodePath, c.Profiles["TestLoad"].Workspaces["base"].CodePath)
		assert.Equal(t, expected["TestLoad"].Workspaces["base"].ProfileName, c.Profiles["TestLoad"].Workspaces["base"].ProfileName)
		assert.Equal(t, expected["TestLoad"].Workspaces["base"].Projects["github.com/kalbasit/tmx"], c.Profiles["TestLoad"].Workspaces["base"].Projects["github.com/kalbasit/tmx"])
		assert.Equal(t, expected["TestLoad"].Workspaces["base"].Projects["github.com/kalbasit/dotfiles"], c.Profiles["TestLoad"].Workspaces["base"].Projects["github.com/kalbasit/dotfiles"])
		assert.Equal(t, expected["TestLoad"].Workspaces["STORY-123"].Name, c.Profiles["TestLoad"].Workspaces["STORY-123"].Name)
		assert.Equal(t, expected["TestLoad"].Workspaces["STORY-123"].CodePath, c.Profiles["TestLoad"].Workspaces["STORY-123"].CodePath)
		assert.Equal(t, expected["TestLoad"].Workspaces["STORY-123"].ProfileName, c.Profiles["TestLoad"].Workspaces["STORY-123"].ProfileName)
		assert.Equal(t, expected["TestLoad"].Workspaces["STORY-123"].Projects["github.com/kalbasit/private"], c.Profiles["TestLoad"].Workspaces["STORY-123"].Projects["github.com/kalbasit/private"])
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
				Workspaces: map[string]*Workspace{
					"base": &Workspace{
						Name:        "base",
						CodePath:    "/home/kalbasit/code",
						ProfileName: "TestSave",
						Projects: map[string]*Project{
							"github.com/kalbasit/tmx": &Project{
								ImportPath:    "github.com/kalbasit/tmx",
								CodePath:      "/home/kalbasit/code",
								ProfileName:   "TestSave",
								WorkspaceName: "base",
							},
							"github.com/kalbasit/dotfiles": &Project{
								ImportPath:    "github.com/kalbasit/dotfiles",
								CodePath:      "/home/kalbasit/code",
								ProfileName:   "TestSave",
								WorkspaceName: "base",
							},
						},
					},
					"STORY-123": &Workspace{
						Name:        "STORY-123",
						CodePath:    "/home/kalbasit/code",
						ProfileName: "TestSave",
						Projects: map[string]*Project{
							"github.com/kalbasit/private": &Project{
								ImportPath:    "github.com/kalbasit/private",
								CodePath:      "/home/kalbasit/code",
								ProfileName:   "TestSave",
								WorkspaceName: "STORY-123",
							},
						},
					},
				},
			},
		}
		assert.Equal(t, expected["TestSave"].Workspaces["base"].Name, c.Profiles["TestSave"].Workspaces["base"].Name)
		assert.Equal(t, expected["TestSave"].Workspaces["base"].CodePath, c.Profiles["TestSave"].Workspaces["base"].CodePath)
		assert.Equal(t, expected["TestSave"].Workspaces["base"].ProfileName, c.Profiles["TestSave"].Workspaces["base"].ProfileName)
		assert.Equal(t, expected["TestSave"].Workspaces["base"].Projects["github.com/kalbasit/tmx"], c.Profiles["TestSave"].Workspaces["base"].Projects["github.com/kalbasit/tmx"])
		assert.Equal(t, expected["TestSave"].Workspaces["base"].Projects["github.com/kalbasit/dotfiles"], c.Profiles["TestSave"].Workspaces["base"].Projects["github.com/kalbasit/dotfiles"])
		assert.Equal(t, expected["TestSave"].Workspaces["STORY-123"].Name, c.Profiles["TestSave"].Workspaces["STORY-123"].Name)
		assert.Equal(t, expected["TestSave"].Workspaces["STORY-123"].CodePath, c.Profiles["TestSave"].Workspaces["STORY-123"].CodePath)
		assert.Equal(t, expected["TestSave"].Workspaces["STORY-123"].ProfileName, c.Profiles["TestSave"].Workspaces["STORY-123"].ProfileName)
		assert.Equal(t, expected["TestSave"].Workspaces["STORY-123"].Projects["github.com/kalbasit/private"], c.Profiles["TestSave"].Workspaces["STORY-123"].Projects["github.com/kalbasit/private"])
	}
	// scan now
	c.Scan()
	assertFn()
	if assert.NoError(t, c.Save()) {
		// assert the cache file exists
		if stat, err := AppFs.Stat(path.Join(CachePath, strings.Replace(c.Path, "/", "-", -1)+".json")); assert.NoError(t, err) {
			assert.Equal(t, int64(808), stat.Size())
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
				Workspaces: map[string]*Workspace{
					"base": &Workspace{
						Name:        "base",
						CodePath:    "/home/kalbasit/code",
						ProfileName: "TestLoadOrScan",
						Projects: map[string]*Project{
							"github.com/kalbasit/tmx": &Project{
								ImportPath:    "github.com/kalbasit/tmx",
								CodePath:      "/home/kalbasit/code",
								ProfileName:   "TestLoadOrScan",
								WorkspaceName: "base",
							},
							"github.com/kalbasit/dotfiles": &Project{
								ImportPath:    "github.com/kalbasit/dotfiles",
								CodePath:      "/home/kalbasit/code",
								ProfileName:   "TestLoadOrScan",
								WorkspaceName: "base",
							},
						},
					},
					"STORY-123": &Workspace{
						Name:        "STORY-123",
						CodePath:    "/home/kalbasit/code",
						ProfileName: "TestLoadOrScan",
						Projects: map[string]*Project{
							"github.com/kalbasit/private": &Project{
								ImportPath:    "github.com/kalbasit/private",
								CodePath:      "/home/kalbasit/code",
								ProfileName:   "TestLoadOrScan",
								WorkspaceName: "STORY-123",
							},
						},
					},
				},
			},
		}
		assert.Equal(t, expected["TestLoadOrScan"].Workspaces["base"].Name, c.Profiles["TestLoadOrScan"].Workspaces["base"].Name)
		assert.Equal(t, expected["TestLoadOrScan"].Workspaces["base"].CodePath, c.Profiles["TestLoadOrScan"].Workspaces["base"].CodePath)
		assert.Equal(t, expected["TestLoadOrScan"].Workspaces["base"].ProfileName, c.Profiles["TestLoadOrScan"].Workspaces["base"].ProfileName)
		assert.Equal(t, expected["TestLoadOrScan"].Workspaces["base"].Projects["github.com/kalbasit/tmx"], c.Profiles["TestLoadOrScan"].Workspaces["base"].Projects["github.com/kalbasit/tmx"])
		assert.Equal(t, expected["TestLoadOrScan"].Workspaces["base"].Projects["github.com/kalbasit/dotfiles"], c.Profiles["TestLoadOrScan"].Workspaces["base"].Projects["github.com/kalbasit/dotfiles"])
		assert.Equal(t, expected["TestLoadOrScan"].Workspaces["STORY-123"].Name, c.Profiles["TestLoadOrScan"].Workspaces["STORY-123"].Name)
		assert.Equal(t, expected["TestLoadOrScan"].Workspaces["STORY-123"].CodePath, c.Profiles["TestLoadOrScan"].Workspaces["STORY-123"].CodePath)
		assert.Equal(t, expected["TestLoadOrScan"].Workspaces["STORY-123"].ProfileName, c.Profiles["TestLoadOrScan"].Workspaces["STORY-123"].ProfileName)
		assert.Equal(t, expected["TestLoadOrScan"].Workspaces["STORY-123"].Projects["github.com/kalbasit/private"], c.Profiles["TestLoadOrScan"].Workspaces["STORY-123"].Projects["github.com/kalbasit/private"])
	}
	// load it now
	if assert.NoError(t, c.LoadOrScan()) {
		assertFn()
		// assert the cache file exists
		if stat, err := AppFs.Stat(path.Join(CachePath, strings.Replace(c.Path, "/", "-", -1)+".json")); assert.NoError(t, err) {
			assert.Equal(t, int64(850), stat.Size())
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
			Workspaces: map[string]*Workspace{
				"base": &Workspace{
					Name:        "base",
					CodePath:    "/home/kalbasit/code",
					ProfileName: "TestCodeScan",
					Projects: map[string]*Project{
						"github.com/kalbasit/tmx": &Project{
							ImportPath:    "github.com/kalbasit/tmx",
							CodePath:      "/home/kalbasit/code",
							ProfileName:   "TestCodeScan",
							WorkspaceName: "base",
						},
						"github.com/kalbasit/dotfiles": &Project{
							ImportPath:    "github.com/kalbasit/dotfiles",
							CodePath:      "/home/kalbasit/code",
							ProfileName:   "TestCodeScan",
							WorkspaceName: "base",
						},
					},
				},
				"STORY-123": &Workspace{
					Name:        "STORY-123",
					CodePath:    "/home/kalbasit/code",
					ProfileName: "TestCodeScan",
					Projects: map[string]*Project{
						"github.com/kalbasit/private": &Project{
							ImportPath:    "github.com/kalbasit/private",
							CodePath:      "/home/kalbasit/code",
							ProfileName:   "TestCodeScan",
							WorkspaceName: "STORY-123",
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected["TestCodeScan"].Workspaces["base"].Name, c.Profiles["TestCodeScan"].Workspaces["base"].Name)
	assert.Equal(t, expected["TestCodeScan"].Workspaces["base"].CodePath, c.Profiles["TestCodeScan"].Workspaces["base"].CodePath)
	assert.Equal(t, expected["TestCodeScan"].Workspaces["base"].ProfileName, c.Profiles["TestCodeScan"].Workspaces["base"].ProfileName)
	assert.Equal(t, expected["TestCodeScan"].Workspaces["base"].Projects["github.com/kalbasit/tmx"], c.Profiles["TestCodeScan"].Workspaces["base"].Projects["github.com/kalbasit/tmx"])
	assert.Equal(t, expected["TestCodeScan"].Workspaces["base"].Projects["github.com/kalbasit/dotfiles"], c.Profiles["TestCodeScan"].Workspaces["base"].Projects["github.com/kalbasit/dotfiles"])
	assert.Equal(t, expected["TestCodeScan"].Workspaces["STORY-123"].Name, c.Profiles["TestCodeScan"].Workspaces["STORY-123"].Name)
	assert.Equal(t, expected["TestCodeScan"].Workspaces["STORY-123"].CodePath, c.Profiles["TestCodeScan"].Workspaces["STORY-123"].CodePath)
	assert.Equal(t, expected["TestCodeScan"].Workspaces["STORY-123"].ProfileName, c.Profiles["TestCodeScan"].Workspaces["STORY-123"].ProfileName)
	assert.Equal(t, expected["TestCodeScan"].Workspaces["STORY-123"].Projects["github.com/kalbasit/private"], c.Profiles["TestCodeScan"].Workspaces["STORY-123"].Projects["github.com/kalbasit/private"])
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
	// assert now
	expected := &Project{
		ImportPath:    "github.com/kalbasit/tmx",
		CodePath:      "/home/kalbasit/code",
		ProfileName:   "TestFindProjectBySessionName",
		WorkspaceName: "base",
	}
	project, err := c.FindProjectBySessionName("TestFindProjectBySessionName@base=github" + dotChar + "com/kalbasit/tmx")
	if assert.NoError(t, err) {
		assert.Equal(t, expected, project)
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
