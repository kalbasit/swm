package tmux

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/kalbasit/swm/ifaces"
	"github.com/rs/zerolog/log"
)

const (
	dotChar   = "\u2022"
	colonChar = "\uFF1A"
)

var (
	// tmuxPath is the PATH to the tmux executable
	tmuxPath string
)

func init() {
	var err error
	tmuxPath, err = exec.LookPath("tmux")
	if err != nil {
		log.Fatal().Msgf("error looking up the tmux executable, is it installed? %s", err)
	}
}

// Options configures the tmux manager
type Options struct {
	// Coder represents the coder instance
	Coder ifaces.Code

	// StoryName represents the story we are going to use to compute the list of
	// available projects.
	StoryName string
}

// tmux implements the Manager interface
type tmux struct{ options *Options }

// socketName returns the session name
func (t *tmux) socketName() string {
	return strings.Replace(fmt.Sprintf("swm-%s", t.options.StoryName), "/", "_", -1)
}

// New returns a new tmux manager
func New(opts *Options) Manager {
	return &tmux{options: opts}
}
