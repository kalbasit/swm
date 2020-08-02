package ifaces

import "github.com/google/go-github/github"

// Code defines the code interface
// code/
// |-- repositories
// |   |   |-- go.import.path
// |-- stories
// |   |-- STORY-123
// |   |   |-- go.import.path
type Code interface {
	// CreateStory creates a story
	CreateStory() error

	// SetStoryName sets the story name
	SetStoryName(string)

	// SetStoryBranchName sets the story branch name
	SetStoryBranchName(string)

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

	// StoryName returns the name of the story if any, empty string otherwise.
	StoryName() string

	// StoryBranchName returns the name of the branch of this story if any, empty string otherwise.
	StoryBranchName() string

	// RepositoriesDir returns the path to the repositories directory.
	RepositoriesDir() string

	// StoriesDir returns the path to the stories directory.
	StoriesDir() string

	// GithubClient represents the client for Github API.
	GithubClient() *github.Client

	// SetGithubClient sets the GitHub client in the code
	SetGithubClient(*github.Client)

	// HookPath returns the absolute path to the hooks directory.
	HookPath() string
}

// Project defines the project interface
type Project interface {
	// Ensure ensures the project exists on disk, by creating a new story from
	// the repository, or noop if the story already exists on disk.
	Ensure() error

	// StoryPath returns the absolute path to the story for this project. It
	// returns an error if there is no active story.
	StoryPath() (string, error)

	// StoryPath returns the absolute path to the repository for this project.
	RepositoryPath() string

	// Path returns the path of the story, if it's a story. Otherwise, the
	// path of the repository is returned.
	Path() string

	// String returns the name of the project as found in the filesystem
	String() string

	// ListPullRequests returns the list of pull requests
	ListPullRequests() ([]*github.PullRequest, error)

	// Code returns the code this project is attached to
	Code() Code
}
