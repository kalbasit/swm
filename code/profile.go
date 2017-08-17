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

// Coder returns the coder under which this exists
func (p *profile) Coder() Coder { return p.code }

// Name returns the name of the profile
func (p *profile) Name() string { return p.name }

// Base returns the base story
func (p *profile) Base() Story { return p.stories[baseStoryName] }

// Path returns the absolute path to this profile
func (p *profile) Path() string { return path.Join(p.code.Path(), p.name) }

// Story returns the story given it's name or an error if no story with this
// name was found
func (p *profile) Story(name string) (Story, error) {
	s, ok := p.stories[name]
	if !ok {
		return nil, ErrStoryNoFound
	}

	return s, nil
}

// scan scans the entire profile to build the workspaces
func (p *profile) scan() {
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
