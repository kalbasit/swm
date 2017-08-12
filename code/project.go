package code

import (
	"errors"
	"path"
)

const srcDir = "src"

type project struct {
	// story returns the parent story
	story *story

	// importPath is the path of the project relative to the GOPATH/src of the profile/workspace
	importPath string
}

// Story returns the story to which this project belongs to
func (p *project) Story() Story { return p.story }

// Path returns the absolute path of the project
func (p *project) Path() string { return path.Join(p.story.GoPath(), srcDir, p.importPath) }

// Ensure ensures the project exists on disk, by creating a new worktree from
// the base project or noop if the worktree already exists on disk.
func (p *project) Ensure() error { return errors.New("no implemented yet") }

// ImportPath returns the path under which this project can be imported in Go
func (p *project) ImportPath() string { return p.importPath }
