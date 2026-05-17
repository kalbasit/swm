// Package session provides the SDK surface for swm session plugins.
// A session plugin manages terminal-multiplexer workspaces (e.g. tmux, zellij).
package session

import (
	"context"
	"os"

	"google.golang.org/grpc"

	hclog "github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/sdk/go/handshake"
	"github.com/kalbasit/swm/sdk/go/internal/pluginlog"
)

// Plugin is the interface a session plugin must implement.
// It is identical to pluginv1.SessionServer, so implementors can embed
// pluginv1.UnimplementedSessionServer and override only the methods they need.
type Plugin = pluginv1.SessionServer

// GRPCPlugin implements go-plugin's GRPCPlugin interface for the Session capability.
type GRPCPlugin struct {
	goplugin.NetRPCUnsupportedPlugin
	Impl Plugin
}

// GRPCClient returns a SessionClient backed by the provided connection.
func (p *GRPCPlugin) GRPCClient(_ context.Context, _ *goplugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return pluginv1.NewSessionClient(conn), nil
}

// GRPCServer registers the Session gRPC server with the provided gRPC server instance.
func (p *GRPCPlugin) GRPCServer(_ *goplugin.GRPCBroker, s *grpc.Server) error {
	pluginv1.RegisterSessionServer(s, p.Impl)

	return nil
}

// Serve starts the go-plugin gRPC server for the given Plugin implementation.
// It blocks until the host signals the plugin to exit.
func Serve(impl Plugin) error {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: handshake.Config,
		Logger: hclog.New(&hclog.LoggerOptions{
			Level:      pluginlog.Level(),
			JSONFormat: true,
			Output:     os.Stderr,
		}),
		Plugins: goplugin.PluginSet{
			"session": &GRPCPlugin{Impl: impl},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})

	return nil
}
