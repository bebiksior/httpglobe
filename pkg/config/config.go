package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// Config represents the application configuration loaded from a JSON file
type Config struct {
	Countries []string    `json:"countries"`
	Proxy     ProxyConfig `json:"proxy"`
	log       *logrus.Logger
	path      string
}

type ProxyConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func CreateDefaultConfig(path string, log *logrus.Logger) error {
	cfg := Config{
		Countries: []string{"cn", "in", "us", "jp", "de"},
		Proxy: ProxyConfig{
			Host:     "example.com",
			Port:     "12345",
			Username: "username",
			Password: "password",
		},
		log:  log,
		path: path,
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

func Load(path string, log *logrus.Logger) (*Config, error) {
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("getting home directory: %w", err)
		}
		path = filepath.Join(homeDir, ".config", "httpglobe", "config.json")
	}

	// Create config directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating config directory: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Infof("Config file not found at %s, creating default config", path)
			if err := CreateDefaultConfig(path, log); err != nil {
				return nil, fmt.Errorf("creating default config: %w", err)
			}

			log.Info("Default config file created, update it with your BrightData proxy credentials")
			os.Exit(1)
		} else {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	cfg.log = log
	cfg.path = path

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if len(c.Countries) == 0 {
		return fmt.Errorf("no countries specified")
	}

	if c.Proxy.Host == "" {
		return fmt.Errorf("proxy host is required")
	}

	if c.Proxy.Host == "example.com" {
		c.log.Warn("Looks like you're using the default proxy credentials. Please update them in your config file.")
		c.log.Warnf("Default config path: %s", c.path)
		return fmt.Errorf("default proxy credentials")
	}

	if c.Proxy.Port == "" {
		return fmt.Errorf("proxy port is required")
	}
	if c.Proxy.Username == "" {
		return fmt.Errorf("proxy username is required")
	}
	if c.Proxy.Password == "" {
		return fmt.Errorf("proxy password is required")
	}

	for _, country := range c.Countries {
		if len(country) != 2 {
			return fmt.Errorf("invalid country code: %s (must be 2 letters)", country)
		}
	}

	return nil
}
