// Package story provides the domain types and persistence for swm stories.
package story

import (
	"errors"
	"time"
)

// Sentinel errors returned by the Store.
var (
	ErrStoryExists            = errors.New("story already exists")
	ErrStoryNotFound          = errors.New("story not found")
	ErrProjectAlreadyAttached = errors.New("project already attached to story")
)

// Project records a repository attached to a story.
type Project struct {
	Host       string    `json:"host"`
	Segments   []string  `json:"segments"`
	VCS        string    `json:"vcs,omitempty"`
	AttachedAt time.Time `json:"attached_at"`
}

// Story is the domain object representing a unit of work.
type Story struct {
	Name       string         `json:"name"`
	BranchName string         `json:"branch_name"`
	CreatedAt  time.Time      `json:"created_at"`
	VCS        string         `json:"vcs,omitempty"`
	Projects   []Project      `json:"projects"`
	Metadata   map[string]any `json:"metadata"`
}
