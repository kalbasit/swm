// Package config loads and represents the swm host configuration.
package config

// Plugins contains names and per-plugin config for all capabilities.
// This maps directly to the [plugins] TOML table.
type Plugins struct {
	Session string   `toml:"session"`
	VCS     string   `toml:"vcs"`
	Picker  string   `toml:"picker"`
	Forges  []string `toml:"forges"`

	// Paths contains explicit binary paths keyed by plugin name, e.g. "vcs-git" -> "/usr/bin/swm-plugin-vcs-git".
	Paths map[string]string `toml:"paths"`

	// Config holds per-plugin raw config sections, keyed by plugin name.
	// Each value is the raw key/value map from [plugins.config.<name>].
	Config map[string]map[string]any `toml:"config"`
}

// Config is the parsed representation of $XDG_CONFIG_HOME/swm/config.toml.
type Config struct {
	CodeRoot     string  `toml:"code_root"`
	DefaultStory string  `toml:"default_story"`
	Plugins      Plugins `toml:"plugins"`

	// HooksConfigHome overrides the XDG config home used for hook discovery.
	// When empty, the system XDG config home is used. Set in tests to avoid
	// writing hooks into the real user config directory.
	HooksConfigHome string `toml:"-"`
}

// Defaults returns a Config populated with default values (no file required).
func Defaults() *Config {
	return &Config{
		CodeRoot:     "~/code",
		DefaultStory: "_default",
	}
}
