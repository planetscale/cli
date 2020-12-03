package config

import (
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/mitchellh/go-homedir"
)

const (
	defaultConfigPath = "~/.config/psctl"
)

type Config struct {
	AccessToken string
}

func New() *Config {
	var accessToken []byte
	_, err := os.Stat(AccessTokenPath())
	if err != nil {
		if !os.IsNotExist(err) {
			// TODO(iheanyi): Is fatal the right move here?
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
	}
}

func (c *Config) IsAuthenticated() bool {
	return c.AccessToken != ""
}

// ConfigDir is the directory for psctl config.
func ConfigDir() string {
	dir, _ := homedir.Expand(defaultConfigPath)
	return dir
}

// AccessTokenPath is the path for the access token file
func AccessTokenPath() string {
	return path.Join(ConfigDir(), "access-token")
}
