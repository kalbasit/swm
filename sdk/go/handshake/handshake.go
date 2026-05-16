// Package handshake exports the go-plugin handshake constants shared by the
// swm host and all plugins. Both sides must import this package to guarantee
// they agree on the magic cookie and protocol version.
package handshake

import "github.com/hashicorp/go-plugin"

const (
	// MagicCookieKey is the environment variable the host sets when launching a plugin.
	// A plugin that sees this key (with the right value) knows it is running in plugin mode.
	MagicCookieKey = "SWM_PLUGIN_MAGIC_COOKIE"

	// MagicCookieValue is the expected value of MagicCookieKey.
	// A plugin binary launched directly (without this value) prints a user-friendly error
	// and exits instead of starting the gRPC server.
	MagicCookieValue = "swm-plugin-v1"

	// ProtocolVersion is bumped when the host/plugin wire protocol changes in a
	// backwards-incompatible way. Plugins must advertise the same version.
	ProtocolVersion = 1
)

// Config is the go-plugin HandshakeConfig that both the host (client side) and
// every plugin (server side) must pass to plugin.Serve / plugin.NewClient.
var Config = plugin.HandshakeConfig{ //nolint:gochecknoglobals // package-level constant by go-plugin convention
	ProtocolVersion:  ProtocolVersion,
	MagicCookieKey:   MagicCookieKey,
	MagicCookieValue: MagicCookieValue,
}
