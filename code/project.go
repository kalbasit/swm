package code

import (
	"fmt"
	"path"
	"strings"
)

const (
	srcDir    = "src"
	dotChar   = "\u2022"
	colonChar = "\uFF1A"
)

type project struct {
	// story returns the parent story
	story *story

	// importPath is the path of the project relative to the GOPATH/src of the profile/workspace
	importPath string
}

// Base returns true if this project is under the base workspace
func (p *project) Base() bool { return p.story.Base() }

// Path returns the absolute path of the project
func (p *project) Path() string {
	return path.Join(p.story.GoPath(), srcDir, p.importPath)
}

// SessionName returns the session name to be used for TMUX. The format is:
// profile@workspace=ImportPath the ImportPath does not include dots or columns
func (p *project) SessionName() string {
	return fmt.Sprintf("%s@%s=%s", p.story.profile.name, p.story.name, p.tmuxSafeName())
}

func (p *project) tmuxSafeName() string {
	return strings.Replace(strings.Replace(p.importPath, ".", dotChar, -1), ":", colonChar, -1)
}
