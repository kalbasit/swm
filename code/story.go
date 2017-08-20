package code

import (
	"os"
	"os/exec"
	"path"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

// baseStoryName represents the name of the base story
const baseStoryName = "base"

// story implements the Story interface
type story struct {
	// profile returns the parent profile
	profile *profile

	// name is the name of the story
	name string

	// projects is a list of projects
	projects unsafe.Pointer // type *map[string]*project
}

func newStory(p *profile, name string) *story {
	projects := make(map[string]*project)
	return &story{
		name:     name,
		profile:  p,
		projects: unsafe.Pointer(&projects),
	}
}

// Profile returns the profile under which this story exists
func (s *story) Profile() Profile { return s.profile }

// Name returns the name of the story
func (s *story) Name() string { return s.name }

// Base returns true if this story is the base story
func (s *story) Base() bool { return s.name == baseStoryName }

// GoPath returns the absolute GOPATH of this story.
func (s *story) GoPath() string {
	if s.name == baseStoryName {
		return path.Join(s.profile.code.Path(), s.profile.name, s.name)
	}

	return path.Join(s.profile.code.Path(), s.profile.name, "stories", s.name)
}

// Projects returns all the projects that are available for this story as
// well as all the projects for this profile in the base story (with no
// duplicates). All projects returned from the base story will be a copy of
// the base project with the story changed. The caller must call Ensure() on
// a project to make sure it exists (as a worktree) before using it.
func (s *story) Projects() []Project {
	var res []Project
	for _, prj := range s.getProjects() {
		res = append(res, prj)
	}
	return res
}

// Project returns the project given the importPath. If the project does not
// exist for this story but does exist in the Base story, it will be copied and
// story changed. The caller must call Ensure() on the project to make sure it
// exists (as a worktree) before using it.
func (s *story) Project(importPath string) (Project, error) {
	// get the project for the story
	prj, ok := s.getProjects()[importPath]
	if !ok {
		basePrj, ok := s.profile.Base().(*story).getProjects()[importPath]
		if !ok {
			return nil, ErrProjectNotFound
		}
		prj = &project{}
		*prj = *basePrj
		prj.story = s
	}

	return prj, nil
}

// AddProject clones url as the new project. Will automatically compute the
// import path from the given URL.
func (s *story) AddProject(url string) error {
	// compute the import path of this URL
	var importPath string
	{
		r := parseRemoteURL(url)
		importPath = r.hostname + "/" + r.path
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
	if prj, err := s.Project(importPath); err == nil {
		// the existance of the project might be due to the project existing in the
		// Base story, so we must really check the filename
		if _, err := AppFS.Stat(prj.Path()); err == nil {
			log.Debug().
				Str("import-path", importPath).
				Str("path", prj.Path()).
				Msg(ErrProjectAlreadyExists.Error())
			return ErrProjectAlreadyExists
		}
	}
	// run a git clone on the absolute path of the project
	cmd := exec.Command(gitPath, "clone", url, s.projectPath(importPath))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	// add this project to the projects
	p := s.addProject(importPath)
	log.Info().
		Str("import-path", importPath).
		Str("path", p.Path()).
		Msg("project successfully cloned")

	return nil
}

// getProjects return the map of projects
func (s *story) getProjects() map[string]*project {
	return *(*map[string]*project)(atomic.LoadPointer(&s.projects))
}

// addProject add the project by the import path
func (s *story) addProject(importPath string) *project {
	if p, ok := s.getProjects()[importPath]; ok {
		return p
	}
	p := newProject(s, importPath)
	for {
		projectsPtr := atomic.LoadPointer(&s.projects)
		projects := *(*map[string]*project)(projectsPtr)
		projects[importPath] = p
		if atomic.CompareAndSwapPointer(&s.projects, projectsPtr, unsafe.Pointer(&projects)) {
			return p
		}
	}
}

// scan scans the entire story to build projects
func (s *story) scan() {
	// initialize the variables
	var wg sync.WaitGroup
	out := make(chan string, 1000)
	// start the workers
	wg.Add(1)
	go s.scanWorker(&wg, out, "")
	// start the reducer
	reducerQuit := make(chan struct{})
	go s.scanReducer(out, reducerQuit)
	// wait for the workers to return
	wg.Wait()
	// ask the reducer to die
	close(out)
	<-reducerQuit
}

func (s *story) scanReducer(out chan string, quit chan struct{}) {
	// iterate over the channel
	for {
		select {
		case importPath, ok := <-out:
			if !ok {
				close(quit)
				return
			}
			s.addProject(importPath)
		}
	}
}

func (s *story) scanWorker(wg *sync.WaitGroup, out chan string, ipath string) {
	defer wg.Done()

	// do we have a .git folder here?
	if _, err := AppFS.Stat(path.Join(s.projectPath(ipath), ".git")); err == nil {
		log.Debug().Str("path", s.projectPath(ipath)).Msg("found a Git repository")
		// return this import path
		out <- ipath

		return
	}

	// scan the folder
	entries, err := afero.ReadDir(AppFS, s.projectPath(ipath))
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Fatal().Str("path", s.projectPath(ipath)).Msgf("error reading the directory: %s", err)
	}
	for _, entry := range entries {
		// scan the entry if it's a directory
		if entry.IsDir() {
			wg.Add(1)
			go s.scanWorker(wg, out, path.Join(ipath, entry.Name()))
		}
	}
}

func (s *story) projectPath(importPath string) string {
	return path.Join(s.GoPath(), srcDir, importPath)
}
