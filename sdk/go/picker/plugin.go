// Package picker provides the SDK surface for swm picker plugins.
// A picker plugin provides interactive item-selection UI (e.g. fzf, skim).
package picker

import (
	"context"

	"google.golang.org/grpc"

	goplugin "github.com/hashicorp/go-plugin"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/sdk/go/handshake"
)

// Plugin is the interface a picker plugin must implement.
// It is identical to pluginv1.PickerServer, so implementors can embed
// pluginv1.UnimplementedPickerServer and override only the methods they need.
type Plugin = pluginv1.PickerServer

// GRPCPlugin implements go-plugin's GRPCPlugin interface for the Picker capability.
type GRPCPlugin struct {
	goplugin.NetRPCUnsupportedPlugin
	Impl Plugin
}

// GRPCClient returns a PickerClient backed by the provided connection.
func (p *GRPCPlugin) GRPCClient(_ context.Context, _ *goplugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return pluginv1.NewPickerClient(conn), nil
}

// GRPCServer registers the Picker gRPC server with the provided gRPC server instance.
func (p *GRPCPlugin) GRPCServer(_ *goplugin.GRPCBroker, s *grpc.Server) error {
	pluginv1.RegisterPickerServer(s, p.Impl)

	return nil
}

// NewClient returns a PickerClient backed by the provided connection.
func NewClient(conn *grpc.ClientConn) pluginv1.PickerClient {
	return pluginv1.NewPickerClient(conn)
}

// Serve starts the go-plugin gRPC server for the given Plugin implementation.
// It blocks until the host signals the plugin to exit.
func Serve(impl Plugin) error {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: handshake.Config,
		Plugins: goplugin.PluginSet{
			"picker": &GRPCPlugin{Impl: impl},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})

	return nil
}
