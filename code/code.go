package code

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/afero"
)

var (
	// AppFs represents the filesystem of the app. It is exported to be used as a
	// test helper.
	AppFs afero.Fs

	// CachePath represents the path at which the cache will be stored
	CachePath string

	// ErrProjectNotFound is returned if the session name did not yield a project
	// we know about
	ErrProjectNotFound = errors.New("project not found")

	// ErrCodePathEmpty is returned if Code.Path is empty or invalid
	ErrCodePathEmpty = errors.New("code path is empty or does not exist")

	sessionNameRegex = regexp.MustCompile("^([a-zA-Z0-9_-]+)@([a-zA-Z0-9_-]+)=(.*)$")
)

func init() {
	// initialize AppFs to use the OS filesystem
	AppFs = afero.NewOsFs()

	// initialize the cache path
	CachePath = path.Join(os.Getenv("HOME"), ".cache", "tmx")
	if _, err := AppFs.Stat(CachePath); os.IsNotExist(err) {
		if err := AppFs.MkdirAll(CachePath, 0755); err != nil {
			log.Fatalf("error creating the directory %q to store cache: %s", CachePath, err)
		}
	}
}

// Code represents a code folder that follows the following structure:
// code/
// |-- profile1
// |   |-- base
// |   |   |-- src
// |   |       |-- go.import.path
// |   |-- stories
// |   |   |-- STORY-123
// |   |       |-- src
// |   |           |-- go.import.path
// |-- profile2
// |   |-- base
// |   |   |-- src
// |   |       |-- go.import.path
// |   |-- stories
// |   |   |-- STORY-123
// |   |       |-- src
// |   |           |-- go.import.path
type Code struct {
	// Path is the base path of this profile
	Path string

	// ExcludePattern is a list of patterns to ignore
	ExcludePattern *regexp.Regexp

	Profiles map[string]*Profile
}

// New returns a new empty Code, caller must call Load to load from cache or
// scan the code directory
func New(p string, ignore *regexp.Regexp) *Code { return &Code{Path: p, ExcludePattern: ignore} }

// LoadOrScan loads the code from the cache (if it exists), otherwise it will
// initiate a full scan and save it in cache.
func (c *Code) LoadOrScan() error {
	// validate the Code, we cannot load an invalid Code
	if err := c.validate(); err != nil {
		return err
	}
	// try loading from cache
	if err := c.Load(); err != nil {
		c.Scan()
		return c.Save()
	}

	return nil
}

// Load load the code from cache
func (c *Code) Load() error {
	// validate the Code, we cannot load an invalid Code
	if err := c.validate(); err != nil {
		return err
	}
	// parse the cache file now
	f, err := AppFs.Open(path.Join(CachePath, strings.Replace(c.Path, "/", "-", -1)+".json"))
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(c)
}

// Save saves the code to the cache file
func (c *Code) Save() error {
	f, err := AppFs.Create(path.Join(CachePath, strings.Replace(c.Path, "/", "-", -1)+".json"))
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(c)
}

// Scan scans the entire profile to build the workspaces
func (c *Code) Scan() {
	// initialize the variables
	var wg sync.WaitGroup
	c.Profiles = make(map[string]*Profile)
	// read the profile and scan all profiles
	entries, err := afero.ReadDir(AppFs, c.Path)
	if err != nil {
		log.Printf("error reading the directory %q: %s", c.Path, err)
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			// does this folder match the exclude pattern?
			if c.ExcludePattern != nil && c.ExcludePattern.MatchString(entry.Name()) {
				continue
			}
			// create the workspace
			p := &Profile{
				Name:     entry.Name(),
				CodePath: c.Path,
			}
			// start scanning it
			wg.Add(1)
			go func() {
				defer wg.Done()
				p.Scan()
			}()
			// add it to the profile
			c.Profiles[entry.Name()] = p
		}
	}
	wg.Wait()
}

// FindProjectBySessionName returns the project represented by the session name
func (c *Code) FindProjectBySessionName(name string) (*Project, error) {
	ms := sessionNameRegex.FindStringSubmatch(name)
	if len(ms) == 4 {
		if p := c.Profiles[ms[1]]; p != nil {
			// load the workspace
			w := p.Stories[ms[2]]
			if w == nil {
				// if workspace is empty, try looking for the same project under the
				// base project.
				return p.BaseStory().FindProjectBySessionName(ms[3])
			}
			if prj, err := w.FindProjectBySessionName(ms[3]); err == nil && prj != nil {
				return prj, nil
			}

			return p.BaseStory().FindProjectBySessionName(ms[3])
		}
	}

	return nil, ErrProjectNotFound
}

// SessionNames returns the session names for projects in all workspaces in all profiles
func (c *Code) SessionNames() []string {
	var res []string
	for _, profile := range c.Profiles {
		for _, workspace := range profile.Stories {
			for _, project := range workspace.Projects {
				res = append(res, project.SessionName())
			}
		}
	}

	return res
}

func (c *Code) validate() error {
	if c.Path == "" {
		return ErrCodePathEmpty
	}
	if _, err := AppFs.Stat(c.Path); err != nil {
		return ErrCodePathEmpty
	}

	return nil
}
