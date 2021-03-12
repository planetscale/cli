package config

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/mitchellh/go-homedir"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"gopkg.in/yaml.v2"
)

const (
	defaultConfigPath = "~/.config/planetscale"
	psdbFile          = ".psdb"
)

var tl = []string{"rev-parse", "--show-toplevel"}

type Config struct {
	AccessToken  string
	BaseURL      string
	Organization string

	ServiceTokenName string
	ServiceToken     string

	// Project Configuration
	Database string
	Branch   string
}

// WritableConfig maps
type WritableConfig struct {
	Organization string `yaml:"org" json:"org"`
}

type WritableProjectConfig struct {
	Database string `yaml:"database"`
	Branch   string `yaml:"branch"`
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

func ProjectConfigPath() (string, error) {
	basePath, err := GetRootGitRepoDir()
	if err != nil {
		return "", err
	}
	return path.Join(basePath, psdbFile), nil
}

// NewClientFromConfig creates a PlaentScale API client from our configuration
func (c *Config) NewClientFromConfig(clientOpts ...ps.ClientOption) (*ps.Client, error) {
	opts := []ps.ClientOption{
		ps.WithBaseURL(c.BaseURL),
	}
	if c.ServiceToken != "" && c.ServiceTokenName != "" {
		opts = append(opts, ps.WithServiceToken(c.ServiceTokenName, c.ServiceToken))
	} else {
		opts = append(opts, ps.WithAccessToken(c.AccessToken))
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

// ToWritableProjectConfig returns an instance of WritableProjectConfig from
// the Config struct.
func (c *Config) ToWritableProjectConfig() *WritableProjectConfig {
	return &WritableProjectConfig{
		Database: c.Database,
		Branch:   c.Branch,
	}
}

// WriteDefault persists the writable project config at the default path
// which is pulled from the root of the git repository if a user is in one.
func (w *WritableProjectConfig) WriteDefault() error {
	cfgFile, err := ProjectConfigPath()
	if err != nil {
		return err
	}

	d, err := yaml.Marshal(w)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(cfgFile, d, 0644)
}

func GetRootGitRepoDir() (string, error) {
	out, err := exec.Command("git", tl...).CombinedOutput()
	if err != nil {
		return "", errors.New("unable to find git root directory")
	}

	return string(strings.TrimSuffix(string(out), "\n")), nil
}

func GetProjectConfigFile() string {
	return psdbFile
}
