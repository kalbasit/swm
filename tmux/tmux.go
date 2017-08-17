package tmux

import (
	"fmt"
	"io"
	"log"
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
func (t *tmux) SwitchClient() error {
	// get all the sessions
	sessions, err := t.getSessionNames()
	if err != nil {
		return err
	}
	// select the session using fzf
	sessionName, err := t.withFilter(func(stdin io.WriteCloser) {
		for _, sess := range sessions {
			io.WriteString(stdin, sess)
			io.WriteString(stdin, "\n")
		}
	})
	if err != nil {
		log.Fatalf("error filtering the session: %s", err)
	}

	return nil
}

// getSessionNameProjects returns a map of a project session name to the project
func (t *tmux) getSessionNameProjects() (map[string]code.Project, error) {
	var sessions []string

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
	for _, prj := range story.Projects() {
		sessions = append(sessions, t.sessionNameForProject(profile.Name(), story.Name(), prj))
	}

	return sessions, nil
}

func (t *tmux) sessionNameForProject(profileName, storyName string, prj code.Project) string {
	safeProjectName := strings.Replace(strings.Replace(prj.ImportPath(), ".", dotChar, -1), ":", colonChar, -1)
	return fmt.Sprintf("%s@%s=%s", profileName, storyName, safeProjectName)
}
