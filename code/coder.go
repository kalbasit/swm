package code

// Coder defines the coder interface
// code/
// |-- profile1
// |   |-- base
// |   |   |-- src
// |   |       |-- go.import.path
// |   |-- stories
// |   |   |-- STORY-123
// |   |       |-- src
// |   |           |-- go.import.path
// |-- profile2
// |   |-- base
// |   |   |-- src
// |   |       |-- go.import.path
// |   |-- stories
// |   |   |-- STORY-123
// |   |       |-- src
// |   |           |-- go.import.path
type Coder interface {
	// Path returns the absolute path of this coder
	Path() string

	// Profile returns the profile given it's name or an error if no profile with
	// this name was found
	Profile(profile string) (Profile, error)

	// Scan scans the code path
	Scan() error
}

// Profile defines the profile interface
type Profile interface {
	// Coder returns the coder under which this exists
	Coder() Coder

	// Base returns true if this story is the base story
	Base() bool

	// Story returns the story given it's name or an error if no story with this
	// name was found
	Story(name string) (Story, error)
}

// Story defines the story interface
type Story interface {
	// Profile returns the profile under which this story exists
	Profile() Profile

	// GoPath returns the absolute GOPATH of this story.
	GoPath() string

	// Projects returns all the projects that are available for this story as
	// well as all the projects for this profile in the base story (with no
	// duplicates). All projects returned from the base story will be a copy of
	// the base project with the story changed. The caller must call Ensure() on
	// a project to make sure it exists (as a worktree) before using it.
	Projects() []Project
}

// Project defines the project interface
type Project interface {
	// Story returns the story to which this project belongs to
	Story() Story

	// Ensure ensures the project exists as a worktree and is ready to be used.
	// This is a noop for base projects.
	Ensure() error

	// Path returns the absolute path to this project
	Path() string

	// ImportPath returns the path under which this project can be imported in Go
	ImportPath() string
}
