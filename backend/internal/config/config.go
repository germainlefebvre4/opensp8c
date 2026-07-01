package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type WorkspaceConfig struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

const (
	defaultChangeLogRetentionDays  = 15
	defaultExploreLogRetentionDays = 15
)

type Config struct {
	Workspaces              []WorkspaceConfig `yaml:"workspaces"`
	ChangeLogRetentionDays  int               `yaml:"changeLogRetentionDays,omitempty"`
	ExploreLogRetentionDays int               `yaml:"exploreLogRetentionDays,omitempty"`
	configPath              string
}

// ChangeLogRetentionDaysOrDefault returns the configured retention window for
// change conversation logs, falling back to the default when unset or invalid.
func (c *Config) ChangeLogRetentionDaysOrDefault() int {
	if c.ChangeLogRetentionDays <= 0 {
		return defaultChangeLogRetentionDays
	}
	return c.ChangeLogRetentionDays
}

// ExploreLogRetentionDaysOrDefault returns the configured retention window for
// unpromoted exploration conversation logs, falling back to the default when unset or invalid.
func (c *Config) ExploreLogRetentionDaysOrDefault() int {
	if c.ExploreLogRetentionDays <= 0 {
		return defaultExploreLogRetentionDays
	}
	return c.ExploreLogRetentionDays
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
