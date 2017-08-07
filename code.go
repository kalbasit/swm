package tmx

import "github.com/spf13/afero"

// AppFs represents the filesystem of the app. It is exported to be used as a
// test helper.
var AppFs afero.Fs

func init() {
	AppFs = afero.NewOsFs()
}

type Code struct {
	// Path is the base path of this profile
	Path string

	Profiles map[string]*Profile
}
