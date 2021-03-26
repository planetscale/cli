package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"
)

// GlobalConfig is sourced from the global config path and contains globaly
// configurable options.
type GlobalConfig struct {
	Organization string `yaml:"org" json:"org"`
}

// DefaultGlobalConfig returns the global config from the default config path.
func DefaultGlobalConfig() (*GlobalConfig, error) {
	dir, err := homedir.Expand(defaultConfigPath)
	if err != nil {
		return nil, fmt.Errorf("can't expand path %q: %s", defaultConfigPath, err)
	}

	configFile := path.Join(dir, "config.yml")

	out, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var cfg *GlobalConfig
	err = yaml.Unmarshal(out, &cfg)
	if err != nil {
		return nil, fmt.Errorf("can't unmarshal file %q: %s", configFile, err)
	}

	return cfg, nil
}

// Write persists the writable global config at the designated path.
func (g *GlobalConfig) Write(path string) error {
	if path == "" {
		return errors.New("path is empty")
	}

	d, err := yaml.Marshal(g)
	if err != nil {
		return fmt.Errorf("can't marshal global config: %s", err)
	}

	return ioutil.WriteFile(path, d, 0644)
}

// WriteDefault persists the writable global config to the default global path.
func (g *GlobalConfig) WriteDefault() error {
	d, err := yaml.Marshal(g)
	if err != nil {
		return err
	}

	path := DefaultGlobalConfigPath()
	return ioutil.WriteFile(path, d, 0644)
}

// DefaultGlobalConfigPath returns the default path for the global config file.
func DefaultGlobalConfigPath() string {
	return path.Join(ConfigDir(), "config.yml")
}
