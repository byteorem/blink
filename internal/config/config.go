package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Source       string   `toml:"source"`
	WowPath     string   `toml:"wowPath"`
	Ignore      []string `toml:"ignore"`
	UseGitignore bool    `toml:"useGitignore"`
}

func Defaults() Config {
	return Config{
		Source:       "auto",
		WowPath:     "auto",
		Ignore:      []string{},
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

func MergeFlags(cfg *Config, source, wowPath string) {
	if source != "" {
		cfg.Source = source
	}
	if wowPath != "" {
		cfg.WowPath = wowPath
	}
}
