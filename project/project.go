package project

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/google/go-github/github"
	"github.com/kalbasit/swm/ifaces"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const srcDir = "src"

var (
	// ErrNoActiveStory is returned if there's no active story
	ErrNoActiveStory = errors.New("no story is active")

	// gitPath is the PATH to the git binary
	gitPath string
)

type project struct {
	// code links back to parent coder
	code ifaces.Code

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

func New(c ifaces.Code, importPath string) ifaces.Project {
	return &project{
		code:       c,
		importPath: importPath,
	}
}

// Path returns the absolute path to the repository or the story for this project.
func (p *project) Path(s ifaces.Story) string {
	if rp := p.storyPath(s); rp != "" {
		return rp
	}

	return path.Join(p.code.RepositoriesDir(), p.importPath)
}

func (p *project) String() string { return p.importPath }

func (p *project) Code() ifaces.Code { return p.code }

// ListPullRequests returns the list of pull requests.
func (p *project) ListPullRequests(ghc *github.Client) ([]*github.PullRequest, error) {
	prs, _, err := ghc.PullRequests.List(context.Background(), p.owner(), p.repo(), nil)
	return prs, err
}

func (p *project) storyPath(s ifaces.Story) string {
	if s == nil {
		return ""
	}
	return path.Join(p.code.StoriesDir(), s.GetName(), p.importPath)
}

func (p *project) repositoryPath() string {
	return path.Join(p.code.RepositoriesDir(), p.importPath)
}

// CreateStory creates the story path for this project.
func (p *project) CreateStory(s ifaces.Story) error {
	wp := p.storyPath(s)
	sbn := s.GetBranchName()

	if _, err := os.Stat(wp); !os.IsNotExist(err) {
		if err != nil {
			log.Error().
				Str("import-path", p.importPath).
				Str("story-path", wp).
				Msg("error stat the path")
			return err
		}

		log.Debug().
			Str("import-path", p.importPath).
			Str("story-path", wp).
			Msg("the story already exists")
		return nil
	}

	// run the pre-hooks
	if err := p.runPreHooks(s); err != nil {
		log.Error().
			Str("import-path", p.importPath).
			Str("story-path", wp).
			Msg("error running the pre-hooks")
		return err
	}
	// create a new story for this project based on the base project
	// TODO(kalbasit): switch to using [go-git](https://github.com/src-d/go-git)
	cmd := exec.Command(gitPath, "worktree", "add", "-B", sbn, wp)
	cmd.Dir = p.repositoryPath()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error creating a new story: %s\nOutput:\n%s", err, string(out))
	}
	// run the post-hooks
	if err := p.runPostHooks(s); err != nil {
		log.Error().
			Str("import-path", p.importPath).
			Str("story0path", wp).
			Msg("error running the post-hooks")
		return err
	}

	log.Debug().
		Str("import-path", p.importPath).
		Str("story-path", wp).
		Msg("story created successfully")

	return nil
}

func (p *project) owner() string {
	parts := strings.Split(p.importPath, "/")
	if len(parts) != 3 {
		return ""
	}
	return parts[1]
}

func (p *project) repo() string {
	parts := strings.Split(p.importPath, "/")
	if len(parts) != 3 {
		return ""
	}
	return parts[2]
}

// runPreHooks iterates over the executable files in
// ~/.config/swm/hooks/coder/pre-hook and runs each with the following
// arguments:
// - The name of the story
// - The path to the story of this project
// - The path to the repository of this project
func (p *project) runPreHooks(s ifaces.Story) error {
	// get the hooks directory
	preHooksDir := path.Join(p.code.HookPath(), "pre-hook")
	// first get the list of the hooks
	hooks, err := ioutil.ReadDir(preHooksDir)
	if err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Str("pre-hook-dir", preHooksDir).Msg("error reading the directory")
		return err
	} else if os.IsNotExist(err) {
		return nil
	}
	// compute the paths needed for all hooks
	rp := p.repositoryPath()
	wp := p.storyPath(s)
	// iterate over the list of hooks and run it
	for _, hook := range hooks {
		// compute the absolute path of the hook
		hookPath := path.Join(preHooksDir, hook.Name())
		log.Debug().
			Str("hook_path", hookPath).
			Bool("executable", hook.Mode().Perm()&0111 != 0).
			Msg("found a pre-hook")
		// is this a file and is executable by the current user?
		if !hook.IsDir() && hook.Mode().Perm()&0111 != 0 {
			cmd := exec.Command(hookPath, s.GetName(), wp, rp)
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("error running the pre-hook: %s\nOutput:\n%s", err, string(out))
			}
		}
	}

	return nil
}

// runPostHooks iterates over the executable files in
// ~/.config/swm/hooks/coder/post-hook and runs each with the following
// arguments:
// - The name of the story
// - The path to the story of this project
// - The path to the repository of this project
func (p *project) runPostHooks(s ifaces.Story) error {
	// compute the absolute path of the hook
	postHooksDir := path.Join(p.code.HookPath(), "post-hook")
	// first get the list of the hooks
	hooks, err := ioutil.ReadDir(postHooksDir)
	if err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Str("post-hook-dir", postHooksDir).Msgf("error reading the directory")
		return err
	} else if os.IsNotExist(err) {
		return nil
	}
	// compute the paths needed for all hooks
	rp := p.repositoryPath()
	wp := p.storyPath(s)
	// iterate over the list of hooks and run it
	for _, hook := range hooks {
		hookPath := path.Join(postHooksDir, hook.Name())
		log.Debug().
			Str("hook_path", hookPath).
			Bool("executable", hook.Mode().Perm()&0111 != 0).
			Msg("found a post-hook")
		// is this a file and is executable by the current user?
		if !hook.IsDir() && hook.Mode().Perm()&0111 != 0 {
			cmd := exec.Command(hookPath, s.GetName(), wp, rp)
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("error running the post-hook: %s\nOutput:\n%s", err, string(out))
			}
		}
	}

	return nil
}
