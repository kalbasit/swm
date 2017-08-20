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
func (p *profile) addStory(name string) (*story, error) {
	if s, ok := p.getStories()[name]; ok {
		return s, ErrStoryAlreadyExists
	}
	s := newStory(p, name)
	for {
		storiesPtr := atomic.LoadPointer(&p.stories)
		stories := *(*map[string]*story)(storiesPtr)
		stories[name] = s
		if atomic.CompareAndSwapPointer(&p.stories, storiesPtr, unsafe.Pointer(&stories)) {
			return s, nil
		}
	}
}

func (p *profile) scan() {
	// initialize the variables
	var wg sync.WaitGroup
	out := make(chan string, 1000)
	// start the workers
	wg.Add(1)
	go p.scanWorker(&wg, out)
	// start the reducer
	reducerQuit := make(chan struct{})
	go p.scanReducer(&wg, out, reducerQuit)
	// wait for the workers to return
	wg.Wait()
	// ask the reducer to die
	close(out)
	<-reducerQuit
}

func (p *profile) scanReducer(wg *sync.WaitGroup, out chan string, quit chan struct{}) {
	// iterate over the channel
	for {
		select {
		case name, ok := <-out:
			if !ok {
				close(quit)
				return
			}
			s, err := p.addStory(name)
			if err != nil && err != ErrStoryAlreadyExists {
				log.Error().Err(err).Str("story-name", name).Msg("error occurred adding the story")
				continue
			}
			if s != nil {
				wg.Add(1)
				go func() {
					s.scan()
					wg.Done()
				}()
			}
		}
	}
}

// scan scans the entire profile to build the workspaces
func (p *profile) scanWorker(wg *sync.WaitGroup, out chan string) {
	// create the base story
	out <- baseStoryName
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
			out <- entry.Name()
		}
	}
}
