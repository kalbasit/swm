package code

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/rs/zerolog/log"
)

const srcDir = "src"

// gitPath is the PATH to the git binary
var gitPath string

type project struct {
	// story returns the parent story
	story *story

	// importPath is the path of the project relative to the GOPATH/src of the profile/workspace
	importPath string
}

func init() {
	var err error
	gitPath, err = exec.LookPath("git")
	if err != nil {
		log.Fatal().Msgf("error looking up the git executable, is it installed? %s", err)
	}
}

func newProject(s *story, importPath string) *project {
	return &project{
		story:      s,
		importPath: importPath,
	}
}

// Story returns the story to which this project belongs to
func (p *project) Story() Story { return p.story }

// Path returns the absolute path of the project
func (p *project) Path() string { return path.Join(p.story.GoPath(), srcDir, p.importPath) }

// Ensure ensures the project exists on disk, by creating a new worktree from
// the base project or noop if the worktree already exists on disk.
func (p *project) Ensure() error {
	if _, err := AppFS.Stat(p.Path()); os.IsNotExist(err) {
		// get the base project
		baseStory := p.story.Profile().Base()
		baseProject, err := baseStory.Project(p.importPath)
		if err != nil {
			return err
		}
		// create a new worktree for this project based on the base project
		// TODO(kalbasit): switch to using [go-git](https://github.com/src-d/go-git)
		cmd := exec.Command(gitPath, "worktree", "add", "-b", p.story.name, p.Path(), "master")
		cmd.Dir = baseProject.Path()
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error creating a new worktree: %s\nOutput:\n%s", err, string(out))
		}
		// add this project to the projects of the story above
		p.story.addProject(p.importPath)
	}

	return nil
}

// ImportPath returns the path under which this project can be imported in Go
func (p *project) ImportPath() string { return p.importPath }
