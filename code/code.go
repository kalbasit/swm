package code

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/google/go-github/github"
	"github.com/kalbasit/swm/ifaces"
	"github.com/kalbasit/swm/project"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	// ErrCodePathEmpty is returned if Code.Path is empty or invalid
	ErrCodePathEmpty = errors.New("code path is empty or does not exist")

	// ErrProjectNotFound is returned if the project is not found
	ErrProjectNotFound = errors.New("project not found")

	// ErrInvalidURL is returned by AddProject if the URL given is not valid
	ErrInvalidURL = errors.New("invalid URL given")

	// ErrProjectAlreadyExists is returned if the project already exists
	ErrProjectAlreadyExists = errors.New("project already exists")

	// ErrCoderNotScanned is returned if San() was never called
	ErrCoderNotScanned = errors.New("code was not scanned")

	// ErrPathIsInvalid is returned if the given absolute path is invalid (it
	// does not make up a full coder path + profile + story + project).
	ErrPathIsInvalid = errors.New("path is invalid")

	// ErrDotGitMalformed is returned if .git is malformed
	ErrDotGitMalformed = errors.New(".git is malformed")

	gitWorktreeRootRegex = regexp.MustCompile(`^gitdir: (.*)/\.git/worktrees/.*$`)

	// gitPath is the PATH to the git binary
	gitPath string
)

func init() {
	var err error
	gitPath, err = exec.LookPath("git")
	if err != nil {
		log.Fatal().Msgf("error looking up the git executable, is it installed? %s", err)
	}
}

// code implements the coder interface
type code struct {
	// ghClient represents a GitHub client
	ghClient *github.Client

	// path is the base path of this profile
	path string

	// the name of the story
	story_name string

	// excludePattern is a list of patterns to ignore
	excludePattern *regexp.Regexp

	mu       sync.RWMutex
	projects map[string]ifaces.Project
}

// New returns a new empty Code, caller must call Load to load from cache or
// scan the code directory
func New(ghc *github.Client, p, sn string, ignore *regexp.Regexp) ifaces.Code {
	return &code{
		ghClient:       ghc,
		excludePattern: ignore,
		path:           path.Clean(p),
		projects:       make(map[string]ifaces.Project),
		story_name:     sn,
	}
}

// Path returns the absolute path of this coder
func (c *code) Path() string { return c.path }

// StoryName returns the name of the story if any, empty string otherwise.
func (c *code) StoryName() string { return c.story_name }

// GithubClient represents the client for Github API.
func (c *code) GithubClient() *github.Client { return c.ghClient }

// Scan loads the code from the cache (if it exists), otherwise it will
// initiate a full scan and save it in cache.
func (c *code) Scan() error {
	// validate the Code, we cannot load an invalid Code
	if err := c.validate(); err != nil {
		return err
	}

	c.scan()

	return nil
}

// Projects returns all the projects that are available for this story as
// well as all the projects for this profile in the base story (with no
// duplicates). All projects returned from the base story will be a copy of
// the base project with the story changed. The caller must call Ensure() on
// a project to make sure it exists (as a story) before using it.
func (c *code) Projects() []ifaces.Project {
	var res []ifaces.Project
	c.mu.RLock()
	for _, prj := range c.projects {
		res = append(res, prj)
	}
	c.mu.RUnlock()

	return res
}

func (c *code) Clone(url string) error {
	// compute the import path of this URL
	var importPath string
	{
		r := parseRemoteURL(url)
		if r.hostname != "" {
			importPath = r.hostname
		}
		importPath = path.Join(importPath, r.path)
		if importPath == "" {
			log.Error().
				Str("import-path", importPath).
				Interface("remote-url", r).
				Msg("parsing failed")
			return ErrInvalidURL
		}
		log.Debug().
			Str("import-path", importPath).
			Interface("remote-url", r).
			Msg("parsing succeded")
	}
	// validate we don't have it already
	if prj, err := c.getProject(importPath); err == nil {
		log.Debug().
			Str("import-path", importPath).
			Str("path", prj.RepositoryPath()).
			Msg(ErrProjectAlreadyExists.Error())
		return ErrProjectAlreadyExists
	}
	// clone the project
	prj := project.New(c, importPath)
	// run a git clone on the absolute path of the project
	cmd := exec.Command(gitPath, "clone", url, prj.RepositoryPath())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	log.Info().
		Str("import-path", prj.String()).
		Str("repository_path", prj.RepositoryPath()).
		Msg("project successfully cloned")
	// add it to the map of projects
	c.addProject(prj)

	return nil
}

