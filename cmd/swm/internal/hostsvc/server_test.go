package hostsvc_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/core/story"
	"github.com/kalbasit/swm/cmd/swm/internal/hostsvc"
)

func setupServer(t *testing.T, cfg *config.Config, codeRoot string) pluginv1.HostClient {
	t.Helper()

	storiesDir := t.TempDir()
	store := story.NewJSONStore(storiesDir)
	resolver := layout.NewResolver(codeRoot)

	srv, err := hostsvc.NewServer(cfg, resolver, store)
	require.NoError(t, err)

	t.Cleanup(func() { srv.Stop() })

	conn, err := grpc.NewClient(
		srv.SocketPath(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() }) //nolint:errcheck,gosec // best-effort in test cleanup

	return pluginv1.NewHostClient(conn)
}

func TestGetConfig_Scoping(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CodeRoot: "/tmp/code",
		Plugins: config.Plugins{
			Config: map[string]map[string]any{
				"vcs-git": {"foo": "bar"},
				"other":   {"secret": "hidden"},
			},
		},
	}

	client := setupServer(t, cfg, "/tmp/code")

	resp, err := client.GetConfig(context.Background(), &pluginv1.GetConfigRequest{PluginName: "vcs-git"})
	require.NoError(t, err)
	require.Contains(t, string(resp.GetToml()), "foo")
	require.NotContains(t, string(resp.GetToml()), "secret")
}

func TestGetCodeRoot(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: "/my/code"}
	client := setupServer(t, cfg, "/my/code")

	resp, err := client.GetCodeRoot(context.Background(), &pluginv1.Empty{})
	require.NoError(t, err)
	require.Equal(t, "/my/code", resp.GetPath())
}

func TestListProjects_MarkerDetection(t *testing.T) {
	t.Parallel()

	codeRoot := t.TempDir()
	repoDir := filepath.Join(codeRoot, "repositories", "github.com", "kalbasit", "swm")
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0o750))

	// Another dir without .git — should not appear.
	nonGitDir := filepath.Join(codeRoot, "repositories", "github.com", "other", "project")
	require.NoError(t, os.MkdirAll(nonGitDir, 0o750))

	cfg := &config.Config{CodeRoot: codeRoot}
	client := setupServer(t, cfg, codeRoot)

	stream, err := client.ListProjects(context.Background(), &pluginv1.ListProjectsRequest{})
	require.NoError(t, err)

	var projects []*pluginv1.Project

	for {
		p, err := stream.Recv()
		if err != nil {
			break
		}

		projects = append(projects, p)
	}

	require.Len(t, projects, 1)
	require.Equal(t, "github.com", projects[0].GetHost())
	require.Equal(t, []string{"kalbasit", "swm"}, projects[0].GetSegments())
}
