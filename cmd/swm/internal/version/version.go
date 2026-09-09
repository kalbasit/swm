// Package version reports the version the running binary was built from.
//
// The version is decided by the build, never by a value a reader could edit
// after the fact: the Nix packages inject it at link time, and the Go
// toolchain records a fallback for builds that bypass Nix. Nothing here
// carries a release-shaped default, so a build whose version was never
// injected reports that it is a development build rather than impersonating
// the last release.
package version

import "runtime/debug"

const (
	// develPlaceholder is what the Go toolchain records as the main module's
	// version when the binary was built from a work tree rather than resolved
	// from the module proxy at a tag.
	develPlaceholder = "(devel)"

	// develPrefix marks a version that no release tag stands behind. It is
	// deliberately not of the form vX.Y.Z so that such a build cannot be read
	// as a release.
	develPrefix = "devel"
)

// version is injected at link time by the build. It is deliberately empty by
// default: a build that forgets to inject one falls through to the build
// information below rather than claiming to be a release.
var version string

// Version returns the version of the running binary.
func Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		info = nil
	}

	return Resolve(version, info)
}

// Resolve returns the version a binary reports, given the version its linker
// was told to embed and whatever the Go toolchain recorded about the build.
//
// The link-time value wins: it is the only source that knows about release
// tags, because a tag reaches a build only when the builder injects it. Absent
// that, the module version records the tag a `go install module@tag` resolved,
// and the recorded revision identifies any other build exactly.
func Resolve(linked string, info *debug.BuildInfo) string {
	if linked != "" {
		return linked
	}

	if info == nil {
		return develPrefix
	}

	if info.Main.Version != "" && info.Main.Version != develPlaceholder {
		return info.Main.Version
	}

	var revision, modified string

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value
		}
	}

	if revision == "" {
		return develPrefix
	}

	resolved := develPrefix + "+" + revision
	if modified == "true" {
		resolved += "-dirty"
	}

	return resolved
}
