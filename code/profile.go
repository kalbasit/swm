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
		// no story found, create one
		s = newStory(p, name)
		stories[name] = s
		p.setStories(stories)
	}

	return s
}

// getStories return the map of stories
func (p *profile) getStories() map[string]*story {
	return *(*map[string]*story)(atomic.LoadPointer(&p.stories))
}

// setStories sets the map of stories
func (p *profile) setStories(stories map[string]*story) {
	atomic.StorePointer(&p.stories, unsafe.Pointer(&stories))
}

// addStory adds the story to the list of stories
func (p *profile) addStory(s *story) error {
	if _, ok := p.getStories()[s.name]; ok {
		return ErrStoryAlreadyExists
	}
	for {
		storiesPtr := atomic.LoadPointer(&p.stories)
		stories := *(*map[string]*story)(storiesPtr)
		stories[s.name] = s
		if atomic.CompareAndSwapPointer(&p.stories, storiesPtr, unsafe.Pointer(&stories)) {
			break
		}
	}
}

// scan scans the entire profile to build the workspaces
func (p *profile) scan() {
	// initialize the variables
	var wg sync.WaitGroup
	stories := make(map[string]*story)
	// create the base story
	stories[baseStoryName] = newStory(p, baseStoryName)
	wg.Add(1)
	go func() {
		defer wg.Done()
		stories[baseStoryName].scan()
	}()
	// read the profile and scan all workspaces
	entries, err := afero.ReadDir(AppFS, path.Join(p.Path(), "stories"))
	if err != nil {
		log.Error().Str("path", path.Join(p.Path(), "stories")).Msgf("error reading the directory: %s", err)
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			// create the story
			log.Debug().Str("profile", p.name).Msgf("found story: %s", entry.Name())
			s := newStory(p, entry.Name())
			// start scanning it
			wg.Add(1)
			go func() {
				defer wg.Done()
				s.scan()
			}()
			// add it to the profile
			stories[entry.Name()] = s
		}
	}
	wg.Wait()

	// set the stories to p now
	p.setStories(stories)
}
