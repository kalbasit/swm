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

type Project struct {
	// ImportPath is the path of the project relative to the GOPATH/src of the profile/workspace
	ImportPath string

	// CodePath is the path of Code.Path
	CodePath string

	// ProfileName is the name of the profile for this workspace
	ProfileName string

	// StoryName is the name
	StoryName string
}

// Base returns true if this project is under the base workspace
func (p *Project) Base() bool { return p.StoryName == BaseStory }

// Path returns the absolute path of the project
func (p *Project) Path() string {
	return path.Join(p.CodePath, p.ProfileName, p.StoryName, srcDir, p.ImportPath)
}

// SessionName returns the session name to be used for TMUX. The format is:
// profile@workspace=ImportPath the ImportPath does not include dots or columns
func (p *Project) SessionName() string {
	return fmt.Sprintf("%s@%s=%s", p.ProfileName, p.StoryName, p.tmuxSafeName())
}

func (p *Project) tmuxSafeName() string {
	return strings.Replace(strings.Replace(p.ImportPath, ".", dotChar, -1), ":", colonChar, -1)
}
