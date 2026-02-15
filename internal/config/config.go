// Package config handles loading and merging blink configuration.
package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config holds blink configuration from blink.toml and CLI flags.
type Config struct {
	Source       string   `toml:"source"`
	WowPath      string   `toml:"wowPath"`
	Ignore       []string `toml:"ignore"`
	UseGitignore bool     `toml:"useGitignore"`
}

// Defaults returns a Config with default values.
func Defaults() Config {
	return Config{
		Source:       "auto",
		WowPath:      "auto",
		Ignore:       []string{},
		UseGitignore: true,
	}
}

// Load reads blink.toml if present and returns the merged config.
func Load() (Config, error) {
	cfg := Defaults()

	if _, err := os.Stat("blink.toml"); os.IsNotExist(err) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile("blink.toml", &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse blink.toml: %w", err)
	}

	return cfg, nil
}

// MergeFlags overrides config values with non-empty CLI flags.
func MergeFlags(cfg *Config, source, wowPath string) {
	if source != "" {
		cfg.Source = source
	}
	if wowPath != "" {
		cfg.WowPath = wowPath
	}
}
