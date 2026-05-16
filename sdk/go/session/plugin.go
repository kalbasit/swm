// Package session provides the SDK surface for swm session plugins.
// A session plugin manages terminal-multiplexer workspaces (e.g. tmux, zellij).
package session

import (
	"errors"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

// ErrNotImplemented is returned by Serve when the gRPC transport has not been
// wired up yet (Phase 0 stub). Replaced with real logic in Phase 1.
var ErrNotImplemented = errors.New("session.Serve: not yet implemented")

// Plugin is the interface a session plugin must implement.
// It is identical to pluginv1.SessionServer, so implementors can embed
// pluginv1.UnimplementedSessionServer and override only the methods they need.
type Plugin = pluginv1.SessionServer

// Serve starts the go-plugin gRPC server for the given Plugin implementation.
// It blocks until the host signals the plugin to exit.
//
// Phase-0 stub: returns ErrNotImplemented. Phase 1 wires the go-plugin transport.
func Serve(_ Plugin) error {
	return ErrNotImplemented
}
