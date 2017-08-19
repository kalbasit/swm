package tmux

// Manager represents a TMUX manager
type Manager interface {
	// SwitchClient switches the TMUX to a different client
	SwitchClient() error

	// VimExit will close any running vim for this session, saving any changed
	// file.
	VimExit() error
}
