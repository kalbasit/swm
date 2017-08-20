package code

import (
	"path"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

// profile represents the profile
type profile struct {
	// code links back to parent coder
	code *code

	// name is the name of the profile
	name string

	// stories is a list of workspaces
	stories unsafe.Pointer // type *map[string]*story
}

func newProfile(c *code, name string) *profile {
	stories := make(map[string]*story)
	return &profile{
		name:    name,
		code:    c,
		stories: unsafe.Pointer(&stories),
	}
}

// Coder returns the coder under which this exists
func (p *profile) Coder() Coder { return p.code }

// Name returns the name of the profile
func (p *profile) Name() string { return p.name }

// Base returns the base story
func (p *profile) Base() Story { return p.Story(baseStoryName) }

// Path returns the absolute path to this profile
func (p *profile) Path() string { return path.Join(p.code.Path(), p.name) }

// Story returns the story given it's name or an error if no story with this
// name was found
func (p *profile) Story(name string) Story {
	// get the stories out of profiles
	stories := p.getStories()
	// fetch the story out
	s, ok := stories[name]
	if !ok {
		return p.addStory(name)
	}

	return s
}

// getStories return the map of stories
func (p *profile) getStories() map[string]*story {
	return *(*map[string]*story)(atomic.LoadPointer(&p.stories))
}

// addStory adds the story to the list of stories
func (p *profile) addStory(name string) *story {
	if s, ok := p.getStories()[name]; ok {
		return s
	}
	s := newStory(p, name)
	for {
		storiesPtr := atomic.LoadPointer(&p.stories)
		stories := *(*map[string]*story)(storiesPtr)
		stories[name] = s
		if atomic.CompareAndSwapPointer(&p.stories, storiesPtr, unsafe.Pointer(&stories)) {
			return s
		}
	}
}

// scan scans the entire profile to build the workspaces
func (p *profile) scan() {
	// initialize the variables
	var wg sync.WaitGroup
	// add the base story
	wg.Add(1)
	go func() {
		s := p.addStory(baseStoryName)
		s.scan()
		wg.Done()
	}()
	// read the profile and scan all stories
	entries, err := afero.ReadDir(AppFS, path.Join(p.Path(), "stories"))
	if err != nil {
		log.Error().Str("path", path.Join(p.Path(), "stories")).Msgf("error reading the directory: %s", err)
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			// create the story
			log.Debug().Str("profile", p.name).Msgf("found story: %s", entry.Name())
			wg.Add(1)
			go func(name string) {
				s := p.addStory(name)
				s.scan()
				wg.Done()
			}(entry.Name())
		}
	}

	wg.Wait()
}
