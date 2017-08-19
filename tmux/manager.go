package tmux

// Manager represents a TMUX manager
type Manager interface {
	// SwitchClient switches the TMUX to a different client
	// KiilPane when set, will close the TMUX pane running swm
	SwitchClient(killPane bool) error

	// VimExit will close any running vim for this session, saving any changed
	// file.
	VimExit() error
}
