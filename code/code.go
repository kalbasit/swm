package code

import (
	"errors"
	"log"
	"regexp"
	"sync"

	"github.com/spf13/afero"
)

var (
	// AppFs represents the filesystem of the app. It is exported to be used as a
	// test helper.
	AppFs afero.Fs

	// ErrCodePathEmpty is returned if Code.Path is empty or invalid
	ErrCodePathEmpty = errors.New("code path is empty or does not exist")

	// ErrProfileNoFound is returned if the profile was not found
	ErrProfileNoFound = errors.New("profile not found")

	// ErrStoryNoFound is returned if the story was not found
	ErrStoryNoFound = errors.New("story not found")

	// ErrProjectNotFound is returned if the session name did not yield a project
	// we know about
	ErrProjectNotFound = errors.New("project not found")
)

func init() {
	// initialize AppFs to use the OS filesystem
	AppFs = afero.NewOsFs()
}

// code implements the coder interface
type code struct {
	// path is the base path of this profile
	path string

	// excludePattern is a list of patterns to ignore
	excludePattern *regexp.Regexp

	profiles map[string]*profile
}

// New returns a new empty Code, caller must call Load to load from cache or
// scan the code directory
func New(p string, ignore *regexp.Regexp) *code { return &code{path: p, excludePattern: ignore} }

// Path returns the absolute path of this coder
func (c *code) Path() string { return c.path }

// Profile returns the profile given it's name or an error if no profile with
// this name was found
func (c *code) Profile(name string) (Profile, error) {
	p, ok := c.profiles[name]
	if !ok {
		return nil, ErrProfileNoFound
	}
	return p, nil
}

// Scan loads the code from the cache (if it exists), otherwise it will
// initiate a full scan and save it in cache.
func (c *code) Scan() error {
	// validate the Code, we cannot load an invalid Code
	if err := c.validate(); err != nil {
		return err
	}
	c.scan()

	return nil
}

// scan scans the entire profile to build the workspaces
func (c *code) scan() {
	// initialize the variables
	var wg sync.WaitGroup
	c.profiles = make(map[string]*profile)
	// read the profile and scan all profiles
	entries, err := afero.ReadDir(AppFs, c.path)
	if err != nil {
		log.Printf("error reading the directory %q: %s", c.path, err)
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			// does this folder match the exclude pattern?
			if c.excludePattern != nil && c.excludePattern.MatchString(entry.Name()) {
				continue
			}
			// create the workspace
			p := &profile{
				name: entry.Name(),
				code: c,
			}
			// start scanning it
			wg.Add(1)
			go func() {
				defer wg.Done()
				p.scan()
			}()
			// add it to the profile
			c.profiles[entry.Name()] = p
		}
	}
	wg.Wait()
}

func (c *code) validate() error {
	if c.path == "" {
		return ErrCodePathEmpty
	}
	if _, err := AppFs.Stat(c.path); err != nil {
		return ErrCodePathEmpty
	}

	return nil
}
