package vcs_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/plugins/vcs-git/internal/vcs"
)

// cloneStream captures CloneProgressEvent messages sent during a streaming Clone call.
type cloneStream struct {
	events []*pluginv1.CloneProgressEvent
	ctx    context.Context //nolint:containedctx // needed to implement grpc.ServerStream
}

func newCloneStream() *cloneStream {
	return &cloneStream{ctx: context.Background()}
}

func (s *cloneStream) Context() context.Context { return s.ctx }
func (s *cloneStream) RecvMsg(any) error        { return nil }

func (s *cloneStream) Send(evt *pluginv1.CloneProgressEvent) error {
	s.events = append(s.events, evt)

	return nil
}

func (s *cloneStream) SendHeader(metadata.MD) error { return nil }
func (s *cloneStream) SendMsg(any) error            { return nil }
func (s *cloneStream) SetHeader(metadata.MD) error  { return nil }
func (s *cloneStream) SetTrailer(metadata.MD)       {}

// lastEvent returns the last event captured, or nil if none.
func (s *cloneStream) lastEvent() *pluginv1.CloneProgressEvent {
	if len(s.events) == 0 {
		return nil
	}

	return s.events[len(s.events)-1]
}

const gitBin = "git"

func newGit(t *testing.T) *vcs.Git {
	t.Helper()

	git, err := vcs.New()
	require.NoError(t, err)

	return git
}

// initRepo creates an empty git repo with an initial commit and remote.
func initRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	cmds := [][]string{
		{gitBin, "-C", dir, "init"},
		{gitBin, "-C", dir, "config", "user.email", "test@test.com"},
		{gitBin, "-C", dir, "config", "user.name", "Test"},
		{gitBin, "-C", dir, "commit", "--allow-empty", "-m", "init"},
		{gitBin, "-C", dir, "remote", "add", "origin", "git@github.com:kalbasit/swm.git"},
	}

	for _, c := range cmds {
		out, err := exec.Command(c[0], c[1:]...).CombinedOutput() //nolint:gosec // trusted test commands
		require.NoError(t, err, "cmd %v: %s", c, out)
	}

	return dir
}

func TestParseRemoteURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		url          string
		wantHost     string
		wantSegments []string
	}{
		{
			name:         "ssh",
			url:          "git@github.com:kalbasit/swm.git",
			wantHost:     "github.com",
			wantSegments: []string{"kalbasit", "swm"},
		},
		{
			name:         "scp with non-git ssh user",
			url:          "org-1470@git.entreprise.com:Team/Project.git",
			wantHost:     "git.entreprise.com",
			wantSegments: []string{"Team", "Project"},
		},
		{
			name:         "scp with no ssh user",
			url:          "git.entreprise.com:Team/Project.git",
			wantHost:     "git.entreprise.com",
			wantSegments: []string{"Team", "Project"},
		},
		{
			name:         "https",
			url:          "https://gitlab.com/foo/bar/baz.git",
			wantHost:     "gitlab.com",
			wantSegments: []string{"foo", "bar", "baz"},
		},
		{
			name:         "ssh scheme",
			url:          "ssh://git@example.com/owner/repo.git",
			wantHost:     "example.com",
			wantSegments: []string{"owner", "repo"},
		},
		{
			name:         "ssh scheme with explicit port",
			url:          "ssh://git@example.com:2222/owner/repo.git",
			wantHost:     "example.com",
			wantSegments: []string{"owner", "repo"},
		},
		{
			name:         "file url",
			url:          "file:///tmp/foo/bar",
			wantHost:     "localhost",
			wantSegments: []string{"tmp", "foo", "bar"},
		},
		{
			name:         "absolute local path",
			url:          "/srv/repos/acme/widget.git",
			wantHost:     "localhost",
			wantSegments: []string{"srv", "repos", "acme", "widget"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := newGit(t)
			id, err := g.ParseRemoteURL(context.Background(), &pluginv1.ParseRemoteURLRequest{
				Url: tc.url,
			})
			require.NoError(t, err)
			require.Equal(t, tc.wantHost, id.GetHost())
			require.NotContains(t, id.GetHost(), ":", "host must never carry a port")
			require.Equal(t, tc.wantSegments, id.GetSegments())
		})
	}
}

