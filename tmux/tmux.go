package tmux

import (
	"fmt"
	"strings"

	"github.com/kalbasit/swm/code"
)

const (
	dotChar   = "\u2022"
	colonChar = "\uFF1A"
)

// Options configures the tmux manager
type Options struct {
	Coder code.Coder

	Profile string

	Story string
}

type tmux struct {
	options *Options
}

// New returns a new tmux manager
func New(opts *Options) Manager {
	return &tmux{options: opts}
}

// SwitchClient switches the TMUX to a different client
func (t *tmux) SwitchClient() error { return nil }

// 	// get all the sessions
// 	sessions, err := t.getSessionNames()
// 	if err != nil {
// 		return err
// 	}
// 	// select the session using fzf
// 	sessionName, err := t.withFilter(func(stdin io.WriteCloser) {
// 		for _, sess := range sessions {
// 			io.WriteString(stdin, sess)
// 			io.WriteString(stdin, "\n")
// 		}
// 	})
// 	if err != nil {
// 		log.Fatalf("error filtering the session: %s", err)
// 	}
//
// 	return nil
// }

// getSessionNameProjects returns a map of a project session name to the project
func (t *tmux) getSessionNameProjects() (map[string]code.Project, error) {
	// get the profile
	profile, err := t.options.Coder.Profile(t.options.Profile)
	if err != nil {
		return nil, err
	}
	// get the story
	story, err := profile.Story(t.options.Story)
	if err != nil {
		return nil, err
	}
	// loop over all projects and get the session name
	sessionNameProjects := make(map[string]code.Project)
	for _, prj := range story.Projects() {
		// generate the session name
		k := fmt.Sprintf("%s@%s=%s",
			profile.Name(),
			story.Name(),
			strings.Replace(strings.Replace(prj.ImportPath(), ".", dotChar, -1), ":", colonChar, -1))
		// assign it to the map
		sessionNameProjects[k] = prj
	}

	return sessionNameProjects, nil
}
