package config

import (
	"fmt"
	"maps"
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
// Unknown fields already present in the file are preserved via deep merge so
// that future versions of swm (or manual additions) are not silently dropped.
// Comments in the original file are not preserved.
func Save(path string, cfg *Config) error {
	// Load existing raw TOML to preserve fields not known to this Config struct.
	rawData := make(map[string]any)

	if existing, err := os.ReadFile(path); err == nil { //nolint:gosec // user-specified config file path
		if err := toml.Unmarshal(existing, &rawData); err != nil {
			rawData = make(map[string]any) // reset in case of partial parse
		}
	}

	cfgBytes, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	var cfgMap map[string]any

	if err := toml.Unmarshal(cfgBytes, &cfgMap); err != nil {
		return fmt.Errorf("parsing config map: %w", err)
	}

	data, err := toml.Marshal(deepMerge(rawData, cfgMap))
	if err != nil {
		return fmt.Errorf("marshaling merged config: %w", err)
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

// deepMerge returns a new map containing all entries from base with entries
// from override applied on top. Nested maps are merged recursively; all other
// value types are replaced by the override value.
func deepMerge(base, override map[string]any) map[string]any {
	result := make(map[string]any, len(base))
	maps.Copy(result, base)

	for k, v := range override {
		if baseVal, ok := result[k]; ok {
			baseMap, baseIsMap := baseVal.(map[string]any)
			overrideMap, overrideIsMap := v.(map[string]any)

			if baseIsMap && overrideIsMap {
				result[k] = deepMerge(baseMap, overrideMap)

				continue
			}
		}

		result[k] = v
	}

	return result
}
