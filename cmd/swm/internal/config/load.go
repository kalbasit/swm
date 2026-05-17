package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// ErrConfigNotFound is returned by Load when the config file does not exist.
var ErrConfigNotFound = errors.New("config file not found")

// ResolveConfigPath returns the effective config file path.
// When swmConfig (the value of $SWM_CONFIG) is non-empty it is returned as-is.
// Otherwise the XDG default <xdgConfigHome>/swm/config.toml is returned.
func ResolveConfigPath(swmConfig, xdgConfigHome string) string {
	if swmConfig != "" {
		return swmConfig
	}

	return filepath.Join(xdgConfigHome, "swm", "config.toml")
}

// Load reads the TOML config file at path and returns a Config with defaults applied.
// Returns ErrConfigNotFound if the file does not exist.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // user-specified config file path
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrConfigNotFound
		}

		return nil, fmt.Errorf("reading config file: %w", err)
	}

	cfg := &Config{
		CodeRoot:     "~/code",
		DefaultStory: "_default",
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	expanded, err := expandTilde(cfg.CodeRoot)
	if err != nil {
		return nil, fmt.Errorf("expanding code_root: %w", err)
	}

	cfg.CodeRoot = expanded

	for name, p := range cfg.Plugins.Paths {
		expanded, err := expandTilde(p)
		if err != nil {
			return nil, fmt.Errorf("expanding plugin path for %s: %w", name, err)
		}

		cfg.Plugins.Paths[name] = expanded
	}

	return cfg, nil
}

// expandTilde replaces a leading "~/" with the current user's home directory.
func expandTilde(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("looking up home directory: %w", err)
	}

	if path == "~" {
		return home, nil
	}

	return filepath.Join(home, path[2:]), nil
}
