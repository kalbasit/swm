package tmx

import (
	"errors"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/afero"
)

var (
	// AppFs represents the filesystem of the app. It is exported to be used as a
	// test helper.
	AppFs afero.Fs

	// ErrSessionNotFound is returned if the session name did not yield a project
	// we know about
	ErrSessionNotFound = errors.New("project not found")

	sessionNameRegex = regexp.MustCompile("^([a-zA-Z0-9]+)@([a-zA-Z0-9]+)=(.*)$")
)

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

// FindProjectBySessionName returns the project represented by the session name
func (c *Code) FindProjectBySessionName(name string) (*Project, error) {
	ms := sessionNameRegex.FindStringSubmatch(name)
	if len(ms) == 4 {
		if profile := c.Profiles[ms[1]]; profile != nil {
			if workspace := profile.Workspaces[ms[2]]; workspace != nil {
				if project := workspace.Projects[strings.Replace(strings.Replace(ms[3], dotChar, ".", -1), colonChar, ":", -1)]; project != nil {
					return project, nil
				}
			}
		}
	}

	return nil, ErrSessionNotFound
}

func (c *Code) SessionNames() []string {
	var res []string
	for _, profile := range c.Profiles {
		for _, workspace := range profile.Workspaces {
			for _, project := range workspace.Projects {
				res = append(res, project.SessionName())
			}
		}
	}

	return res
}
