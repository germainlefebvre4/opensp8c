package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type WorkspaceConfig struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type Config struct {
	Workspaces []WorkspaceConfig `yaml:"workspaces"`
	configPath string
}

func Load(path string) (*Config, error) {
	cfg := &Config{configPath: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(c.configPath, data, 0644)
}
