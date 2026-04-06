// Package config handles loading Juli's configuration from disk.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SplitPreset defines a user-configured split pane preset.
type SplitPreset struct {
	Name      string `yaml:"name"`
	Command   string `yaml:"command"`
	Direction string `yaml:"direction"` // "horizontal" (right) or "vertical" (down)
	Percent   int    `yaml:"percent"`   // pane size 1-90
	Detach    bool   `yaml:"detach"`    // true = focus stays on original pane
}

// Config holds all Juli configuration. Currently only split presets.
type Config struct {
	Splits []SplitPreset `yaml:"splits"`
}

// Load reads the global config file at path. If path is empty, it defaults to
// ~/.config/juli/config.yaml. Returns an empty Config (not an error) when the
// file does not exist, so callers always get a usable value.
func Load(path string) (*Config, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return &Config{}, nil
		}
		path = filepath.Join(home, ".config", "juli", "config.yaml")
	}

	cfg, err := LoadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	return cfg, err
}

// LoadFile reads a config from a specific file path.
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	applyDefaults(&cfg)
	return &cfg, nil
}

// FindProjectConfig walks up from dir looking for a .juli.yaml file.
// Returns the file path if found, or "" if not found.
func FindProjectConfig(dir string) string {
	for {
		candidate := filepath.Join(dir, ".juli.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func applyDefaults(cfg *Config) {
	for i := range cfg.Splits {
		if cfg.Splits[i].Direction == "" {
			cfg.Splits[i].Direction = "horizontal"
		}
		if cfg.Splits[i].Percent == 0 {
			cfg.Splits[i].Percent = 50
		}
	}
}
