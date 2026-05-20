package layout

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

// Projects is an alias for ScanRepos that lets *Resolver satisfy the
// workspace.ProjectLister interface without a wrapper type.
func (r *Resolver) Projects(ctx context.Context) ([]*pluginv1.ProjectID, error) {
	return r.ScanRepos(ctx)
}

// ScanRepos discovers all git repositories under <code_root>/repositories/.
//
// It treats first-level entries as host directories (github.com, gitlab.com, …) and
// never checks them for .git — a host directory can never be a repo because
// ProjectIDFromPath requires host + at least one segment.  Fan-out begins at each
// host's children: one goroutine per child, each checking for .git before
// descending, stopping as soon as a repo root is found.
func (r *Resolver) ScanRepos(ctx context.Context) ([]*pluginv1.ProjectID, error) {
	start := time.Now()

	defer func() {
		slog.DebugContext(ctx, "ScanRepos complete", "duration", time.Since(start))
	}()

	hostsDir := filepath.Join(r.codeRoot, "repositories")

	hosts, err := os.ReadDir(hostsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading repositories dir: %w", err)
	}

	var (
		mu      sync.Mutex
		results []*pluginv1.ProjectID
		wg      sync.WaitGroup
	)

	for _, h := range hosts {
		if !h.IsDir() {
			continue
		}

		hostPath := filepath.Join(hostsDir, h.Name())

		wg.Go(func() {
			children, rdErr := os.ReadDir(hostPath)
			if rdErr != nil {
				return
			}

			var cwg sync.WaitGroup

			for _, child := range children {
				if !child.IsDir() || child.Name() == ".git" {
					continue
				}

				childPath := filepath.Join(hostPath, child.Name())

				cwg.Go(func() {
					r.scanDir(ctx, childPath, &mu, &results)
				})
			}

			cwg.Wait()
		})
	}

	wg.Wait()

	return results, nil
}

// scanDir checks path for a .git entry; if found it records the ProjectID and
// returns without descending further.  Otherwise it fans out one goroutine per
// child directory (skipping entries named .git to avoid recursing into git
// internals).
func (r *Resolver) scanDir(ctx context.Context, path string, mu *sync.Mutex, results *[]*pluginv1.ProjectID) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
		if id := r.ProjectIDFromPath(path); id != nil {
			mu.Lock()

			*results = append(*results, id)

			mu.Unlock()
		}

		return
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	var wg sync.WaitGroup

	for _, e := range entries {
		if !e.IsDir() || e.Name() == ".git" {
			continue
		}

		childPath := filepath.Join(path, e.Name())

		wg.Go(func() {
			r.scanDir(ctx, childPath, mu, results)
		})
	}

	wg.Wait()
}
