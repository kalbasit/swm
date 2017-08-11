package code

import (
	"log"
	"path"
	"sync"

	"github.com/spf13/afero"
)

// BaseStory represents the name of the base story
const BaseStory = "base"

// Profile represents the profile
type Profile struct {
	// Name is the name of the profile
	Name string

	// CodePath is the path of Code.Path
	CodePath string

	// Stories is a list of workspaces
	Stories map[string]*Story
}

// BaseStory returns the base workspace
func (p *Profile) BaseStory() *Story {
	return p.Stories[BaseStory]
}

// Path returns the absolute path of the profile
func (p *Profile) Path() string {
	return path.Join(p.CodePath, p.Name)
}

// Scan scans the entire profile to build the workspaces
func (p *Profile) Scan() {
	// initialize the variables
	var wg sync.WaitGroup
	p.Stories = make(map[string]*Story)
	// create the base story
	p.Stories[BaseStory] = &Story{
		Name:        BaseStory,
		CodePath:    p.CodePath,
		ProfileName: p.Name,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.Stories[BaseStory].Scan()
	}()
	// read the profile and scan all workspaces
	entries, err := afero.ReadDir(AppFs, path.Join(p.Path(), "stories"))
	if err != nil {
		log.Printf("error reading the directory %q: %s", path.Join(p.Path(), "stories"), err)
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			// create the workspace
			s := &Story{
				Name:        entry.Name(),
				CodePath:    p.CodePath,
				ProfileName: p.Name,
			}
			// start scanning it
			wg.Add(1)
			go func() {
				defer wg.Done()
				s.Scan()
			}()
			// add it to the profile
			p.Stories[entry.Name()] = s
		}
	}
	wg.Wait()
}

// SessionNames returns the session names for projects in all workspaces of this profile
func (p *Profile) SessionNames() []string {
	var res []string
	for _, story := range p.Stories {
		for _, project := range story.Projects {
			res = append(res, project.SessionName())
		}
	}

	return res
}
