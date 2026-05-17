package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

	return cfg, nil
}
