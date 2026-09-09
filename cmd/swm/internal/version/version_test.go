package version_test

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kalbasit/swm/cmd/swm/internal/version"
)

const (
	// tag is a stand-in for any released version.
	tag = "v1.2.3"

	// develPlaceholder is what the Go toolchain records as the main module's
	// version for a binary built from a work tree.
	develPlaceholder = "(devel)"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		linked string
		info   *debug.BuildInfo
		want   string
	}{
		{
			name:   "link-time version wins over recorded build info",
			linked: tag,
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v0.0.1"},
			},
			want: tag,
		},
		{
			name:   "recorded main module version is used when nothing was linked in",
			linked: "",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: tag},
			},
			want: tag,
		},
		{
			name:   "clean work tree falls through the devel placeholder to the revision",
			linked: "",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: develPlaceholder},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc123"},
					{Key: "vcs.modified", Value: "false"},
				},
			},
			want: "devel+abc123",
		},
		{
			name:   "modified work tree marks the revision dirty",
			linked: "",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: develPlaceholder},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc123"},
					{Key: "vcs.modified", Value: "true"},
				},
			},
			want: "devel+abc123-dirty",
		},
		{
			name:   "no build information at all still reports a development build",
			linked: "",
			info:   nil,
			want:   "devel",
		},
		{
			name:   "build information without a recorded revision reports a development build",
			linked: "",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: develPlaceholder},
			},
			want: "devel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, version.Resolve(tt.linked, tt.info))
		})
	}
}

// TestVersionIsAlwaysReported pins the one guarantee Version() can make about
// any binary: it always has something to say. Which branch of Resolve answers
// depends on how this test binary was built — a release build links a tag into
// the test binary too — so the shape of the answer is asserted by the table
// above, where the inputs are known.
func TestVersionIsAlwaysReported(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, version.Version())
}
