package picker_test

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/plugins/picker-fzf/internal/picker"
)

const (
	testSWMProject        = "github.com/kalbasit/swm"
	testSWMProjectDisplay = "kalbasit/swm"
)

var fakefzfBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "picker-fzf-fakefzf-*")
	if err != nil {
		panic("create temp dir: " + err.Error())
	}

	defer os.RemoveAll(dir) //nolint:errcheck // best-effort cleanup in TestMain

	fakefzfBin = filepath.Join(dir, "fakefzf")

	buildCmd := exec.Command("go", "build", "-o", fakefzfBin, "./testdata/fakefzf") //nolint:gosec // test build

	out, err := buildCmd.CombinedOutput()
	if err != nil {
		panic("build fakefzf: " + string(out))
	}

	os.Exit(m.Run())
}

func newFzf(t *testing.T) *picker.Fzf {
	t.Helper()

	ttyFile := filepath.Join(t.TempDir(), "tty")
	if err := os.WriteFile(ttyFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	return picker.NewWithBin(fakefzfBin, ttyFile)
}

func TestInfo(t *testing.T) {
	t.Parallel()

	f := newFzf(t)
	info, err := f.Info(context.Background(), &pluginv1.Empty{})
	require.NoError(t, err)
	require.Equal(t, "fzf", info.GetPluginInfo().GetName())
}

func TestPick_SuccessfulSelection(t *testing.T) {
	t.Parallel()

	f := newFzf(t)

	items := []*pluginv1.PickItem{
		{Key: testSWMProject, Display: testSWMProjectDisplay},
		{Key: "github.com/kalbasit/dotfiles", Display: "kalbasit/dotfiles"},
	}

	stream := newPickStream(items)
	err := f.Pick(stream)
	require.NoError(t, err)

	results := stream.sent
	require.Len(t, results, 1)
	// fakefzf outputs the first candidate line: "<key>\t<display>"
	require.Equal(t, testSWMProject, results[0].GetKey())
}

func TestPick_UserCancels(t *testing.T) {
	// Cannot be parallel — uses t.Setenv.
	f := newFzf(t)
	t.Setenv("FAKEFZF_EXIT", "1")

	items := []*pluginv1.PickItem{
		{Key: testSWMProject, Display: testSWMProjectDisplay},
	}

	stream := newPickStream(items)
	err := f.Pick(stream)
	require.Error(t, err)
	require.Equal(t, codes.Aborted, status.Code(err))
}

func TestPick_NoTTY(t *testing.T) {
	t.Parallel()

	// Use a non-existent path to simulate a missing TTY.
	f := picker.NewWithBin(fakefzfBin, "/nonexistent/dev/tty")

	items := []*pluginv1.PickItem{
		{Key: testSWMProject, Display: testSWMProjectDisplay},
	}

	stream := newPickStream(items)
	err := f.Pick(stream)
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

// pickStream is an in-process mock of grpc.BidiStreamingServer[PickItem, PickResult].
type pickStream struct {
	pluginv1.Picker_PickServer
	items []*pluginv1.PickItem
	pos   int
	sent  []*pluginv1.PickResult
}

func newPickStream(items []*pluginv1.PickItem) *pickStream {
	return &pickStream{items: items}
}

func (s *pickStream) Context() context.Context {
	return context.Background()
}

func (s *pickStream) Recv() (*pluginv1.PickItem, error) {
	if s.pos >= len(s.items) {
		return nil, io.EOF
	}

	item := s.items[s.pos]
	s.pos++

	return item, nil
}

func (s *pickStream) Send(r *pluginv1.PickResult) error {
	s.sent = append(s.sent, r)

	return nil
}