// GetProject returns a project identified by it's relative path to the repositories directory.
func (c *code) GetProject(importPath string) (ifaces.Project, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	prj, ok := c.projects[importPath]
	if !ok {
		return nil, ErrProjectNotFound
	}

	return prj, nil
}

// ProjectByAbsolutePath returns the project corresponding to the absolute
// path.
func (c *code) ProjectByAbsolutePath(p string) (ifaces.Project, error) {
	dotGit := path.Join(p, ".git")
	gitInfo, err := os.Stat(dotGit)
	if err != nil {
		return nil, errors.Wrap(err, "error stat the .git directory")
	}

	pp := p
	if !gitInfo.IsDir() {
		gitC, err := ioutil.ReadFile(dotGit)
		if err != nil {
			return nil, errors.Wrap(err, "error reading the .git file")
		}

		sm := gitWorktreeRootRegex.FindSubmatch(bytes.Trim(gitC, "\n"))
		if len(sm) != 2 {
			return nil, ErrDotGitMalformed
		}

		pp = string(sm[1])
	}

	// clean the path
	pp = path.Clean(pp)
	// trim the coder path from the path we are looking for (along with the pathSeparator)
	pp = strings.TrimPrefix(pp, c.RepositoriesDir()+string(os.PathSeparator))

	return c.GetProject(pp)
}

func (c *code) RepositoriesDir() string { return path.Join(c.path, "repositories") }

func (c *code) StoriesDir() string { return path.Join(c.path, "stories") }

// scan scans the entire code directory to build the workspaces
func (c *code) scan() {
	// initialize the variables
	var wg sync.WaitGroup
	out := make(chan string, 1000)
	// start the workers
	wg.Add(1)
	go c.scanWorker(&wg, out, "")
	// start the reducer
	reducerQuit := make(chan struct{})
	go c.scanReducer(out, reducerQuit)
	// wait for the workers to return
	wg.Wait()
	// ask the reducer to die
	close(out)
	<-reducerQuit
}

func (c *code) projectPath(importPath string) string {
	return path.Join(c.RepositoriesDir(), importPath)
}

func (c *code) scanWorker(wg *sync.WaitGroup, out chan string, ipath string) {
	defer wg.Done()

	// do we have a .git folder here?
	if _, err := os.Stat(path.Join(c.projectPath(ipath), ".git")); err == nil {
		log.Debug().Str("path", c.projectPath(ipath)).Msg("found a Git repository")
		// return this import path
		out <- ipath

		return
	}

	// scan the folder
	entries, err := ioutil.ReadDir(c.projectPath(ipath))
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Fatal().Str("path", c.projectPath(ipath)).Msgf("error reading the directory: %s", err)
	}
	for _, entry := range entries {
		// scan the entry if it's a directory
		if entry.IsDir() {
			wg.Add(1)
			go c.scanWorker(wg, out, path.Join(ipath, entry.Name()))
		}
	}
}

func (c *code) scanReducer(out chan string, quit chan struct{}) {
	// iterate over the channel
	for {
		select {
		case importPath, ok := <-out:
			if !ok {
				close(quit)
				return
			}
			c.addProject(importPath)
		}
	}
}

// addProject add the project by the import path. The project is not checked if
// it exists.
func (c *code) addProject(p interface{}) {
	var prj ifaces.Project
	switch pt := p.(type) {
	case ifaces.Project:
		prj = pt
	case string:
		prj = project.New(c, pt)
	default:
		panic(fmt.Sprintf("%t is not implemented", pt))
	}

	// noting to do if the project already exists
	c.mu.RLock()
	_, ok := c.projects[prj.String()]
	c.mu.RUnlock()
	if ok {
		return
	}

	// otherwise add it to the map
	c.mu.Lock()
	c.projects[prj.String()] = prj
	c.mu.Unlock()
}

// getProject return the project for importPath from the map
func (c *code) getProject(importPath string) (ifaces.Project, error) {
	// get the project out of the map
	c.mu.RLock()
	prj, ok := c.projects[importPath]
	c.mu.RUnlock()

	if !ok {
		return nil, ErrProjectNotFound
	}

	return prj, nil
}

func (c *code) validate() error {
	if c.path == "" {
		return ErrCodePathEmpty
	}
	if _, err := os.Stat(c.path); err != nil {
		return ErrCodePathEmpty
	}

	return nil
}

// HookPath returns the absolute path to the hooks directory.
func (c *code) HookPath() string {
	return path.Join(os.Getenv("HOME"), ".config", "swm", "hooks", "coder")
}
