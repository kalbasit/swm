package tmux

import (
	"errors"

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
)

// Options configures the tmux manager
type Options struct {
	Coder code.Coder

	Profile string

	Story string

	KillPane bool
}

type tmux struct {
	options *Options
}

// New returns a new tmux manager
func New(opts *Options) Manager {
	return &tmux{options: opts}
}
