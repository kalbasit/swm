// Package hostsvc implements the Host gRPC service that plugins call back into.
package hostsvc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/core/story"
)

// Server implements pluginv1.HostServer and manages its own gRPC listener.
type Server struct {
	pluginv1.UnimplementedHostServer

	cfg      *config.Config
	resolver *layout.Resolver
	store    story.Store

	grpcSrv    *grpc.Server
	socketPath string
}

// NewServer starts a Host gRPC server on a temporary Unix socket.
func NewServer(cfg *config.Config, resolver *layout.Resolver, store story.Store) (*Server, error) {
	sock, err := os.CreateTemp("", "swm-hostsvc-*.sock")
	if err != nil {
		return nil, fmt.Errorf("creating socket file: %w", err)
	}

	socketPath := sock.Name()
	sock.Close()          //nolint:errcheck,gosec // closing temp file before replacing with socket
	os.Remove(socketPath) //nolint:errcheck,gosec // remove temp file so Unix listener can create the socket

	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listening on unix socket: %w", err)
	}

	srv := &Server{
		cfg:        cfg,
		resolver:   resolver,
		store:      store,
		grpcSrv:    grpc.NewServer(),
		socketPath: "unix://" + socketPath,
	}

	pluginv1.RegisterHostServer(srv.grpcSrv, srv)

	go func() {
		if err := srv.grpcSrv.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			slog.Error("host gRPC server error", "err", err)
		}
	}()

	return srv, nil
}

// GetCodeRoot returns the configured code root directory.
func (s *Server) GetCodeRoot(_ context.Context, _ *pluginv1.Empty) (*pluginv1.PathResponse, error) {
	return &pluginv1.PathResponse{Path: s.cfg.CodeRoot}, nil
}

// GetConfig returns TOML bytes scoped to the requesting plugin's config section.
func (s *Server) GetConfig(_ context.Context, req *pluginv1.GetConfigRequest) (*pluginv1.Config, error) {
	pluginCfg, ok := s.cfg.Plugins.Config[req.GetPluginName()]
	if !ok {
		return &pluginv1.Config{}, nil
	}

	data, err := toml.Marshal(pluginCfg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshaling plugin config: %v", err)
	}

	return &pluginv1.Config{Toml: data}, nil
}

// GetCurrentStory returns the story for the current $SWM_STORY env var.
func (s *Server) GetCurrentStory(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.Story, error) {
	name := os.Getenv("SWM_STORY")
	if name == "" {
		name = s.cfg.DefaultStory
	}

	st, err := s.store.Get(ctx, name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "story %q not found: %v", name, err)
	}

	return storyToProto(st), nil
}

// ListProjects walks repositories/ and streams project messages for directories
// containing VCS markers.
func (s *Server) ListProjects(
	_ *pluginv1.ListProjectsRequest,
	stream grpc.ServerStreamingServer[pluginv1.Project],
) error {
	reposDir := filepath.Join(s.cfg.CodeRoot, "repositories")

	return filepath.WalkDir(reposDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}

			return err
		}

		if !d.IsDir() {
			return nil
		}

		if d.Name() == ".git" {
			// Parent of .git is a project directory.
			projectDir := filepath.Dir(path)
			id := s.resolver.ProjectIDFromPath(projectDir)

			if id == nil {
				return nil
			}

			if err := stream.Send(&pluginv1.Project{
				Host:     id.Host,
				Segments: id.Segments,
			}); err != nil {
				return err
			}

			return filepath.SkipDir
		}

		return nil
	})
}

// Log writes a log message to the host's structured logger.
func (s *Server) Log(ctx context.Context, req *pluginv1.LogRequest) (*pluginv1.Empty, error) {
	level := slog.LevelInfo

	switch strings.ToLower(req.GetLevel()) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	slog.Log(ctx, level, req.GetMessage(), "fields", req.GetFields())

	return &pluginv1.Empty{}, nil
}

// SocketPath returns the gRPC dial address for this server.
func (s *Server) SocketPath() string {
	return s.socketPath
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop() {
	s.grpcSrv.GracefulStop()
}

func storyToProto(s *story.Story) *pluginv1.Story {
	projects := make([]*pluginv1.Project, len(s.Projects))

	for i, p := range s.Projects {
		projects[i] = &pluginv1.Project{
			Host:     p.Host,
			Segments: p.Segments,
			Vcs:      p.VCS,
		}
	}

	return &pluginv1.Story{
		Name:       s.Name,
		BranchName: s.BranchName,
		Projects:   projects,
	}
}