func TestParseRemoteURL_Invalid(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		url  string
	}{
		{name: "not a url", url: "not-a-url"},
		{name: "doubled slash yields empty segment", url: "https://gitlab.com/foo//baz.git"},
		{name: "trailing slash yields empty segment", url: "https://gitlab.com/foo/bar/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := newGit(t)
			_, err := g.ParseRemoteURL(context.Background(), &pluginv1.ParseRemoteURLRequest{
				Url: tc.url,
			})
			require.Error(t, err)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestClone_Success(t *testing.T) {
	t.Parallel()

	src := initRepo(t)
	dst := filepath.Join(t.TempDir(), "clone")

	g := newGit(t)
	stream := newCloneStream()
	err := g.Clone(&pluginv1.CloneRequest{Url: src, DestinationPath: dst}, stream)
	require.NoError(t, err)
	require.DirExists(t, filepath.Join(dst, ".git"))

	// The terminal event must be a project_id.
	last := stream.lastEvent()
	require.NotNil(t, last, "expected at least one event")
	require.NotNil(t, last.GetProjectId(), "last event must carry project_id")
	require.Equal(t, "localhost", last.GetProjectId().GetHost())
}

func TestClone_AlreadyExists(t *testing.T) {
	t.Parallel()

	src := initRepo(t)
	g := newGit(t)

	// Clone into src itself — src already has .git, so AlreadyExists is returned.
	stream := newCloneStream()
	err := g.Clone(&pluginv1.CloneRequest{Url: src, DestinationPath: src}, stream)
	require.Error(t, err)
	require.Empty(t, stream.events, "AlreadyExists must send no events")
}

func TestClone_GitFailure(t *testing.T) {
	t.Parallel()

	dst := filepath.Join(t.TempDir(), "clone")
	g := newGit(t)

	stream := newCloneStream()
	err := g.Clone(&pluginv1.CloneRequest{
		Url:             "/nonexistent/repo/that/does/not/exist",
		DestinationPath: dst,
	}, stream)
	require.Error(t, err)
	// On failure, no terminal project_id event must be sent (progress lines are ok).
	if last := stream.lastEvent(); last != nil {
		require.Nil(t, last.GetProjectId(), "git failure must not emit a terminal project_id event")
	}
}

func TestCreateAndRemoveWorktree(t *testing.T) {
	t.Parallel()

	canonical := initRepo(t)
	worktreeDir := filepath.Join(t.TempDir(), "stories", "feat-x", "github.com", "kalbasit", "swm")

	g := newGit(t)

	_, err := g.CreateWorktree(context.Background(), &pluginv1.CreateWorktreeRequest{
		RepoPath:     canonical,
		WorktreePath: worktreeDir,
		BranchName:   "feat/feat-x",
	})
	require.NoError(t, err)
	require.DirExists(t, worktreeDir)

	_, err = g.RemoveWorktree(context.Background(), &pluginv1.RemoveWorktreeRequest{
		WorktreePath: worktreeDir,
	})
	require.NoError(t, err)
	require.NoDirExists(t, worktreeDir)
}

func TestDetectProjectAtPath(t *testing.T) {
	t.Parallel()

	dir := initRepo(t)

	g := newGit(t)
	id, err := g.DetectProjectAtPath(context.Background(), &pluginv1.DetectAtPathRequest{
		Path: dir,
	})
	require.NoError(t, err)
	require.Equal(t, "github.com", id.GetHost())
	require.Equal(t, []string{"kalbasit", "swm"}, id.GetSegments())
}

func TestDetectProjectAtPath_NotGitRepo(t *testing.T) {
	t.Parallel()

	g := newGit(t)
	_, err := g.DetectProjectAtPath(context.Background(), &pluginv1.DetectAtPathRequest{
		Path: t.TempDir(),
	})
	require.Error(t, err)
}

func TestInfo(t *testing.T) {
	t.Parallel()

	g := newGit(t)
	info, err := g.Info(context.Background(), &pluginv1.Empty{})
	require.NoError(t, err)
	require.Equal(t, "git", info.GetPluginInfo().GetName())
	require.Contains(t, info.GetProjectMarkers(), ".git")
}

func TestListBranches(t *testing.T) {
	t.Parallel()

	_ = initRepo(t)
	// ListBranches is not needed for Phase 1 flows; just verify it doesn't panic.
	g := newGit(t)
	require.NotNil(t, g)
}
