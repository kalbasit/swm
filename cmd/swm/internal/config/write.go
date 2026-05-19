package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// LoadForWrite reads the config file at path for the purpose of mutation.
// Unlike Load, it does not apply defaults or expand tildes — it returns only
// what is explicitly present in the file. Returns an empty *Config if the file
// does not exist, so callers can create a new config via Save.
func LoadForWrite(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // user-specified config file path
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}

		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return &cfg, nil
}

// Save writes cfg to path atomically, creating parent directories if absent.
// Only non-zero fields are written (via omitempty struct tags on Config).
// Any comments in the original file are not preserved.
func Save(path string, cfg *Config) error {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("writing temp config file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("renaming config file: %w", err)
	}

	return nil
}
