package story

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

// errStoryNameEmpty is returned when an empty story name is provided.
var errStoryNameEmpty = errors.New("story name must not be empty")

// Store is the interface for story persistence.
type Store interface {
	Create(ctx context.Context, name, branchName string) (*Story, error)
	Get(ctx context.Context, name string) (*Story, error)
	List(ctx context.Context) ([]*Story, error)
	Delete(ctx context.Context, name string) error
	Update(ctx context.Context, s *Story) error
}

// JSONStore implements Store using JSON files in a directory.
type JSONStore struct {
	dir string
}

// NewJSONStore returns a JSONStore backed by the given directory.
func NewJSONStore(dir string) Store {
	return &JSONStore{dir: dir}
}

// Create creates a new story with the given name and branch name.
func (s *JSONStore) Create(_ context.Context, name, branchName string) (*Story, error) {
	if name == "" {
		return nil, errStoryNameEmpty
	}

	if err := s.ensureDir(); err != nil {
		return nil, fmt.Errorf("initializing stories directory: %w", err)
	}

	p := s.path(name)
	if _, err := os.Stat(p); err == nil {
		return nil, fmt.Errorf("%w: %s", ErrStoryExists, name)
	}

	story := &Story{
		Name:       name,
		BranchName: branchName,
		CreatedAt:  time.Now().UTC(),
		Projects:   []Project{},
		Metadata:   map[string]any{},
	}

	if err := s.writeWithLock(p, story); err != nil {
		return nil, err
	}

	return story, nil
}

// Delete removes the story with the given name.
func (s *JSONStore) Delete(_ context.Context, name string) error {
	p := s.path(name)
	if err := os.Remove(p); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrStoryNotFound, name)
		}

		return fmt.Errorf("deleting story file: %w", err)
	}

	_ = os.Remove(p + ".lock") //nolint:errcheck // best-effort lock file cleanup

	return nil
}

// Get returns the story with the given name.
func (s *JSONStore) Get(_ context.Context, name string) (*Story, error) {
	p := s.path(name)

	data, err := os.ReadFile(p) //nolint:gosec // path is constructed from trusted store directory
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrStoryNotFound, name)
		}

		return nil, fmt.Errorf("reading story file: %w", err)
	}

	var story Story
	if err := json.Unmarshal(data, &story); err != nil {
		return nil, fmt.Errorf("parsing story file: %w", err)
	}

	return &story, nil
}

// List returns all stories sorted by name.
func (s *JSONStore) List(ctx context.Context) ([]*Story, error) {
	if err := s.ensureDir(); err != nil {
		return nil, fmt.Errorf("initializing stories directory: %w", err)
	}

	if err := s.ensureDefault(ctx); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("listing stories directory: %w", err)
	}

	var stories []*Story

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		name := strings.TrimSuffix(e.Name(), ".json")

		st, err := s.Get(ctx, name)
		if err != nil {
			return nil, err
		}

		stories = append(stories, st)
	}

	sort.Slice(stories, func(i, j int) bool { return stories[i].Name < stories[j].Name })

	return stories, nil
}

// Update writes the updated story to disk, validating for duplicate projects.
func (s *JSONStore) Update(_ context.Context, story *Story) error {
	p := s.path(story.Name)

	if _, err := os.Stat(p); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrStoryNotFound, story.Name)
	}

	// Validate no duplicate projects.
	seen := make(map[string]bool, len(story.Projects))

	for _, proj := range story.Projects {
		key := proj.Host + "/" + strings.Join(proj.Segments, "/")
		if seen[key] {
			return fmt.Errorf("%w: %s", ErrProjectAlreadyAttached, key)
		}

		seen[key] = true
	}

	// Set AttachedAt for new projects (those with zero time).
	for i := range story.Projects {
		if story.Projects[i].AttachedAt.IsZero() {
			story.Projects[i].AttachedAt = time.Now().UTC()
		}
	}

	return s.writeWithLock(p, story)
}

func (s *JSONStore) ensureDefault(ctx context.Context) error {
	p := s.path("_default")
	if _, err := os.Stat(p); err == nil {
		return nil
	}

	_, err := s.Create(ctx, "_default", "_default")
	if err != nil && !errors.Is(err, ErrStoryExists) {
		return fmt.Errorf("creating default story: %w", err)
	}

	return nil
}

func (s *JSONStore) ensureDir() error {
	return os.MkdirAll(s.dir, 0o700)
}

func (s *JSONStore) path(name string) string {
	return filepath.Join(s.dir, name+".json")
}

func (s *JSONStore) writeWithLock(p string, story *Story) error {
	fl := flock.New(p + ".lock")
	if err := fl.Lock(); err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}

	defer fl.Unlock() //nolint:errcheck // lock release errors are non-actionable

	data, err := json.MarshalIndent(story, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling story: %w", err)
	}

	if err := os.WriteFile(p, data, 0o600); err != nil {
		return fmt.Errorf("writing story file: %w", err)
	}

	return nil
}
