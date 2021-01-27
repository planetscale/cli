package config

import (
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/mitchellh/go-homedir"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"gopkg.in/yaml.v2"
)

const (
	defaultConfigPath = "~/.config/planetscale"
)

type Config struct {
	AccessToken  string
	BaseURL      string
	Organization string
}

// WritableConfig maps
type WritableConfig struct {
	Organization string `yaml:"org" json:"org"`
}

func New() *Config {
	var accessToken []byte
	_, err := os.Stat(AccessTokenPath())
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatal(err)
		}
	} else {
		accessToken, err = ioutil.ReadFile(AccessTokenPath())
		if err != nil {
			log.Fatal(err)
		}
	}

	return &Config{
		AccessToken: string(accessToken),
		BaseURL:     ps.DefaultBaseURL,
	}
}

func (c *Config) IsAuthenticated() bool {
	return c.AccessToken != ""
}

// ConfigDir is the directory for PlanetScale config.
func ConfigDir() string {
	dir, _ := homedir.Expand(defaultConfigPath)
	return dir
}

// DefaultConfigPath is the path for the config file.
func DefaultConfigPath() string {
	return path.Join(ConfigDir(), "config.yml")
}

// AccessTokenPath is the path for the access token file
func AccessTokenPath() string {
	return path.Join(ConfigDir(), "access-token")
}

// NewClientFromConfig creates a PlaentScale API client from our configuration
func (c *Config) NewClientFromConfig(clientOpts ...ps.ClientOption) (*ps.Client, error) {
	opts := []ps.ClientOption{
		ps.WithBaseURL(c.BaseURL),
		ps.WithAccessToken(c.AccessToken),
	}
	opts = append(opts, clientOpts...)

	return ps.NewClient(opts...)
}

// ToWritableConfig returns an instance of WritableConfig from the Config
// struct.
func (c *Config) ToWritableConfig() *WritableConfig {
	return &WritableConfig{
		Organization: c.Organization,
	}
}

// Write persists the writable config at the designated path.
func (w *WritableConfig) Write(path string) error {
	if path == "" {
		path = DefaultConfigPath()
	}

	d, err := yaml.Marshal(w)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, d, 0644)
	if err != nil {
		return err
	}

	return nil
}
