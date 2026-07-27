package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Agent struct {
		UUID    string `yaml:"id"`
		Name    string `yaml:"name"`
		Version string `yamal"verison`
	} `yaml:"agent"`

	Panel struct {
		URL   string `yaml:"url"`
		Token string `yaml:"token"`
	} `yaml:"panel"`

	Heartbeat struct {
		Interval int `yaml:"interval"`
	} `yaml:"heartbeat"`

	Terminal struct {
		Shell string `yaml:"shell"`
	} `yaml:"terminal"`

	Monitor struct {
		Interval int `yaml:"interval"`
	} `yaml:"monitor"`

	Logger struct {
		Level string `yaml:"level"`
	} `yaml:"logger"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}
