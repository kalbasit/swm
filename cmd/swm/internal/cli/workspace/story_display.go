package workspace

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"

	"github.com/kalbasit/swm/cmd/swm/internal/ageformat"
)

const (
	displaySep     = "   " // separator between columns
	projectSep     = " · "
	ellipsis       = "…"
	mainRepoSuffix = " (main repo)"
)

// BuildStoryDisplay returns a terminal-width-aware display string for a story
// picker entry. The format is:
//
//	<name>[ (<branch>)]   <age>   [<proj1> · <proj2> · …]
//
// Branch is shown only when it differs from the story name, except for the
// _default story which always shows "(main repo)" instead.
// Truncation priority (right-to-left): projects → branch → story name.
func BuildStoryDisplay(s *coreStory.Story, width int, now time.Time) string {
	age := ageformat.FormatAge(s.CreatedAt, now)

	// Build the left column: name with optional branch/label.
	var nameCol string

	if s.Name == defaultStoryName {
		nameCol = defaultStoryName + mainRepoSuffix
	} else if s.BranchName != "" && s.BranchName != s.Name {
		nameCol = s.Name + " (" + s.BranchName + ")"
	} else {
		nameCol = s.Name
	}

	// Build the projects string.
	projects := buildProjectsStr(s)

	// Assemble without any truncation and check length.
	full := assembleLine(nameCol, age, projects)
	if utf8.RuneCountInString(full) <= width {
		return full
	}

	// Step 1: trim projects list.
	if len(s.Projects) > 0 {
		trimmed := trimProjects(s, age, nameCol, width)
		if trimmed != "" {
			return trimmed
		}
	}

	// Step 2: trim branch name (if present and not _default).
	if s.Name != defaultStoryName && s.BranchName != "" && s.BranchName != s.Name {
		nameCol = trimBranch(s.Name, s.BranchName, age, width)

		line := assembleLine(nameCol, age, "")
		if utf8.RuneCountInString(line) <= width {
			return line
		}
	}

	// Step 3: trim story name itself (last resort).
	return trimStoryName(s.Name, age, width)
}

// buildProjectsStr joins attached project keys with the project separator.
func buildProjectsStr(s *coreStory.Story) string {
	if len(s.Projects) == 0 {
		return ""
	}

	keys := make([]string, 0, len(s.Projects))

	for i := range s.Projects {
		p := &s.Projects[i]
		keys = append(keys, p.Host+"/"+strings.Join(p.Segments, "/"))
	}

	return strings.Join(keys, projectSep)
}

// assembleLine joins the three columns with separators.
func assembleLine(nameCol, age, projects string) string {
	var sb strings.Builder

	sb.WriteString(nameCol)
	sb.WriteString(displaySep)
	sb.WriteString(age)

	if projects != "" {
		sb.WriteString(displaySep)
		sb.WriteString(projects)
	}

	return sb.String()
}

// trimProjects tries dropping trailing projects until the line fits width.
// Returns "" if even zero projects don't fit.
func trimProjects(s *coreStory.Story, age, nameCol string, width int) string {
	keys := make([]string, 0, len(s.Projects))

	for i := range s.Projects {
		p := &s.Projects[i]
		keys = append(keys, p.Host+"/"+strings.Join(p.Segments, "/"))
	}

	// Try progressively fewer projects, adding ellipsis when trimming.
	for n := range slices.Backward(keys) {
		var proj string

		if n == len(keys)-1 {
			// All projects — already tried, doesn't fit.
			proj = strings.Join(keys, projectSep)
		} else if n == 0 {
			// Just the first project + ellipsis.
			proj = keys[0] + projectSep + ellipsis
		} else {
			proj = strings.Join(keys[:n], projectSep) + projectSep + ellipsis
		}

		line := assembleLine(nameCol, age, proj)
		if utf8.RuneCountInString(line) <= width {
			return line
		}
	}

	// No projects at all.
	line := assembleLine(nameCol, age, "")
	if utf8.RuneCountInString(line) <= width {
		return line
	}

	return ""
}

// trimBranch builds a nameCol with the branch truncated to fit within width
// (considering the age suffix). Returns the best nameCol string found.
func trimBranch(storyName, branch, age string, width int) string {
	// Fixed cost: story name + " (" + ")" + sep + age
	fixed := utf8.RuneCountInString(storyName) +
		utf8.RuneCountInString(displaySep) +
		utf8.RuneCountInString(age) +
		4 // " (" + ")"
	available := width - fixed

	if available <= 1 {
		// No room for branch at all.
		return storyName
	}

	branchRunes := []rune(branch)
	if len(branchRunes) > available {
		branchRunes = branchRunes[:available-utf8.RuneCountInString(ellipsis)]

		return storyName + " (" + string(branchRunes) + ellipsis + ")"
	}

	return storyName + " (" + branch + ")"
}

// trimStoryName truncates the story name itself to fit width with the age.
func trimStoryName(name, age string, width int) string {
	// Minimum output: name + sep + age
	base := assembleLine(name, age, "")
	if utf8.RuneCountInString(base) <= width {
		return base
	}

	// Available for name: width - sep - age
	available := width -
		utf8.RuneCountInString(displaySep) -
		utf8.RuneCountInString(age) -
		utf8.RuneCountInString(ellipsis)
	if available <= 0 {
		return name // can't truncate meaningfully; return as-is
	}

	nameRunes := []rune(name)
	if len(nameRunes) > available {
		nameRunes = nameRunes[:available]
	}

	return string(nameRunes) + ellipsis + displaySep + age
}
