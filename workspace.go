package tmx

import (
	"log"
	"path"
	"sync"

	"github.com/spf13/afero"
)

type Workspace struct {
	// Name is the name of the workspace
	Name string

	// CodePath is the path of Code.Path
	CodePath string

	// ProfileName is the name of the profile for this workspace
	ProfileName string

	// Projects is a list of projects
	Projects map[string]*Project
}

func (w *Workspace) Path() string {
	return path.Join(w.CodePath, w.ProfileName, w.Name)
}

// Scan scans the entire workspace to build projects
func (w *Workspace) Scan() {
	// initialize the variables
	var wg sync.WaitGroup
	out := make(chan *Project, 1000)
	w.Projects = make(map[string]*Project)
	// start the workers
	wg.Add(1)
	go w.scanWorker(&wg, out, "")
	// start the reducer
	reducerQuit := make(chan struct{})
	go w.scanReducer(out, reducerQuit)
	// wait for the workers to return
	wg.Wait()
	// ask the reducer to die
	close(out)
	<-reducerQuit
}

func (w *Workspace) scanReducer(out chan *Project, quit chan struct{}) {
	for {
		select {
		case project, ok := <-out:
			if !ok {
				close(quit)
				return
			}
			w.Projects[project.ImportPath] = project
		}
	}
}

func (w *Workspace) scanWorker(wg *sync.WaitGroup, out chan *Project, ipath string) {
	defer wg.Done()

	// do we have a .git folder here?
	if _, err := AppFs.Stat(path.Join(w.projectPath(ipath), ".git")); err == nil {
		// return this project
		out <- &Project{
			ImportPath:    ipath,
			CodePath:      w.CodePath,
			ProfileName:   w.ProfileName,
			WorkspaceName: w.Name,
		}

		return
	}

	// scan the folder
	entries, err := afero.ReadDir(AppFs, w.projectPath(ipath))
	if err != nil {
		log.Printf("error reading the directory %q: %s", w.projectPath(ipath), err)
		return
	}
	for _, entry := range entries {
		// scan the entry if it's a directory
		if entry.IsDir() {
			wg.Add(1)
			go w.scanWorker(wg, out, path.Join(ipath, entry.Name()))
		}
	}
}

func (w *Workspace) projectPath(ipath string) string {
	return path.Join(w.Path(), srcDir, ipath)
}
