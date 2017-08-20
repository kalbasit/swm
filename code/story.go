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
				Str("importPath", importPath).
				Interface("remoteURL", r).
				Msg("parsing failed")
			return ErrInvalidURL
		}
		log.Debug().
			Str("importPath", importPath).
			Interface("remoteURL", r).
			Msg("parsing succeded")
	}
	// run a git clone on the absolute path of the project
	if err := exec.Command(gitPath, "clone", url, s.projectPath(importPath)).Run(); err != nil {
		return err
	}
	// add this project to the projects
	projects := s.getProjects()
	projects[importPath] = newProject(s, importPath)
	s.setProjects(projects)

	log.Info().Str("importPath", importPath).Msgf("cloned at %s", projects[importPath].Path())

	return nil
}

// getProjects return the map of projects
func (s *story) getProjects() map[string]*project {
	return *(*map[string]*project)(atomic.LoadPointer(&s.projects))
}

// setProjects sets the map of projects
func (s *story) setProjects(projects map[string]*project) {
	atomic.StorePointer(&s.projects, unsafe.Pointer(&projects))
}

// scan scans the entire story to build projects
func (s *story) scan() {
	// initialize the variables
	var wg sync.WaitGroup
	out := make(chan *project, 1000)
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

func (s *story) scanReducer(out chan *project, quit chan struct{}) {
	// initialize the variables
	projects := make(map[string]*project)

	// iterate over the channel
	for {
		select {
		case project, ok := <-out:
			if !ok {
				close(quit)
				// set the projects to s now
				s.setProjects(projects)
				return
			}
			projects[project.importPath] = project
		}
	}
}

func (s *story) scanWorker(wg *sync.WaitGroup, out chan *project, ipath string) {
	defer wg.Done()

	// do we have a .git folder here?
	if _, err := AppFS.Stat(path.Join(s.projectPath(ipath), ".git")); err == nil {
		log.Debug().Str("path", s.projectPath(ipath)).Msg("found a Git repository")
		// return this project
		out <- &project{
			story:      s,
			importPath: ipath,
		}

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
