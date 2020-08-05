package ifaces

import (
	"time"

	"github.com/google/go-github/github"
)

// Code defines the code interface
// code/
// |-- repositories
// |   |   |-- go.import.path
// |-- stories
// |   |-- STORY-123
// |   |   |-- go.import.path
type Code interface {
	// Path returns the absolute path of this coder
	Path() string

	// Projects returns the projects in this coder
	Projects() []Project

	// GetProjectByRelativePath returns a project identified by it's relative path to the repositories directory.
	GetProjectByRelativePath(string) (Project, error)

	// Clone clones url as the new project. Will automatically compute the import
	// path from the given URL.
	Clone(url string) error

	// GetProjectByAbsolutePath returns the project corresponding to the absolute
	// path.
	GetProjectByAbsolutePath(absolutePath string) (Project, error)

	// Scan scans the code path.
	Scan() error

	// RepositoriesDir returns the absolute path to the repositories directory.
	RepositoriesDir() string

	// StoriesDir returns the path to the stories directory.
	StoriesDir() string

	// HookPath returns the absolute path to the hooks directory.
	HookPath() string
}

// Project defines the project interface
type Project interface {
	// CreateStory creates the story path for this project.
	CreateStory(s Story) error

	// Path returns the absolute path to the repository or the story for this project.
	Path(s Story) string

	// String returns the name of the project as found in the filesystem
	String() string

	// ListPullRequests returns the list of pull requests
	ListPullRequests(*github.Client) ([]*github.PullRequest, error)

	// Code returns the code this project is attached to
	Code() Code
}

// Story defines the story interface
type Story interface {
	// SetName sets the name of the story.
	SetName(string)

	// SetBranchName sets the name of the branch that will be used to create
	// stories for projects.
	SetBranchName(string)

	// GetName returns the name of the story
	GetName() string

	// GetBranchName returns the name of the branch of this story
	GetBranchName() string

	// GetCreatedAt returns the timestamp when this story was created
	GetCreatedAt() time.Time

	// Save saves the story in the data directory.
	Save() error

	// Remove saves the story in the data directory.
	Remove() error
}
