package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Source       string   `toml:"source"`
	WowPath      string   `toml:"wowPath"`
	Version      string   `toml:"version"`
	Ignore       []string `toml:"ignore"`
	UseGitignore bool     `toml:"useGitignore"`
}

func Defaults() Config {
	return Config{
		Source:       "auto",
		WowPath:      "auto",
		Version:      "retail",
		Ignore:       []string{},
		UseGitignore: true,
	}
}

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

var validVersions = map[string]bool{
	"retail":      true,
	"classic":     true,
	"classic_era": true,
}

func MergeFlags(cfg *Config, source, wowPath, version string) error {
	if source != "" {
		cfg.Source = source
	}
	if wowPath != "" {
		cfg.WowPath = wowPath
	}
	if version != "" {
		cfg.Version = version
	}

	if !validVersions[cfg.Version] {
		return fmt.Errorf("invalid version %q: must be retail, classic, or classic_era", cfg.Version)
	}

	return nil
}
