package layout_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kalbasit/swm/plugins/session-tmux/internal/layout"
)

const (
	testSock    = "/run/user/1000/swm/tmux/feat-x.sock"
	testSession = "github•com/org/repo"
)

// recorder is a RunFunc that records calls and returns auto-incrementing pane IDs
// for display-message and split-window calls.
type recorder struct {
	mu      sync.Mutex
	calls   []string // each call as "arg1 arg2 ..."
	paneSeq int
}

func (r *recorder) run(_ context.Context, args ...string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls = append(r.calls, strings.Join(args, " "))

	joined := strings.Join(args, " ")
	if strings.Contains(joined, "display-message") || strings.Contains(joined, "split-window") {
		id := "%" + string(rune('0'+r.paneSeq))
		r.paneSeq++
		return id, nil
	}

	return "", nil
}

func (r *recorder) hasCall(substr string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, c := range r.calls {
		if strings.Contains(c, substr) {
			return true
		}
	}

	return false
}

func (r *recorder) callCount(substr string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	n := 0

	for _, c := range r.calls {
		if strings.Contains(c, substr) {
			n++
		}
	}

	return n
}

func flex(n int) *int { return &n }

func TestApply_SingleWindowNoPanes(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	cfg := &layout.Config{
		Windows: []layout.Window{
			{Name: "editor"},
		},
	}

	err := layout.Apply(context.Background(), rec.run, testSock, testSession, cfg)
	require.NoError(t, err)
	require.True(t, rec.hasCall("rename-window"), "must rename default window")
	require.True(t, rec.hasCall("editor"), "window name must appear in rename-window call")
}

func TestApply_TwoWindows(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	cfg := &layout.Config{
		Windows: []layout.Window{
			{Name: "editor"},
			{Name: "shell"},
		},
	}

	err := layout.Apply(context.Background(), rec.run, testSock, testSession, cfg)
	require.NoError(t, err)
	require.True(t, rec.hasCall("rename-window"), "first window must be renamed")
	require.True(t, rec.hasCall("new-window"), "second window must be created with new-window")
	require.True(t, rec.hasCall("shell"), "second window name must appear")
}

func TestApply_TwoPanesEqualFlex(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	cfg := &layout.Config{
		Windows: []layout.Window{
			{
				Name: "main",
				Panes: []layout.Pane{
					{Commands: []string{"nvim ."}},
					{Commands: []string{"bash"}},
				},
			},
		},
	}

	err := layout.Apply(context.Background(), rec.run, testSock, testSession, cfg)
	require.NoError(t, err)
	require.Equal(t, 1, rec.callCount("split-window"), "exactly one split for two panes")
	require.True(t, rec.hasCall("-p 50"), "equal flex must produce 50% split")
	require.True(t, rec.hasCall("nvim ."), "pane command must be sent")
	require.True(t, rec.hasCall("bash"), "pane command must be sent")
}

func TestApply_ThreePanesEqualFlex(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	cfg := &layout.Config{
		Windows: []layout.Window{
			{
				Name: "main",
				Panes: []layout.Pane{
					{Commands: []string{"cmd0"}},
					{Commands: []string{"cmd1"}},
					{Commands: []string{"cmd2"}},
				},
			},
		},
	}

	err := layout.Apply(context.Background(), rec.run, testSock, testSession, cfg)
	require.NoError(t, err)
	require.Equal(t, 2, rec.callCount("split-window"), "two splits for three panes")
	require.True(t, rec.hasCall("-p 66"), "first split gives 66% to remaining (floor division)")
	require.True(t, rec.hasCall("-p 50"), "second split halves the remainder")
}

func TestApply_WeightedFlex(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	cfg := &layout.Config{
		Windows: []layout.Window{
			{
				Name: "main",
				Panes: []layout.Pane{
					{Flex: flex(2), Commands: []string{"big"}},
					{Flex: flex(1), Commands: []string{"small"}},
				},
			},
		},
	}

	err := layout.Apply(context.Background(), rec.run, testSock, testSession, cfg)
	require.NoError(t, err)
	require.True(t, rec.hasCall("-p 33"), "2:1 flex ratio must produce 33% split for remaining")
}

func TestApply_RowDirectionUsesHorizontalFlag(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	cfg := &layout.Config{
		Windows: []layout.Window{
			{
				Name:          "main",
				FlexDirection: layout.FlexDirectionRow,
				Panes: []layout.Pane{
					{Commands: []string{"left"}},
					{Commands: []string{"right"}},
				},
			},
		},
	}

	err := layout.Apply(context.Background(), rec.run, testSock, testSession, cfg)
	require.NoError(t, err)
	require.True(t, rec.hasCall("split-window"), "must call split-window")
	require.True(t, rec.hasCall("-h"), "row direction must use -h flag")
}

