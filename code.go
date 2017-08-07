package tmx

import (
	"log"
	"regexp"
	"sync"

	"github.com/spf13/afero"
)

// AppFs represents the filesystem of the app. It is exported to be used as a
// test helper.
var AppFs afero.Fs

func init() {
	AppFs = afero.NewOsFs()
}

type Code struct {
	// Path is the base path of this profile
	Path string

	// ExcludePattern is a list of patterns to ignore
	ExcludePattern *regexp.Regexp

	Profiles map[string]*Profile
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
