// Package vcs implements the swm VCS capability using the system git binary.
package vcs

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

// buildVersion is set via -ldflags at build time.
var buildVersion = "dev" //nolint:gochecknoglobals // set via ldflags at link time

// sshURLRe matches SSH-style git URLs: git@github.com:owner/repo.git.
var sshURLRe = regexp.MustCompile(`^git@([^:]+):(.+?)(?:\.git)?$`)

// Git implements pluginv1.VCSServer by shelling out to the system git.
type Git struct {
	gitBin string
}

// New returns a Git instance using the system git binary.
func New() (*Git, error) {
	bin, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git binary not found in PATH: %w", err)
	}

	return &Git{gitBin: bin}, nil
}

// Clone clones a repository to the given destination path, streaming progress
// events from git's stderr followed by a terminal project_id event on success.
func (g *Git) Clone(req *pluginv1.CloneRequest, stream pluginv1.VCS_CloneServer) error {
	if _, err := os.Stat(filepath.Join(req.GetDestinationPath(), ".git")); err == nil {
		return status.Errorf(codes.AlreadyExists, "repository already exists at %s", req.GetDestinationPath())
	}

	//nolint:gosec // gitBin from exec.LookPath; args are controlled
	cmd := exec.CommandContext(stream.Context(), g.gitBin, "clone", "--progress", req.GetUrl(), req.GetDestinationPath())

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return status.Errorf(codes.Internal, "git clone: stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return status.Errorf(codes.Internal, "git clone: start: %v", err)
	}

	// Stream each \r- or \n-delimited progress segment from git's stderr.
	var stderrBuf strings.Builder

	scanner := bufio.NewScanner(stderr)
	scanner.Split(splitCR)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimRight(line, "\r\n") == "" {
			continue
		}

		stderrBuf.WriteString(line)

		if sendErr := stream.Send(&pluginv1.CloneProgressEvent{
			Event: &pluginv1.CloneProgressEvent_ProgressLine{ProgressLine: line},
		}); sendErr != nil {
			_ = cmd.Wait() //nolint:errcheck // stream failed; drain process before returning

			return sendErr
		}
	}

	if err := cmd.Wait(); err != nil {
		return status.Errorf(codes.Internal, "git clone %s: %s", req.GetUrl(), stderrBuf.String())
	}

	id, _ := parseURL(req.GetUrl()) //nolint:errcheck // best-effort; nil ProjectId is valid for unrecognized URLs

	return stream.Send(&pluginv1.CloneProgressEvent{
		Event: &pluginv1.CloneProgressEvent_ProjectId{ProjectId: id},
	})
}

// splitCR is a bufio.SplitFunc that splits on \r or \n so git's \r-based
// in-place progress updates are each emitted as separate tokens.
func splitCR(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	for i, b := range data {
		if b == '\r' || b == '\n' {
			return i + 1, data[:i+1], nil
		}
	}

	if atEOF {
		return len(data), data, nil
	}

	return 0, nil, nil
}

// CreateWorktree creates a git worktree for a story.
func (g *Git) CreateWorktree(ctx context.Context, req *pluginv1.CreateWorktreeRequest) (*pluginv1.Empty, error) {
	if err := os.MkdirAll(filepath.Dir(req.GetWorktreePath()), 0o750); err != nil {
		return nil, status.Errorf(codes.Internal, "creating worktree parent: %v", err)
	}

	// Check if branch exists.
	_, branchErr := g.run(ctx, "-C", req.GetRepoPath(), "rev-parse", "--verify", req.GetBranchName())

	var args []string
	if branchErr != nil {
		// Branch doesn't exist — create it.
		args = []string{"-C", req.GetRepoPath(), "worktree", "add", "-b", req.GetBranchName(), req.GetWorktreePath()}
	} else {
		args = []string{"-C", req.GetRepoPath(), "worktree", "add", req.GetWorktreePath(), req.GetBranchName()}
	}

	if _, err := g.run(ctx, args...); err != nil {
		return nil, err
	}

	return &pluginv1.Empty{}, nil
}

// DetectProjectAtPath detects a git project at the given path.
func (g *Git) DetectProjectAtPath(
	ctx context.Context,
	req *pluginv1.DetectAtPathRequest,
) (*pluginv1.ProjectID, error) {
	originURL, err := g.run(ctx, "-C", req.GetPath(), "remote", "get-url", "origin")
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "no git repository at %s", req.GetPath())
	}

	return parseURL(originURL)
}

