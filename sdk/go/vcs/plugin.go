// Package vcs provides the SDK surface for swm VCS plugins.
// A VCS plugin handles cloning, worktree management, and URL parsing
// for a specific version-control system (e.g. git, jujutsu).
package vcs

import (
	"errors"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

// ErrNotImplemented is returned by Serve when the gRPC transport has not been
// wired up yet (Phase 0 stub). Replaced with real logic in Phase 1.
var ErrNotImplemented = errors.New("vcs.Serve: not yet implemented")

// Plugin is the interface a VCS plugin must implement.
// It is identical to pluginv1.VCSServer, so implementors can embed
// pluginv1.UnimplementedVCSServer and override only the methods they need.
type Plugin = pluginv1.VCSServer

// Serve starts the go-plugin gRPC server for the given Plugin implementation.
// It blocks until the host signals the plugin to exit.
//
// Phase-0 stub: returns ErrNotImplemented. Phase 1 wires the go-plugin transport.
func Serve(_ Plugin) error {
	return ErrNotImplemented
}
