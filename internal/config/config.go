package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Feeds    []FeedConfig   `yaml:"feeds"`
	Interests []string      `yaml:"interests"`
	Ollama   OllamaConfig   `yaml:"ollama"`
	Raindrop RaindropConfig `yaml:"raindrop"`
	UI       UIConfig       `yaml:"ui"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type FeedConfig struct {
	URL  string `yaml:"url"`
	Name string `yaml:"name"`
}

type OllamaConfig struct {
	Host  string `yaml:"host"`
	Model string `yaml:"model"`
}

type RaindropConfig struct {
	APIToken string `yaml:"api_token"`
}

type UIConfig struct {
	RefreshInterval  string `yaml:"refresh_interval"`
	ArticleMaxAgeDays int   `yaml:"article_max_age_days"`
}

// GetRefreshInterval parses the refresh interval string
func (u *UIConfig) GetRefreshInterval() (time.Duration, error) {
	return time.ParseDuration(u.RefreshInterval)
}

// Load reads configuration from file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Expand home directory in database path
	if cfg.Database.Path != "" {
		cfg.Database.Path = expandPath(cfg.Database.Path)
	}

	// Set defaults
	if cfg.Ollama.Host == "" {
		cfg.Ollama.Host = "http://localhost:11434"
	}
	if cfg.Ollama.Model == "" {
		cfg.Ollama.Model = "llama2"
	}
	if cfg.UI.RefreshInterval == "" {
		cfg.UI.RefreshInterval = "15m"
	}
	if cfg.UI.ArticleMaxAgeDays == 0 {
		cfg.UI.ArticleMaxAgeDays = 14
	}

	return &cfg, nil
}

// Save writes configuration to file
func Save(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}

// DefaultConfigPath returns the default configuration file path
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.yaml"
	}
	return filepath.Join(home, ".config", "newsreader", "config.yaml")
}
