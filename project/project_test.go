package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStoryPath(t *testing.T) {
	t.Skip("not implemented yet")
	// // create a new project
	// p := &project{
	// 	story: &story{
	// 		name: "base",
	// 		profile: &profile{
	// 			name: "personal",
	// 			code: &code{
	// 				path: "/home/kalbasit/code",
	// 			},
	// 		},
	// 	},
	//
	// 	importPath: "github.com/kalbasit/swm",
	// }
	// // assert the Path
	// assert.Equal(t, "/home/kalbasit/code/personal/base/src/github.com/kalbasit/swm", p.Path())
}

func TestRepositoryPath(t *testing.T) { t.Skip("not implemented yet") }

func TestEnsure(t *testing.T) { t.Skip("not implemented yet") }

func TestString(t *testing.T) {
	assert.Equal(t, "github.com/kalbasit/swm", (&project{importPath: "github.com/kalbasit/swm"}).String())
}

func TestPath(t *testing.T) {
	t.Skip("not implemented yet")
}
