package workspace

import (
	"slices"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
)

const defaultStoryName = "_default"

// SortStoriesForPicker returns a new slice of stories sorted for display in the
// story picker: feature stories by CreatedAt descending (ties broken by name
// ascending), with the _default story always pinned as the last entry.
func SortStoriesForPicker(stories []*coreStory.Story) []*coreStory.Story {
	out := make([]*coreStory.Story, len(stories))
	copy(out, stories)

	slices.SortStableFunc(out, func(a, b *coreStory.Story) int {
		aDefault := a.Name == defaultStoryName
		bDefault := b.Name == defaultStoryName

		// _default is always last.
		switch {
		case aDefault && !bDefault:
			return 1
		case !aDefault && bDefault:
			return -1
		}

		// Both _default (shouldn't happen) or both non-default: newer first.
		if !a.CreatedAt.Equal(b.CreatedAt) {
			if a.CreatedAt.After(b.CreatedAt) {
				return -1
			}

			return 1
		}

		// Tie-break: lexicographic by name.
		if a.Name < b.Name {
			return -1
		}

		if a.Name > b.Name {
			return 1
		}

		return 0
	})

	return out
}
