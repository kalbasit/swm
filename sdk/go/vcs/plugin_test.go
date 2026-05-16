package vcs_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	goplugin "github.com/hashicorp/go-plugin"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/sdk/go/handshake"
	"github.com/kalbasit/swm/sdk/go/vcs"
)

// Compile-time interface check: GRPCPlugin must implement goplugin.GRPCPlugin.
var _ goplugin.GRPCPlugin = (*vcs.GRPCPlugin)(nil)

func TestHandshakeConfig(t *testing.T) {
	t.Parallel()

	require.Equal(t, handshake.MagicCookieKey, handshake.Config.MagicCookieKey)
	require.Equal(t, handshake.MagicCookieValue, handshake.Config.MagicCookieValue)
	require.Equal(t, handshake.ProtocolVersion, int(handshake.Config.ProtocolVersion))
}

func TestGRPCPlugin_GRPCServer(t *testing.T) {
	t.Parallel()

	p := &vcs.GRPCPlugin{Impl: pluginv1.UnimplementedVCSServer{}}
	srv := grpc.NewServer()
	err := p.GRPCServer(nil, srv)
	require.NoError(t, err)

	info := srv.GetServiceInfo()
	require.Contains(t, info, "swm.plugin.v1.VCS")
}

func TestGRPCPlugin_GRPCClient(t *testing.T) {
	t.Parallel()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	defer lis.Close() //nolint:errcheck // best-effort close in test cleanup

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	defer conn.Close() //nolint:errcheck // best-effort close in test cleanup

	p := &vcs.GRPCPlugin{}
	raw, err := p.GRPCClient(context.Background(), nil, conn)
	require.NoError(t, err)

	_, ok := raw.(pluginv1.VCSClient)
	require.True(t, ok, "GRPCClient must return a pluginv1.VCSClient")
}
