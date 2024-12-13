package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/99designs/keyring"
	"github.com/pkg/errors"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/mitchellh/go-homedir"
	exec "golang.org/x/sys/execabs"
)

const (
	defaultConfigPath = "~/.config/planetscale"
	projectConfigName = ".pscale.yml"
	configName        = "pscale.yml"
	keyringService    = "pscale"
	keyringKey        = "access-token"
	keyringLabel      = "PlanetScale CLI Access Token"
	tokenFileMode     = 0o600
)

// Config is dynamically sourced from various files and environment variables.
type Config struct {
	AccessToken  string
	BaseURL      string
	Organization string

	ServiceTokenID string
	ServiceToken   string

	// Project Configuration
	Database string
	Branch   string
}

func New() (*Config, error) {
	accessToken, err := readAccessToken()
	if err != nil {
		return nil, err
	}

	return &Config{
		AccessToken: accessToken,
		BaseURL:     ps.DefaultBaseURL,
	}, nil
}

func (c *Config) IsAuthenticated() error {
	if (c.ServiceToken == "" && c.ServiceTokenID != "") || (c.ServiceToken != "" && c.ServiceTokenID == "") {
		return errors.New("both --service-token and --service-token-id are required for service token authentication")
	}

	if c.ServiceTokenIsSwapped() {
		return errors.New("the --service-token and --service-token-id values are swapped")
	}

	if c.ServiceTokenIsSet() {
		return nil
	}

	if c.AccessToken == "" {
		return errors.New("you must run 'pscale auth login' to authenticate before using this command")
	}

	return nil
}

func (c *Config) ServiceTokenIsSet() bool {
	return c.ServiceToken != "" && c.ServiceTokenID != ""
}

func (c *Config) ServiceTokenIsSwapped() bool {
	return strings.HasPrefix(c.ServiceTokenID, "pscale_tkn_") && len(c.ServiceToken) == 12
}

