// Package forge provides the SDK surface for swm forge plugins.
// A forge plugin talks to a code-hosting platform (e.g. GitHub, GitLab)
// to manage pull requests and other forge operations.
package forge

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

// Plugin is the interface a forge plugin must implement.
// It is identical to pluginv1.ForgeServer, so implementors can embed
// pluginv1.UnimplementedForgeServer and override only the methods they need.
type Plugin = pluginv1.ForgeServer

// GRPCPlugin implements go-plugin's GRPCPlugin interface for the Forge capability.
type GRPCPlugin struct {
	goplugin.NetRPCUnsupportedPlugin
	Impl Plugin
}

// GRPCClient returns a ForgeClient backed by the provided connection.
func (p *GRPCPlugin) GRPCClient(_ context.Context, _ *goplugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return pluginv1.NewForgeClient(conn), nil
}

// GRPCServer registers the Forge gRPC server with the provided gRPC server instance.
func (p *GRPCPlugin) GRPCServer(_ *goplugin.GRPCBroker, s *grpc.Server) error {
	pluginv1.RegisterForgeServer(s, p.Impl)

	return nil
}

// NewClient returns a ForgeClient backed by the provided connection.
func NewClient(conn *grpc.ClientConn) pluginv1.ForgeClient {
	return pluginv1.NewForgeClient(conn)
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
			"forge": &GRPCPlugin{Impl: impl},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})

	return nil
}
