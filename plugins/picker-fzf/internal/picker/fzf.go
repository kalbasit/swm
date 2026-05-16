// Package picker implements the swm Picker capability using the system fzf binary.
package picker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

// buildVersion is set via -ldflags at build time.
var buildVersion = "dev" //nolint:gochecknoglobals // set via ldflags at link time

// Fzf implements pluginv1.PickerServer by wrapping the system fzf binary.
type Fzf struct {
	pluginv1.UnimplementedPickerServer
	fzfBin  string
	ttyPath string // defaults to /dev/tty; overridable in tests
}

// New returns a Fzf instance using the system fzf binary.
func New() (*Fzf, error) {
	bin, err := exec.LookPath("fzf")
	if err != nil {
		return nil, fmt.Errorf("fzf binary not found in PATH: %w", err)
	}

	return &Fzf{fzfBin: bin, ttyPath: "/dev/tty"}, nil
}

// NewWithBin returns a Fzf instance with injected binary and tty paths (for tests).
func NewWithBin(fzfBin, ttyPath string) *Fzf {
	return &Fzf{fzfBin: fzfBin, ttyPath: ttyPath}
}

// Info returns metadata about this Picker plugin.
func (f *Fzf) Info(_ context.Context, _ *pluginv1.Empty) (*pluginv1.PickerInfo, error) {
	return &pluginv1.PickerInfo{
		PluginInfo: &pluginv1.PluginInfo{
			Name:    "fzf",
			Version: buildVersion,
		},
	}, nil
}

// Pick implements bidirectional streaming: receives PickItem candidates from the host,
// presents them to the user via fzf, and streams a single PickResult back.
func (f *Fzf) Pick(stream grpc.BidiStreamingServer[pluginv1.PickItem, pluginv1.PickResult]) error {
	// Accumulate all candidates from the host stream.
	var candidates []*pluginv1.PickItem

	for {
		item, err := stream.Recv()
		if err != nil {
			break
		}

		candidates = append(candidates, item)
	}

	if len(candidates) == 0 {
		return status.Errorf(codes.Aborted, "no candidates received")
	}

	// Open the TTY so fzf can render its TUI in the user's terminal.
	// fzf uses /dev/tty for its interactive interface when stdin is a pipe.
	tty, err := os.OpenFile(f.ttyPath, os.O_RDWR, 0)
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "no TTY available: %v", err)
	}

	defer tty.Close() //nolint:errcheck // best-effort close of /dev/tty

	// Build the candidate list for fzf: "<key>\t<display>\n" per line.
	// --with-nth=2 tells fzf to show only the second tab-delimited field (display),
	// but the full line (including key) is output on selection.
	var input bytes.Buffer
	for _, c := range candidates {
		fmt.Fprintf(&input, "%s\t%s\n", c.GetKey(), c.GetDisplay())
	}

	var outBuf bytes.Buffer

	cmd := exec.Command(f.fzfBin, "--with-nth=2", "--delimiter=\t") //nolint:gosec // fzfBin from LookPath
	cmd.Stdin = &input
	cmd.Stdout = &outBuf
	cmd.Stderr = tty // fzf renders its TUI on stderr (attached to /dev/tty)

	if err := cmd.Run(); err != nil {
		// fzf exits 1 when the user presses Escape or Ctrl-C.
		return status.Errorf(codes.Aborted, "picker cancelled")
	}

	selected := strings.TrimSuffix(outBuf.String(), "\n")
	if selected == "" {
		return status.Errorf(codes.Aborted, "no item selected")
	}

	// Output is "<key>\t<display>"; extract the key (first field).
	key := strings.SplitN(selected, "\t", 2)[0]

	return stream.Send(&pluginv1.PickResult{Key: key})
}