// NewClientFromConfig creates a PlanetScale API client from our configuration
func (c *Config) NewClientFromConfig(clientOpts ...ps.ClientOption) (*ps.Client, error) {
	opts := []ps.ClientOption{
		ps.WithBaseURL(c.BaseURL),
	}

	if (c.ServiceToken == "" && c.ServiceTokenID != "") || (c.ServiceToken != "" && c.ServiceTokenID == "") {
		return nil, errors.New("both --service-token and --service-token-id are required for service token authentication")
	}

	if c.ServiceTokenIsSwapped() {
		fmt.Println("The --service-token and --service-token-id are currently swapped.")
		var correctServiceTokenID = c.ServiceToken
		var correctServiceToken = c.ServiceTokenID

		c.ServiceTokenID = correctServiceTokenID
		c.ServiceToken = correctServiceToken
	}

	if c.ServiceTokenIsSet() {
		opts = append(opts, ps.WithServiceToken(c.ServiceTokenID, c.ServiceToken))
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

// ProjectConfigPath returns the path of a configuration inside a Git
// repository.
func ProjectConfigPath() (string, error) {
	basePath, err := RootGitRepoDir()
	if err == nil {
		return filepath.Join(basePath, projectConfigName), nil
	}
	return filepath.Join("", projectConfigName), nil
}

// LocalDir returns the path within the current working directory
func LocalDir() (string, error) {
	localDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return localDir, nil
}

func RootGitRepoDir() (string, error) {
	tl := []string{"rev-parse", "--show-toplevel"}
	out, err := exec.Command("git", tl...).CombinedOutput()
	if err != nil {
		return "", errors.New("unable to find git root directory")
	}

	return strings.TrimSuffix(string(out), "\n"), nil
}

func ProjectConfigFile() string {
	return projectConfigName
}

func readAccessToken() (string, error) {
	ring, err := openKeyring()

	if errors.Is(err, keyring.ErrNoAvailImpl) {
		accessToken, tokenErr := readAccessTokenPath()
		// Token not existing means we're not authenticated.
		if os.IsNotExist(tokenErr) {
			return "", nil
		}
		return string(accessToken), tokenErr
	}

	item, err := ring.Get(keyringKey)
	if err == nil {
		// We're shipping this first without removing the
		// existing token file. Once we're confident that
		// the keyring works well, we can remove the token
		// from disk here.
		//path, err := accessTokenPath()
		//if err != nil {
		//	return "", err
		//}
		//err = os.Remove(path)
		//if err != nil {
		//	return "", err
		//}

		return string(item.Data), nil
	}

	if errors.Is(err, keyring.ErrKeyNotFound) {
		// Migrate to keychain
		accessToken, tokenErr := readAccessTokenPath()
		if len(accessToken) > 0 && tokenErr == nil {
			return migrateAccessToken(ring, accessToken)
		}
		// Might need to improve this, but today the empty
		// token value represents no auth known, and we should
		// not error here since that breaks if you're not logged
		// in yet.
		return "", nil
	}

	return "", err
}

func migrateAccessToken(ring keyring.Keyring, accessToken []byte) (string, error) {
	err := ring.Set(keyring.Item{
		Key:   keyringKey,
		Data:  accessToken,
		Label: keyringLabel,
	})
	if err != nil {
		return "", err
	}

	path, err := accessTokenPath()
	if err == nil {
		fmt.Fprintf(os.Stderr, "Your access token has been migrated to your keyring.\n"+
			"In a future version we will remove the existing token located at: \n\n%s\n\n"+
			"If you want, you can manually delete the token file to complete the migration.\n\n", path)
	}

	return string(accessToken), nil
}

func WriteAccessToken(accessToken string) error {
	ring, err := openKeyring()

	if errors.Is(err, keyring.ErrNoAvailImpl) {
		return writeAccessTokenPath(accessToken)
	}

	return ring.Set(keyring.Item{
		Key:   keyringKey,
		Data:  []byte(accessToken),
		Label: keyringLabel,
	})
}

func DeleteAccessToken() error {
	ring, err := openKeyring()

	if errors.Is(err, keyring.ErrNoAvailImpl) {
		return deleteAccessTokenPath()
	}

	return ring.Remove(keyringKey)
}

func openKeyring() (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{
		AllowedBackends: []keyring.BackendType{
			keyring.SecretServiceBackend,
			keyring.KWalletBackend,
			keyring.KeychainBackend,
			keyring.WinCredBackend,
		},
		ServiceName:              keyringService,
		KeychainTrustApplication: true,
		KeychainSynchronizable:   true,
	})
}

func accessTokenPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, keyringKey), nil
}

func readAccessTokenPath() ([]byte, error) {
	var accessToken []byte
	tokenPath, err := accessTokenPath()
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(tokenPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatal(err)
		}
		return nil, err
	} else {
		if stat.Mode()&^tokenFileMode != 0 {
			err = os.Chmod(tokenPath, tokenFileMode)
			if err != nil {
				log.Printf("Unable to change %v file mode to 0%o: %v", tokenPath, tokenFileMode, err)
			}
		}
		accessToken, err = os.ReadFile(tokenPath)
		if err != nil {
			log.Fatal(err)
		}
	}

	return accessToken, nil
}

func deleteAccessTokenPath() error {
	tokenPath, err := accessTokenPath()
	if err != nil {
		return err
	}

	err = os.Remove(tokenPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, "error removing access token file")
		}
	}

	configFile, err := DefaultConfigPath()
	if err != nil {
		return err
	}

	err = os.Remove(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, "error removing default config file")
		}
	}
	return nil
}

func writeAccessTokenPath(accessToken string) error {
	tokenPath, err := accessTokenPath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(tokenPath)

	_, err = os.Stat(configDir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(configDir, 0o771)
		if err != nil {
			return errors.New("error creating config directory")
		}
	} else if err != nil {
		return err
	}

	tokenBytes := []byte(accessToken)
	err = os.WriteFile(tokenPath, tokenBytes, tokenFileMode)
	if err != nil {
		return errors.Wrap(err, "error writing token")
	}

	return nil
}
