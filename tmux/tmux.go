package tmux

import (
	"errors"
	"log"
	"os/exec"

	"github.com/kalbasit/swm/code"
)

const (
	dotChar   = "\u2022"
	colonChar = "\uFF1A"
)

var (
	// ErrProjectNotFoundForGivenSessionName is returned by SwitchClient if the
	// selected session (via fzf currently) was not found. This usually means
	// that fzf output was not one of the input.
	ErrProjectNotFoundForGivenSessionName = errors.New("project not found for the given session name")

	// tmuxPath is the PATH to the tmux executable
	tmuxPath string
)

func init() {
	var err error
	tmuxPath, err = exec.LookPath("tmux")
	if err != nil {
		log.Fatalf("error looking up the tmux executable, is it installed? %s", err)
	}
}

// Options configures the tmux manager
type Options struct {
	// Coder represents the coder instance
	Coder code.Coder

	// Profile represents the profile we are going to use to compute the list of
	// available projects as well as the ACTIVE_PROFILE of new sessions.
	Profile string

	// Story represents the story we are going to use to compute the list of
	// available projects.
	Story string

	// KiilPane when set, will close the TMUX pane running swm
	KillPane bool
}

// tmux implements the Manager interface
type tmux struct{ options *Options }

// New returns a new tmux manager
func New(opts *Options) Manager {
	return &tmux{options: opts}
}
