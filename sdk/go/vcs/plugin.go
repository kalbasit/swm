// Package vcs provides the SDK surface for swm VCS plugins.
// A VCS plugin handles cloning, worktree management, and URL parsing
// for a specific version-control system (e.g. git, jujutsu).
package vcs

import (
	"context"

	"google.golang.org/grpc"

	goplugin "github.com/hashicorp/go-plugin"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/sdk/go/handshake"
)

// Plugin is the interface a VCS plugin must implement.
// It is identical to pluginv1.VCSServer, so implementors can embed
// pluginv1.UnimplementedVCSServer and override only the methods they need.
type Plugin = pluginv1.VCSServer

// GRPCPlugin implements go-plugin's GRPCPlugin interface for the VCS capability.
type GRPCPlugin struct {
	goplugin.NetRPCUnsupportedPlugin
	Impl Plugin
}

// GRPCClient returns a VCSClient backed by the provided connection.
func (p *GRPCPlugin) GRPCClient(_ context.Context, _ *goplugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return pluginv1.NewVCSClient(conn), nil
}

// GRPCServer registers the VCS gRPC server with the provided gRPC server instance.
func (p *GRPCPlugin) GRPCServer(_ *goplugin.GRPCBroker, s *grpc.Server) error {
	pluginv1.RegisterVCSServer(s, p.Impl)

	return nil
}

// Serve starts the go-plugin gRPC server for the given Plugin implementation.
// It blocks until the host signals the plugin to exit.
func Serve(impl Plugin) error {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: handshake.Config,
		Plugins: goplugin.PluginSet{
			"vcs": &GRPCPlugin{Impl: impl},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})

	return nil
}
