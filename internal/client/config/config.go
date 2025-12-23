package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Token string `yaml:"token"`
}

// ProjectConfig represents gopublic.yaml project configuration
type ProjectConfig struct {
	Version string             `yaml:"version"`
	Tunnels map[string]*Tunnel `yaml:"tunnels"`
}

// Tunnel represents a single tunnel configuration
type Tunnel struct {
	Proto     string `yaml:"proto"`     // http, https, tcp
	Addr      string `yaml:"addr"`      // local port or host:port
	Subdomain string `yaml:"subdomain"` // subdomain to bind
}

func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gopublic"), nil
}

func LoadConfig() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// LoadProjectConfig loads gopublic.yaml from the specified path or current directory
func LoadProjectConfig(path string) (*ProjectConfig, error) {
	if path == "" {
		path = "gopublic.yaml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
