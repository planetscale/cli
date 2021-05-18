package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/mitchellh/go-homedir"
	exec "golang.org/x/sys/execabs"
)

const (
	defaultConfigPath = "~/.config/planetscale"
	projectConfigName = ".pscale.yml"
	configName        = "pscale.yml"
	TokenFileMode     = 0600
)

// Config is dynamically sourced from various files and environment variables.
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

func New() (*Config, error) {
	var accessToken []byte
	tokenPath, err := AccessTokenPath()
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(tokenPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatal(err)
		}
	} else {
		if stat.Mode()&^TokenFileMode != 0 {
			err = os.Chmod(tokenPath, TokenFileMode)
			if err != nil {
				log.Printf("Unable to change %v file mode to 0%o: %v", tokenPath, TokenFileMode, err)
			}
		}
		accessToken, err = ioutil.ReadFile(tokenPath)
		if err != nil {
			log.Fatal(err)
		}
	}

	return &Config{
		AccessToken: string(accessToken),
		BaseURL:     ps.DefaultBaseURL,
	}, nil
}

func (c *Config) IsAuthenticated() bool {
	return ((c.ServiceToken != "" && c.ServiceTokenName != "") || (c.AccessToken != ""))
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

// ConfigDir is the directory for PlanetScale config.
func ConfigDir() (string, error) {
	dir, err := homedir.Expand(defaultConfigPath)
	if err != nil {
		return "", fmt.Errorf("can't expand path %q: %s", defaultConfigPath, err)
	}

	return dir, nil
}

// AccessTokenPath is the path for the access token file
func AccessTokenPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}

	return path.Join(dir, "access-token"), nil
}

// ProjectConfigPath returns the path of a configuration inside a Git
// repository.
func ProjectConfigPath() (string, error) {
	basePath, err := RootGitRepoDir()
	if err == nil {
		return path.Join(basePath, projectConfigName), nil
	}
	return path.Join("", projectConfigName), nil
}

func RootGitRepoDir() (string, error) {
	var tl = []string{"rev-parse", "--show-toplevel"}
	out, err := exec.Command("git", tl...).CombinedOutput()
	if err != nil {
		return "", errors.New("unable to find git root directory")
	}

	return string(strings.TrimSuffix(string(out), "\n")), nil
}

func ProjectConfigFile() string {
	return projectConfigName
}
