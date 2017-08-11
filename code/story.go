package code

import (
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/spf13/afero"
)

type Story struct {
	// Name is the name of the story
	Name string

	// CodePath is the path of Code.Path
	CodePath string

	// ProfileName is the name of the profile for this story
	ProfileName string

	// Projects is a list of projects
	Projects map[string]*Project
}

// FindProjectBySessionName returns the project represented by the session name
func (s *Story) FindProjectBySessionName(name string) (*Project, error) {
	if project := s.Projects[strings.Replace(strings.Replace(name, dotChar, ".", -1), colonChar, ":", -1)]; project != nil {
		return project, nil
	}

	return nil, ErrProjectNotFound
}

// GoPath returns the absolute GOPATH of this story.
func (s *Story) GoPath() string {
	if s.Name == BaseStory {
		return path.Join(s.CodePath, s.ProfileName, s.Name)
	}

	return path.Join(s.CodePath, s.ProfileName, "stories", s.Name)
}

// Scan scans the entire story to build projects
func (s *Story) Scan() {
	// initialize the variables
	var wg sync.WaitGroup
	out := make(chan *Project, 1000)
	s.Projects = make(map[string]*Project)
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

// SessionNames returns the session names for this story
func (s *Story) SessionNames() []string {
	var res []string
	for _, project := range s.Projects {
		res = append(res, project.SessionName())
	}

	return res
}

func (s *Story) scanReducer(out chan *Project, quit chan struct{}) {
	for {
		select {
		case project, ok := <-out:
			if !ok {
				close(quit)
				return
			}
			s.Projects[project.ImportPath] = project
		}
	}
}

func (s *Story) scanWorker(wg *sync.WaitGroup, out chan *Project, ipath string) {
	defer wg.Done()

	// do we have a .git folder here?
	if _, err := AppFs.Stat(path.Join(s.projectPath(ipath), ".git")); err == nil {
		// return this project
		out <- &Project{
			ImportPath:  ipath,
			CodePath:    s.CodePath,
			ProfileName: s.ProfileName,
			StoryName:   s.Name,
		}

		return
	}

	// scan the folder
	entries, err := afero.ReadDir(AppFs, s.projectPath(ipath))
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Fatalf("error reading the directory %q: %s", s.projectPath(ipath), err)
	}
	for _, entry := range entries {
		// scan the entry if it's a directory
		if entry.IsDir() {
			wg.Add(1)
			go s.scanWorker(wg, out, path.Join(ipath, entry.Name()))
		}
	}
}

func (s *Story) projectPath(ipath string) string {
	return path.Join(s.GoPath(), srcDir, ipath)
}
