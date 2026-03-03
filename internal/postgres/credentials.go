package postgres

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/99designs/keyring"
)

const (
	keyringService = "pscale"
)

// credType represents the type of credential being stored.
type credType string

const (
	credSource       credType = "source"
	credRoleID       credType = "role_id"
	credRoleUsername credType = "role_username"
	credRolePassword credType = "role_password"
	credRoleHost     credType = "role_host"
	credPublication  credType = "publication"
	credSubscription credType = "subscription"
	credDBName       credType = "dbname"
)

// ImportCredentials manages import credentials in the keyring.
type ImportCredentials struct {
	ring keyring.Keyring
}

// NewImportCredentials creates a new ImportCredentials manager.
func NewImportCredentials() (*ImportCredentials, error) {
	ring, err := openKeyring()
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}
	return &ImportCredentials{ring: ring}, nil
}

func openKeyring() (keyring.Keyring, error) {
	// Use file backend in test mode to avoid keychain prompts
	if os.Getenv("PSCALE_TEST_MODE") == "1" {
		return keyring.Open(keyring.Config{
			AllowedBackends:          []keyring.BackendType{keyring.FileBackend},
			ServiceName:              keyringService,
			FileDir:                  os.TempDir() + "/pscale-test-keyring",
			FilePasswordFunc:         keyring.FixedStringPrompt("test"),
			KeychainTrustApplication: true,
		})
	}

	ring, err := keyring.Open(keyring.Config{
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

	// Fall back to encrypted file storage if no secure backend available
	if errors.Is(err, keyring.ErrNoAvailImpl) {
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", homeErr)
		}

		return keyring.Open(keyring.Config{
			AllowedBackends:  []keyring.BackendType{keyring.FileBackend},
			ServiceName:      keyringService,
			FileDir:          homeDir + "/.config/planetscale/import-credentials",
			FilePasswordFunc: keyring.TerminalPrompt,
		})
	}

	return ring, err
}

// keyForSubscription generates a subscription-scoped keyring key.
func keyForSubscription(org, db, branch, subscription string, ct credType) string {
	return fmt.Sprintf("import:%s:%s:%s:%s:%s", org, db, branch, subscription, ct)
}

// ImportInfo contains all stored credentials for an import.
type ImportInfo struct {
	SourceConnStr    string
	RoleID           string
	RoleUsername     string
	RolePassword     string
	RoleHost         string
	PublicationName  string
	SubscriptionName string
	DBName           string
}

// StoreImportForSubscription stores all credentials for a subscription.
func (c *ImportCredentials) StoreImportForSubscription(org, db, branch, subscription string, info *ImportInfo) error {
	key := func(ct credType) string {
		return keyForSubscription(org, db, branch, subscription, ct)
	}

	items := []struct {
		ct    credType
		value string
	}{
		{credSource, info.SourceConnStr},
		{credRoleID, info.RoleID},
		{credRoleUsername, info.RoleUsername},
		{credRolePassword, info.RolePassword},
		{credRoleHost, info.RoleHost},
		{credPublication, info.PublicationName},
		{credSubscription, subscription},
		{credDBName, info.DBName},
	}

	for _, item := range items {
		if item.value == "" {
			continue
		}
		if err := c.ring.Set(keyring.Item{
			Key:   key(item.ct),
			Data:  []byte(item.value),
			Label: fmt.Sprintf("PlanetScale Import - %s/%s/%s/%s - %s", org, db, branch, subscription, item.ct),
		}); err != nil {
			return fmt.Errorf("failed to store %s: %w", item.ct, err)
		}
	}

	return nil
}

// GetImportInfoForSubscription retrieves import info for a subscription.
func (c *ImportCredentials) GetImportInfoForSubscription(org, db, branch, subscription string) (*ImportInfo, error) {
	key := func(ct credType) string {
		return keyForSubscription(org, db, branch, subscription, ct)
	}

	retrieve := func(ct credType) (string, error) {
		item, err := c.ring.Get(key(ct))
		if err != nil {
			if errors.Is(err, keyring.ErrKeyNotFound) {
				return "", nil
			}
			return "", err
		}
		return string(item.Data), nil
	}

	info := &ImportInfo{}
	var err error

	info.SourceConnStr, err = retrieve(credSource)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve source: %w", err)
	}
	if info.SourceConnStr == "" {
		return nil, fmt.Errorf("no import found for subscription %s", subscription)
	}

	info.RoleID, _ = retrieve(credRoleID)
	info.RoleUsername, _ = retrieve(credRoleUsername)
	info.RolePassword, _ = retrieve(credRolePassword)
	info.RoleHost, _ = retrieve(credRoleHost)
	info.PublicationName, _ = retrieve(credPublication)
	info.SubscriptionName, _ = retrieve(credSubscription)
	info.DBName, _ = retrieve(credDBName)

	if info.DBName == "" {
		info.DBName = "postgres"
	}

	return info, nil
}

// ListStoredSubscriptions lists all subscriptions with stored credentials for a branch.
func (c *ImportCredentials) ListStoredSubscriptions(org, db, branch string) ([]string, error) {
	prefix := fmt.Sprintf("import:%s:%s:%s:", org, db, branch)

	keys, err := c.ring.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	subsMap := make(map[string]bool)
	for _, key := range keys {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		// Key format: import:org:db:branch:subscription:credType
		parts := strings.Split(key, ":")
		if len(parts) >= 6 {
			sub := parts[4]
			subsMap[sub] = true
		}
	}

	var subs []string
	for sub := range subsMap {
		subs = append(subs, sub)
	}

	return subs, nil
}

// ClearSubscriptionCredentials clears all credentials for a subscription.
func (c *ImportCredentials) ClearSubscriptionCredentials(org, db, branch, subscription string) error {
	types := []credType{
		credSource,
		credRoleID,
		credRoleUsername,
		credRolePassword,
		credRoleHost,
		credPublication,
		credSubscription,
		credDBName,
	}

	var errs []error
	for _, ct := range types {
		key := keyForSubscription(org, db, branch, subscription, ct)
		if err := c.ring.Remove(key); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to clear some credentials: %v", errs)
	}
	return nil
}
