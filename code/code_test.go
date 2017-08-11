package code

import (
	"regexp"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCodeScan(t *testing.T) {
	// swap the filesystem
	oldAppFS := AppFs
	AppFs = afero.NewMemMapFs()
	defer func() { AppFs = oldAppFS }()
	// create the filesystem we want to scan
	prepareFilesystem(t.Name())
	// create a code
	c := &code{
		path:           "/code",
		excludePattern: regexp.MustCompile("^.snapshots$"),
	}
	// define the assertion function
	assertFn := func() {
		// create the expected structs
		p := &profile{
			code: c,
			name: "TestCodeScan",
		}
		p.stories = map[string]*story{
			"base": &story{
				profile: p,
				name:    "base",
			},
			"STORY-123": &story{
				profile: p,
				name:    "STORY-123",
			},
		}
		p.stories["base"].projects = map[string]*project{
			"github.com/kalbasit/swm": &project{
				story:      p.stories["base"],
				importPath: "github.com/kalbasit/swm",
			},
			"github.com/kalbasit/dotfiles": &project{
				story:      p.stories["base"],
				importPath: "github.com/kalbasit/dotfiles",
			},
		}
		p.stories["STORY-123"].projects = map[string]*project{
			"github.com/kalbasit/dotfiles": &project{
				story:      p.stories["STORY-123"],
				importPath: "github.com/kalbasit/dotfiles",
			},
		}
		expected := map[string]*profile{"TestCodeScan": p}

		assert.Equal(t, expected["TestCodeScan"].stories["base"].name, c.profiles["TestCodeScan"].stories["base"].name)
		assert.Equal(t, expected["TestCodeScan"].stories["base"].profile, c.profiles["TestCodeScan"].stories["base"].profile)
		assert.Equal(t, expected["TestCodeScan"].stories["base"].projects["github.com/kalbasit/swm"], c.profiles["TestCodeScan"].stories["base"].projects["github.com/kalbasit/swm"])
		assert.Equal(t, expected["TestCodeScan"].stories["base"].projects["github.com/kalbasit/dotfiles"], c.profiles["TestCodeScan"].stories["base"].projects["github.com/kalbasit/dotfiles"])
		assert.Equal(t, expected["TestCodeScan"].stories["STORY-123"].name, c.profiles["TestCodeScan"].stories["STORY-123"].name)
		assert.Equal(t, expected["TestCodeScan"].stories["STORY-123"].profile, c.profiles["TestCodeScan"].stories["STORY-123"].profile)
		assert.Equal(t, expected["TestCodeScan"].stories["STORY-123"].projects["github.com/kalbasit/dotfiles"], c.profiles["TestCodeScan"].stories["STORY-123"].projects["github.com/kalbasit/dotfiles"])
	}
	// scan now
	c.scan()
	assertFn()
}
