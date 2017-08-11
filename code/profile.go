package code

import (
	"log"
	"path"
	"sync"

	"github.com/spf13/afero"
)

// profile represents the profile
type profile struct {
	// code links back to parent coder
	code *code

	// name is the name of the profile
	name string

	// stories is a list of workspaces
	stories map[string]*story
}

// BaseStory returns the base workspace
func (p *profile) BaseStory() *story {
	return p.stories[baseStoryName]
}

// Path returns the absolute path of the profile
func (p *profile) Path() string {
	return path.Join(p.code.Path(), p.name)
}

// Scan scans the entire profile to build the workspaces
func (p *profile) Scan() {
	// initialize the variables
	var wg sync.WaitGroup
	p.stories = make(map[string]*story)
	// create the base story
	p.stories[baseStoryName] = &story{
		name:    baseStoryName,
		profile: p,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.stories[baseStoryName].scan()
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
			s := &story{
				name:    entry.Name(),
				profile: p,
			}
			// start scanning it
			wg.Add(1)
			go func() {
				defer wg.Done()
				s.scan()
			}()
			// add it to the profile
			p.stories[entry.Name()] = s
		}
	}
	wg.Wait()
}

// SessionNames returns the session names for projects in all workspaces of this profile
func (p *profile) SessionNames() []string {
	var res []string
	for _, story := range p.stories {
		for _, project := range story.projects {
			res = append(res, project.SessionName())
		}
	}

	return res
}
