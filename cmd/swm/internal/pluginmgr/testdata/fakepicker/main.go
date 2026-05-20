// fakepicker is a minimal swm-plugin-picker-fake binary used in pluginmgr tests.
package main

import (
	"context"

	goplugin "github.com/hashicorp/go-plugin"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
	"github.com/kalbasit/swm/sdk/go/handshake"
	"google.golang.org/grpc"
)

type fakePicker struct {
	pluginv1.UnimplementedPickerServer
}

func (f *fakePicker) Info(_ context.Context, _ *pluginv1.Empty) (*pluginv1.PickerInfo, error) {
	return &pluginv1.PickerInfo{
		PluginInfo: &pluginv1.PluginInfo{
			Name:    "fake",
			Version: "0.0.1",
		},
	}, nil
}

type grpcPlugin struct {
	goplugin.NetRPCUnsupportedPlugin
}

func (g *grpcPlugin) GRPCServer(_ *goplugin.GRPCBroker, s *grpc.Server) error {
	pluginv1.RegisterPickerServer(s, &fakePicker{})

	return nil
}

func (g *grpcPlugin) GRPCClient(_ context.Context, _ *goplugin.GRPCBroker, conn *grpc.ClientConn) (interface{}, error) {
	return pluginv1.NewPickerClient(conn), nil
}

func main() {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: handshake.Config,
		Plugins: goplugin.PluginSet{
			"picker": &grpcPlugin{},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})
}
