package pluginmgr_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/pluginmgr"
)

const fakePluginName = "fake"

var (
	fakeVCSBin    string
	fakeStderrBin string
)

// syncBuffer is a thread-safe bytes.Buffer for use as a stderr target.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.buf.String()
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.buf.Write(p)
}

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "pluginmgr-test-*")
	if err != nil {
		panic("creating temp dir: " + err.Error())
	}

	defer os.RemoveAll(dir) //nolint:errcheck // best-effort cleanup in test teardown

	fakeVCSBin = filepath.Join(dir, "swm-plugin-vcs-fake")

	cmd := exec.Command("go", "build", "-o", fakeVCSBin, //nolint:gosec // building from trusted test paths
		"./testdata/fakevcs/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		panic("building fake plugin: " + err.Error())
	}

	fakeStderrBin = filepath.Join(dir, "swm-plugin-vcs-fakestderr")

	cmd = exec.Command("go", "build", "-o", fakeStderrBin, //nolint:gosec // building from trusted test paths
		"./testdata/fakestderr/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		panic("building fakestderr plugin: " + err.Error())
	}

	os.Exit(m.Run())
}

func newCfg(session, vcs string) *config.Config {
	return &config.Config{
		CodeRoot:     "/tmp/code",
		DefaultStory: "_default",
		Plugins: config.Plugins{
			Session: session,
			VCS:     vcs,
		},
	}
}

func TestDiscover_Path(t *testing.T) {
	// Cannot use t.Parallel with t.Setenv.
	dir := filepath.Dir(fakeVCSBin)
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+string(filepath.ListSeparator)+oldPath)

	// Copy fake binary to name expected by discovery.
	dst := filepath.Join(dir, "swm-plugin-vcs-git")
	data, err := os.ReadFile(fakeVCSBin) //nolint:gosec // reading trusted test binary
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(dst, data, 0o755)) //nolint:gosec // binary must be executable

	mgr := pluginmgr.New(newCfg("", "git"), "")
	defer mgr.Close() //nolint:errcheck // best-effort cleanup in test teardown

	raw, err := mgr.Get(context.Background(), "vcs")
	require.NoError(t, err)
	require.NotNil(t, raw)
}

func TestGet_LazyLaunch(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CodeRoot:     "/tmp/code",
		DefaultStory: "_default",
		Plugins: config.Plugins{
			VCS: fakePluginName,
			Paths: map[string]string{
				fakePluginName: fakeVCSBin,
			},
		},
	}

	mgr := pluginmgr.New(cfg, "")
	defer mgr.Close() //nolint:errcheck // best-effort cleanup in test teardown

	// First call triggers launch.
	raw, err := mgr.Get(context.Background(), "vcs")
	require.NoError(t, err)
	require.NotNil(t, raw)

	// Second call returns cached client.
	raw2, err := mgr.Get(context.Background(), "vcs")
	require.NoError(t, err)
	require.Equal(t, raw, raw2)
}

func TestGet_ClientIsVCSClient(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Plugins: config.Plugins{
			VCS: fakePluginName,
			Paths: map[string]string{
				fakePluginName: fakeVCSBin,
			},
		},
	}

	mgr := pluginmgr.New(cfg, "")
	defer mgr.Close() //nolint:errcheck // best-effort cleanup in test teardown

	raw, err := mgr.Get(context.Background(), "vcs")
	require.NoError(t, err)

	client, ok := raw.(pluginv1.VCSClient)
	require.True(t, ok)

	info, err := client.Info(context.Background(), &pluginv1.Empty{})
	require.NoError(t, err)
	require.Equal(t, fakePluginName, info.GetPluginInfo().GetName())
}

func TestGet_MissingPlugin(t *testing.T) {
	t.Parallel()

	mgr := pluginmgr.New(newCfg("", "nonexistent"), "")
	defer mgr.Close() //nolint:errcheck // best-effort cleanup in test teardown

	_, err := mgr.Get(context.Background(), "vcs")
	require.Error(t, err)
}

func TestGet_UnconfiguredCapability(t *testing.T) {
	t.Parallel()

	mgr := pluginmgr.New(newCfg("", ""), "")
	defer mgr.Close() //nolint:errcheck // best-effort cleanup in test teardown

	_, err := mgr.Get(context.Background(), "vcs")
	require.Error(t, err)
}

func TestClose_Cleanup(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Plugins: config.Plugins{
			VCS: fakePluginName,
			Paths: map[string]string{
				fakePluginName: fakeVCSBin,
			},
		},
	}

	mgr := pluginmgr.New(cfg, "")
	_, err := mgr.Get(context.Background(), "vcs")
	require.NoError(t, err)

	require.NoError(t, mgr.Close())

	// After close, a new Get would re-launch (no cache left).
	raw, err := mgr.Get(context.Background(), "vcs")
	require.NoError(t, err)
	require.NotNil(t, raw)

	require.NoError(t, mgr.Close())
}

func TestGet_PluginStderrForwarded(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Plugins: config.Plugins{
			VCS: "fakestderr",
			Paths: map[string]string{
				"fakestderr": fakeStderrBin,
			},
		},
	}

	var sink syncBuffer

	mgr := pluginmgr.New(cfg, "", pluginmgr.WithStderr(&sink))
	defer mgr.Close() //nolint:errcheck // best-effort cleanup in test teardown

	_, err := mgr.Get(context.Background(), "vcs")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return bytes.Contains([]byte(sink.String()), []byte("FAKESTDERR_MARKER"))
	}, 5*time.Second, 10*time.Millisecond)
}

func TestGet_NoDebugLogs_AtWarnLevel(t *testing.T) { //nolint:paralleltest // mutates slog.Default global state
	original := slog.Default()

	t.Cleanup(func() { slog.SetDefault(original) })
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn})))

	cfg := &config.Config{
		Plugins: config.Plugins{
			VCS: fakePluginName,
			Paths: map[string]string{
				fakePluginName: fakeVCSBin,
			},
		},
	}

	var sink syncBuffer

	mgr := pluginmgr.New(cfg, "", pluginmgr.WithStderr(&sink))
	defer mgr.Close() //nolint:errcheck // best-effort cleanup in test teardown

	_, err := mgr.Get(context.Background(), "vcs")
	require.NoError(t, err)

	// Wait for go-plugin startup to complete and confirm no debug/trace output arrives.
	require.Eventually(t, func() bool {
		out := sink.String()

		return !strings.Contains(out, "[DEBUG]") && !strings.Contains(out, "[TRACE]")
	}, 1*time.Second, 10*time.Millisecond)

	require.NotContains(t, sink.String(), `"@level":"debug"`)
	require.NotContains(t, sink.String(), `"@level":"trace"`)
}

func TestGet_DebugLogs_AtDebugLevel(t *testing.T) { //nolint:paralleltest // mutates slog.Default global state
	original := slog.Default()

	t.Cleanup(func() { slog.SetDefault(original) })
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})))

	cfg := &config.Config{
		Plugins: config.Plugins{
			VCS: fakePluginName,
			Paths: map[string]string{
				fakePluginName: fakeVCSBin,
			},
		},
	}

	var sink syncBuffer

	mgr := pluginmgr.New(cfg, "", pluginmgr.WithStderr(&sink))
	defer mgr.Close() //nolint:errcheck // best-effort cleanup in test teardown

	_, err := mgr.Get(context.Background(), "vcs")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return bytes.Contains([]byte(sink.String()), []byte("[DEBUG]"))
	}, 5*time.Second, 10*time.Millisecond)
}