// Info returns metadata about this VCS plugin.
func (g *Git) Info(_ context.Context, _ *pluginv1.Empty) (*pluginv1.VCSInfo, error) {
	return &pluginv1.VCSInfo{
		PluginInfo: &pluginv1.PluginInfo{
			Name:    "git",
			Version: buildVersion,
		},
		ProjectMarkers: []string{".git"},
	}, nil
}

// ListBranches streams branches for the given project (not required for Phase 1).
func (g *Git) ListBranches(req *pluginv1.ListBranchesRequest, stream pluginv1.VCS_ListBranchesServer) error {
	out, err := g.run(stream.Context(), "-C", req.GetRepoPath(), "branch", "--format=%(refname:short)")
	if err != nil {
		return err
	}

	for name := range strings.SplitSeq(out, "\n") {
		if name == "" {
			continue
		}

		if err := stream.Send(&pluginv1.Branch{Name: name}); err != nil {
			return err
		}
	}

	return nil
}

// ParseRemoteURL parses a remote URL into a ProjectID.
func (g *Git) ParseRemoteURL(_ context.Context, req *pluginv1.ParseRemoteURLRequest) (*pluginv1.ProjectID, error) {
	return parseURL(req.GetUrl())
}

// RemoveWorktree removes a git worktree for a story.
func (g *Git) RemoveWorktree(ctx context.Context, req *pluginv1.RemoveWorktreeRequest) (*pluginv1.Empty, error) {
	mainRepo, err := g.mainRepoPath(ctx, req.GetWorktreePath())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "worktree not found at %s", req.GetWorktreePath())
	}

	if _, err := g.run(ctx, "-C", mainRepo, "worktree", "remove", "--force", req.GetWorktreePath()); err != nil {
		return nil, status.Errorf(codes.Internal, "removing worktree at %s: %v", req.GetWorktreePath(), err)
	}

	_, _ = g.run(ctx, "-C", mainRepo, "worktree", "prune") //nolint:errcheck // best-effort worktree prune

	return &pluginv1.Empty{}, nil
}

// mainRepoPath resolves the main repository root from any path within a worktree.
func (g *Git) mainRepoPath(ctx context.Context, worktreePath string) (string, error) {
	gitCommonDir, err := g.run(ctx, "-C", worktreePath, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}

	if !filepath.IsAbs(gitCommonDir) {
		gitCommonDir = filepath.Join(worktreePath, gitCommonDir)
	}

	return filepath.Dir(gitCommonDir), nil
}

func (g *Git) run(ctx context.Context, args ...string) (string, error) {
	var stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, g.gitBin, args...) //nolint:gosec // gitBin is from exec.LookPath, args are controlled
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return "", status.Errorf(codes.Internal, "git %s: %s", strings.Join(args, " "), stderr.String())
	}

	return strings.TrimSpace(string(out)), nil
}

func parseURL(raw string) (*pluginv1.ProjectID, error) {
	// SSH format: git@github.com:owner/repo.git
	if m := sshURLRe.FindStringSubmatch(raw); m != nil {
		host := m[1]
		segments := strings.Split(m[2], "/")

		return &pluginv1.ProjectID{Host: host, Segments: segments}, nil
	}

	// Absolute local path (no scheme) — treat as localhost.
	if after, ok := strings.CutPrefix(raw, "/"); ok {
		path := strings.TrimSuffix(after, ".git")

		return &pluginv1.ProjectID{Host: "localhost", Segments: strings.Split(path, "/")}, nil
	}

	// All other formats (HTTPS, file://, git+ssh, etc.)
	u, err := url.Parse(raw)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot parse remote URL %q", raw)
	}

	// file:// — local repository; host is "localhost".
	if u.Scheme == "file" {
		path := strings.TrimPrefix(u.Path, "/")
		path = strings.TrimSuffix(path, ".git")

		if path == "" {
			return nil, status.Errorf(codes.InvalidArgument, "file URL %q has no path", raw)
		}

		return &pluginv1.ProjectID{Host: "localhost", Segments: strings.Split(path, "/")}, nil
	}

	if u.Host == "" {
		return nil, status.Errorf(codes.InvalidArgument, "cannot parse remote URL %q", raw)
	}

	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")

	if path == "" {
		return nil, status.Errorf(codes.InvalidArgument, "URL %q has no path", raw)
	}

	segments := strings.Split(path, "/")

	return &pluginv1.ProjectID{Host: u.Host, Segments: segments}, nil
}
