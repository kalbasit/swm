package story

import (
	"encoding/json"
	"os"
	"path"

	"github.com/adrg/xdg"
	"github.com/kalbasit/swm/ifaces"
	"github.com/pkg/errors"
)

// ErrNameRequired is returned if the name of the story was not passed in.
var ErrNameRequired = errors.New("the name of the story is required")

// ErrStoryExists is returned if the story already exists
var ErrStoryExists = errors.New("the story already exists")

type story struct {
	Name       string
	BranchName string
}

func newStory(name, branchName string) (*story, error) {
	if name == "" {
		return nil, ErrNameRequired
	}
	if branchName == "" {
		branchName = name
	}
	return &story{Name: name, BranchName: branchName}, nil
}

func New(name, branchName string) (ifaces.Story, error) {
	return newStory(name, branchName)
}

// Create creates a new story
func Create(name, branchName string) error {
	if name == "" {
		return ErrNameRequired
	}

	s, err := newStory(name, branchName)
	if err != nil {
		return err
	}

	if _, err := os.Stat(s.filePath()); err == nil {
		return ErrStoryExists
	}

	return s.Save()
}

// Remove saves the story in the data directory.
func (s *story) Remove() error { return os.Remove(s.filePath()) }

// Load returns the story identified by its name.
func Load(name string) (ifaces.Story, error) {
	if name == "" {
		return nil, ErrNameRequired
	}

	s := &story{Name: name}

	f, err := os.Open(s.filePath())
	if err != nil {
		return nil, errors.Wrap(err, "error opening the story file")
	}

	if err := json.NewDecoder(f).Decode(s); err != nil {
		return nil, errors.Wrap(err, "error decoding the story")
	}
	if s.Name == "" {
		return nil, ErrNameRequired
	}

	return s, nil
}

// SetName sets the name of the story.
func (s *story) SetName(v string) { s.Name = v }

// SetBranchName sets the name of the branch that will be used to create
// stories for projects.
func (s *story) SetBranchName(v string) { s.BranchName = v }

// GetName returns the name of the story
func (s *story) GetName() string { return s.Name }

// GetBranchName returns the name of the branch of this story
func (s *story) GetBranchName() string { return s.BranchName }

func (s *story) Save() error {
	if s.Name == "" {
		return ErrNameRequired
	}

	if _, err := os.Stat(path.Dir(s.filePath())); err != nil {
		if os.IsNotExist(err) {
			// make sure the parent directory exists before opening the file
			if err := os.MkdirAll(path.Dir(s.filePath()), 0777); err != nil {
				return errors.Wrap(err, "error creating the parent directory of the story file")
			}
		} else {
			return errors.Wrap(err, "error stat the parent directory of the story save file")
		}
	}

	// open the file for writing
	f, err := os.OpenFile(s.filePath(), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return errors.Wrap(err, "error opening the story file on the system")
	}

	defer f.Close()

	if err := json.NewEncoder(f).Encode(s); err != nil {
		return errors.Wrap(err, "error encoding the story as JSON")
	}

	return nil
}

func (s *story) filePath() string {
	return path.Join(xdg.DataHome, "swm", "stories", s.Name+".json")
}
