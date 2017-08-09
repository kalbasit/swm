package tmx

import (
	"log"
	"path"
	"sync"

	"github.com/spf13/afero"
)

const baseWorkspaceName = "base"

type Profile struct {
	// Name is the name of the profile
	Name string

	// CodePath is the path of Code.Path
	CodePath string

	// Workspaces is a list of workspaces
	Workspaces map[string]*Workspace
}

// BaseWorkspace returns the base workspace
func (p *Profile) BaseWorkspace() *Workspace {
	return p.Workspaces[baseWorkspaceName]
}

// Path returns the absolute path of the profile
func (p *Profile) Path() string {
	return path.Join(p.CodePath, p.Name)
}

// Scan scans the entire profile to build the workspaces
func (p *Profile) Scan() {
	// initialize the variables
	var wg sync.WaitGroup
	p.Workspaces = make(map[string]*Workspace)
	// read the profile and scan all workspaces
	entries, err := afero.ReadDir(AppFs, p.Path())
	if err != nil {
		log.Printf("error reading the directory %q: %s", p.Path(), err)
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			// create the workspace
			w := &Workspace{
				Name:        entry.Name(),
				CodePath:    p.CodePath,
				ProfileName: p.Name,
			}
			// start scanning it
			wg.Add(1)
			go func() {
				defer wg.Done()
				w.Scan()
			}()
			// add it to the profile
			p.Workspaces[entry.Name()] = w
		}
	}
	wg.Wait()
}

// SessionNames returns the session names for projects in all workspaces of this profile
func (p *Profile) SessionNames() []string {
	var res []string
	for _, workspace := range p.Workspaces {
		for _, project := range workspace.Projects {
			res = append(res, project.SessionName())
		}
	}

	return res
}
