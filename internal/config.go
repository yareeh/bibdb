package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Backend struct {
	Path   string `yaml:"path"`
	Remote string `yaml:"remote"`
	Branch string `yaml:"branch"`
}

type Config struct {
	Default  string             `yaml:"default"`
	Backends map[string]Backend `yaml:"backends"`
}

func ConfigPath() string {
	if dir := os.Getenv("BIBDB_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "config.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "bibdb", "config.yaml")
}

func LoadConfig() (*Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Backends: make(map[string]Backend)}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if cfg.Backends == nil {
		cfg.Backends = make(map[string]Backend)
	}

	// Expand ~ in paths
	for name, b := range cfg.Backends {
		b.Path = expandHome(b.Path)
		cfg.Backends[name] = b
	}

	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// ExpandPath expands ~ in paths.
func ExpandPath(path string) string {
	return expandHome(path)
}

func expandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
