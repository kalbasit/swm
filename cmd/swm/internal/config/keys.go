package config

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// ErrNotWritable is returned by KeyDef.Set when the key cannot be set via swm config set.
var ErrNotWritable = errors.New("not writable in this version; edit config.toml directly to change it")

// ErrUnknownKey is returned when a key path is not found in the registry.
var ErrUnknownKey = errors.New("unknown config key; run 'swm config list --all' to see valid keys")

// KeyDef describes one configurable key accessible via swm config.
type KeyDef struct {
	Path        string
	Description string
	Writable    bool
	get         func(cfg *Config) string
	set         func(cfg *Config, value string) error
}

// Get returns the string representation of this key's effective value.
func (k KeyDef) Get(cfg *Config) string { return k.get(cfg) }

// Set writes value into cfg. Returns ErrNotWritable if the key is not writable.
func (k KeyDef) Set(cfg *Config, value string) error {
	if !k.Writable {
		return fmt.Errorf("%q: %w", k.Path, ErrNotWritable)
	}

	return k.set(cfg, value)
}

// wildcardPathPrefix is the dot-path prefix for dynamic plugins.paths.* keys.
const wildcardPathPrefix = "plugins.paths."

// AllKeys returns all statically registered keys followed by any currently-configured
// plugins.paths.* entries (sorted for stable output).
func AllKeys(cfg *Config) []KeyDef {
	reg := keyRegistry()
	out := append([]KeyDef{}, reg...)

	if len(cfg.Plugins.Paths) > 0 {
		names := make([]string, 0, len(cfg.Plugins.Paths))
		for name := range cfg.Plugins.Paths {
			names = append(names, name)
		}

		sort.Strings(names)

		for _, name := range names {
			k, _ := LookupKey(wildcardPathPrefix + name)
			out = append(out, k)
		}
	}

	return out
}

// ConfiguredKeys returns KeyDef entries for all keys explicitly present in the
// config file at path. Returns nil, nil if the file does not exist.
func ConfiguredKeys(path string, cfg *Config) ([]KeyDef, error) {
	data, err := os.ReadFile(path) //nolint:gosec // user-specified config file path
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var raw map[string]any

	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	var result []KeyDef

	for _, k := range AllKeys(cfg) {
		if isKeyPresent(raw, k.Path) {
			result = append(result, k)
		}
	}

	return result, nil
}

// LookupKey finds a KeyDef by exact dot-path or by plugins.paths.<name> wildcard match.
func LookupKey(path string) (KeyDef, bool) {
	for _, k := range keyRegistry() {
		if k.Path == path {
			return k, true
		}
	}

	if strings.HasPrefix(path, wildcardPathPrefix) {
		name := path[len(wildcardPathPrefix):]
		if name == "" {
			return KeyDef{}, false
		}

		return KeyDef{
			Path:        path,
			Description: fmt.Sprintf("Explicit binary path for plugin %q", name),
			Writable:    true,
			get: func(cfg *Config) string {
				return cfg.Plugins.Paths[name]
			},
			set: func(cfg *Config, v string) error {
				if cfg.Plugins.Paths == nil {
					cfg.Plugins.Paths = make(map[string]string)
				}

				cfg.Plugins.Paths[name] = v

				return nil
			},
		}, true
	}

	return KeyDef{}, false
}

// isKeyPresent reports whether a dot-separated path exists in a nested map.
func isKeyPresent(m map[string]any, dotPath string) bool {
	if m == nil {
		return false
	}

	top, rest, found := strings.Cut(dotPath, ".")
	if !found {
		_, ok := m[dotPath]

		return ok
	}

	sub, ok := m[top]
	if !ok {
		return false
	}

	subMap, ok := sub.(map[string]any)
	if !ok {
		return false
	}

	return isKeyPresent(subMap, rest)
}

// keyRegistry returns the ordered list of all statically known config keys.
func keyRegistry() []KeyDef {
	return []KeyDef{
		{
			Path:        "code_root",
			Description: "Root directory for all code repositories (default: ~/code)",
			Writable:    true,
			get:         func(cfg *Config) string { return cfg.CodeRoot },
			set: func(cfg *Config, v string) error {
				cfg.CodeRoot = v

				return nil
			},
		},
		{
			Path:        "default_story",
			Description: "Name of the default story (default: _default)",
			Writable:    true,
			get:         func(cfg *Config) string { return cfg.DefaultStory },
			set: func(cfg *Config, v string) error {
				cfg.DefaultStory = v

				return nil
			},
		},
		{
			Path:        "plugins.session",
			Description: "Session plugin name (e.g. tmux)",
			Writable:    true,
			get:         func(cfg *Config) string { return cfg.Plugins.Session },
			set: func(cfg *Config, v string) error {
				cfg.Plugins.Session = v

				return nil
			},
		},
		{
			Path:        "plugins.vcs",
			Description: "VCS plugin name (e.g. git)",
			Writable:    true,
			get:         func(cfg *Config) string { return cfg.Plugins.VCS },
			set: func(cfg *Config, v string) error {
				cfg.Plugins.VCS = v

				return nil
			},
		},
		{
			Path:        "plugins.picker",
			Description: "Picker plugin name (e.g. fzf)",
			Writable:    true,
			get:         func(cfg *Config) string { return cfg.Plugins.Picker },
			set: func(cfg *Config, v string) error {
				cfg.Plugins.Picker = v

				return nil
			},
		},
		{
			Path:        "plugins.forges",
			Description: "Forge plugin names (read-only via set; edit config.toml to change)",
			Writable:    false,
			get: func(cfg *Config) string {
				if len(cfg.Plugins.Forges) == 0 {
					return "[]"
				}

				parts := make([]string, len(cfg.Plugins.Forges))
				for i, f := range cfg.Plugins.Forges {
					parts[i] = `"` + f + `"`
				}

				return "[" + strings.Join(parts, ", ") + "]"
			},
			set: nil,
		},
		{
			Path:        "story.branch_name_template",
			Description: `Go template for branch names on story create (default: feat/{{.Name}})`,
			Writable:    true,
			get:         func(cfg *Config) string { return cfg.Story.BranchNameTemplate },
			set: func(cfg *Config, v string) error {
				cfg.Story.BranchNameTemplate = v

				return nil
			},
		},
	}
}