func TestApply_NestedPanes(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	cfg := &layout.Config{
		Windows: []layout.Window{
			{
				Name: "main",
				Panes: []layout.Pane{
					{Commands: []string{"editor"}},
					{
						Panes: []layout.Pane{
							{Commands: []string{"tests"}},
							{Commands: []string{"git"}},
						},
					},
				},
			},
		},
	}

	err := layout.Apply(context.Background(), rec.run, testSock, testSession, cfg)
	require.NoError(t, err)
	// 1 top-level split + 1 nested split = 2 total
	require.Equal(t, 2, rec.callCount("split-window"))
	require.True(t, rec.hasCall("editor"))
	require.True(t, rec.hasCall("tests"))
	require.True(t, rec.hasCall("git"))
}

func TestApply_FocusPaneSelected(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	cfg := &layout.Config{
		Windows: []layout.Window{
			{
				Name: "main",
				Panes: []layout.Pane{
					{Commands: []string{"a"}},
					{Commands: []string{"b"}, Focus: true},
				},
			},
		},
	}

	err := layout.Apply(context.Background(), rec.run, testSock, testSession, cfg)
	require.NoError(t, err)
	require.True(t, rec.hasCall("select-pane"), "focused pane must trigger select-pane")
}

func TestApply_ZoomAppliedAfterFocus(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	cfg := &layout.Config{
		Windows: []layout.Window{
			{
				Name: "main",
				Panes: []layout.Pane{
					{Commands: []string{"a"}},
					{Commands: []string{"b"}, Focus: true, Zoom: true},
				},
			},
		},
	}

	err := layout.Apply(context.Background(), rec.run, testSock, testSession, cfg)
	require.NoError(t, err)

	// Verify select-pane comes before resize-pane -Z in call order.
	selectIdx, zoomIdx := -1, -1
	rec.mu.Lock()
	for i, c := range rec.calls {
		if strings.Contains(c, "select-pane") {
			selectIdx = i
		}
		if strings.Contains(c, "resize-pane") && strings.Contains(c, "-Z") {
			zoomIdx = i
		}
	}
	rec.mu.Unlock()

	require.Greater(t, selectIdx, -1, "select-pane must be called")
	require.Greater(t, zoomIdx, -1, "resize-pane -Z must be called")
	require.Less(t, selectIdx, zoomIdx, "select-pane must come before resize-pane -Z")
}

func TestApply_NoFocusOrZoomOnSwitchTo(t *testing.T) {
	t.Parallel()

	// Apply does not re-apply focus/zoom — that is the caller's responsibility.
	// This test verifies Apply itself does not call select-pane when no pane has focus.
	rec := &recorder{}
	cfg := &layout.Config{
		Windows: []layout.Window{
			{
				Name:  "main",
				Panes: []layout.Pane{{Commands: []string{"a"}}},
			},
		},
	}

	err := layout.Apply(context.Background(), rec.run, testSock, testSession, cfg)
	require.NoError(t, err)
	require.False(t, rec.hasCall("select-pane"), "select-pane must not be called when no pane has focus")
	require.False(t, rec.hasCall("resize-pane"), "resize-pane must not be called when no pane has zoom")
}

func TestApply_StartupCommandsSentBeforeLayout(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	cfg := &layout.Config{
		Startup: []layout.Command{{Command: "mise install"}},
		Windows: []layout.Window{
			{
				Name:  "main",
				Panes: []layout.Pane{{Commands: []string{"nvim ."}}},
			},
		},
	}

	err := layout.Apply(context.Background(), rec.run, testSock, testSession, cfg)
	require.NoError(t, err)

	// "mise install" must appear before "nvim ." in the recorded calls.
	startupIdx, layoutIdx := -1, -1
	rec.mu.Lock()
	for i, c := range rec.calls {
		if strings.Contains(c, "mise install") {
			startupIdx = i
		}
		if strings.Contains(c, "nvim .") {
			layoutIdx = i
		}
	}
	rec.mu.Unlock()

	require.Greater(t, startupIdx, -1, "startup command must be sent")
	require.Greater(t, layoutIdx, -1, "layout command must be sent")
	require.Less(t, startupIdx, layoutIdx, "startup must precede layout commands")
}

func TestApply_SessionEnvSetBeforePanes(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	cfg := &layout.Config{
		Env: map[string]string{"EDITOR": "nvim"},
		Windows: []layout.Window{
			{Name: "main", Panes: []layout.Pane{{Commands: []string{"a"}}}},
		},
	}

	err := layout.Apply(context.Background(), rec.run, testSock, testSession, cfg)
	require.NoError(t, err)

	envIdx, splitIdx := -1, -1
	rec.mu.Lock()
	for i, c := range rec.calls {
		if strings.Contains(c, "setenv") && strings.Contains(c, "EDITOR") {
			envIdx = i
		}
		if strings.Contains(c, "split-window") {
			splitIdx = i
		}
	}
	rec.mu.Unlock()

	require.Greater(t, envIdx, -1, "setenv must be called")
	// If there's a split, env must come before it.
	if splitIdx >= 0 {
		require.Less(t, envIdx, splitIdx, "setenv must precede split-window")
	}
}
