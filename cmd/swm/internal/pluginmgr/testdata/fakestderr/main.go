// fakestderr is a minimal swm-plugin-vcs-* binary used to test stderr forwarding.
// It writes a known marker to stderr before serving, then behaves like a normal VCS plugin.
package main

import (
	"context"
	"fmt"
	"os"

	goplugin "github.com/hashicorp/go-plugin"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
	"github.com/kalbasit/swm/sdk/go/handshake"
	"google.golang.org/grpc"
)

const stderrMarker = "FAKESTDERR_MARKER: hello from fakestderr"

type fakeVCS struct {
	pluginv1.UnimplementedVCSServer
}

func (f *fakeVCS) Info(_ context.Context, _ *pluginv1.Empty) (*pluginv1.VCSInfo, error) {
	return &pluginv1.VCSInfo{
		PluginInfo: &pluginv1.PluginInfo{
			Name:    "fakestderr",
			Version: "0.0.1",
		},
		ProjectMarkers: []string{".git"},
	}, nil
}

type grpcPlugin struct {
	goplugin.NetRPCUnsupportedPlugin
}

func (g *grpcPlugin) GRPCServer(_ *goplugin.GRPCBroker, s *grpc.Server) error {
	pluginv1.RegisterVCSServer(s, &fakeVCS{})

	return nil
}

func (g *grpcPlugin) GRPCClient(_ context.Context, _ *goplugin.GRPCBroker, conn *grpc.ClientConn) (interface{}, error) {
	return pluginv1.NewVCSClient(conn), nil
}

func main() {
	fmt.Fprintln(os.Stderr, stderrMarker)

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: handshake.Config,
		Plugins: goplugin.PluginSet{
			"vcs": &grpcPlugin{},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})
}
