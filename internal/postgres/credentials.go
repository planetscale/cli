package postgres

import (
	"errors"
	"fmt"
	"os"

	"github.com/99designs/keyring"
)

const (
	keyringService = "pscale"
)

// CredentialType represents the type of credential being stored.
type CredentialType string

const (
	// CredentialTypeSource is for source database connection strings.
	CredentialTypeSource CredentialType = "source"
	// CredentialTypeRoleID is for the replication role ID.
	CredentialTypeRoleID CredentialType = "role_id"
	// CredentialTypeRoleUsername is for the replication role username.
	CredentialTypeRoleUsername CredentialType = "role_username"
	// CredentialTypeRolePassword is for the replication role password.
	CredentialTypeRolePassword CredentialType = "role_password"
	// CredentialTypeRoleHost is for the replication role host.
	CredentialTypeRoleHost CredentialType = "role_host"
	// CredentialTypePublication is for the publication name.
	CredentialTypePublication CredentialType = "publication"
	// CredentialTypeSubscription is for the subscription name.
	CredentialTypeSubscription CredentialType = "subscription"
	// CredentialTypeDBName is for the PostgreSQL database name on the destination.
	CredentialTypeDBName CredentialType = "dbname"
)

// ImportCredentials manages import-related credentials in the keyring.
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

// keyFor generates a keyring key for the given parameters.
func keyFor(org, db, branch string, credType CredentialType) string {
	return fmt.Sprintf("import:%s:%s:%s:%s", org, db, branch, credType)
}

// Store stores a credential in the keyring.
func (c *ImportCredentials) Store(org, db, branch string, credType CredentialType, value string) error {
	key := keyFor(org, db, branch, credType)
	return c.ring.Set(keyring.Item{
		Key:   key,
		Data:  []byte(value),
		Label: fmt.Sprintf("PlanetScale Import - %s/%s/%s - %s", org, db, branch, credType),
	})
}

// Retrieve retrieves a credential from the keyring.
func (c *ImportCredentials) Retrieve(org, db, branch string, credType CredentialType) (string, error) {
	key := keyFor(org, db, branch, credType)
	item, err := c.ring.Get(key)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return "", fmt.Errorf("credential not found for %s/%s/%s (%s)", org, db, branch, credType)
		}
		return "", fmt.Errorf("failed to retrieve credential: %w", err)
	}
	return string(item.Data), nil
}

// Delete deletes a credential from the keyring.
func (c *ImportCredentials) Delete(org, db, branch string, credType CredentialType) error {
	key := keyFor(org, db, branch, credType)
	err := c.ring.Remove(key)
	if err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return fmt.Errorf("failed to delete credential: %w", err)
	}
	return nil
}

// StoreSourceCredentials stores the source database connection string.
func (c *ImportCredentials) StoreSourceCredentials(org, db, branch, connStr string) error {
	return c.Store(org, db, branch, CredentialTypeSource, connStr)
}

// RetrieveSourceCredentials retrieves the source database connection string.
func (c *ImportCredentials) RetrieveSourceCredentials(org, db, branch string) (string, error) {
	return c.Retrieve(org, db, branch, CredentialTypeSource)
}

// StoreRoleID stores the replication role ID.
func (c *ImportCredentials) StoreRoleID(org, db, branch, roleID string) error {
	return c.Store(org, db, branch, CredentialTypeRoleID, roleID)
}

// RetrieveRoleID retrieves the replication role ID.
func (c *ImportCredentials) RetrieveRoleID(org, db, branch string) (string, error) {
	return c.Retrieve(org, db, branch, CredentialTypeRoleID)
}

// StoreRoleCredentials stores the role username, password, and host.
func (c *ImportCredentials) StoreRoleCredentials(org, db, branch, username, password, host string) error {
	if err := c.Store(org, db, branch, CredentialTypeRoleUsername, username); err != nil {
		return err
	}
	if err := c.Store(org, db, branch, CredentialTypeRolePassword, password); err != nil {
		return err
	}
	return c.Store(org, db, branch, CredentialTypeRoleHost, host)
}

// RetrieveRoleCredentials retrieves the role username, password, and host.
func (c *ImportCredentials) RetrieveRoleCredentials(org, db, branch string) (username, password, host string, err error) {
	username, err = c.Retrieve(org, db, branch, CredentialTypeRoleUsername)
	if err != nil {
		return "", "", "", err
	}
	password, err = c.Retrieve(org, db, branch, CredentialTypeRolePassword)
	if err != nil {
		return "", "", "", err
	}
	host, err = c.Retrieve(org, db, branch, CredentialTypeRoleHost)
	if err != nil {
		return "", "", "", err
	}
	return username, password, host, nil
}

// StorePublicationName stores the publication name.
func (c *ImportCredentials) StorePublicationName(org, db, branch, pubName string) error {
	return c.Store(org, db, branch, CredentialTypePublication, pubName)
}

// RetrievePublicationName retrieves the publication name.
func (c *ImportCredentials) RetrievePublicationName(org, db, branch string) (string, error) {
	return c.Retrieve(org, db, branch, CredentialTypePublication)
}

// StoreSubscriptionName stores the subscription name.
func (c *ImportCredentials) StoreSubscriptionName(org, db, branch, subName string) error {
	return c.Store(org, db, branch, CredentialTypeSubscription, subName)
}

// RetrieveSubscriptionName retrieves the subscription name.
func (c *ImportCredentials) RetrieveSubscriptionName(org, db, branch string) (string, error) {
	return c.Retrieve(org, db, branch, CredentialTypeSubscription)
}

// StoreDBName stores the destination database name.
func (c *ImportCredentials) StoreDBName(org, db, branch, destDB string) error {
	return c.Store(org, db, branch, CredentialTypeDBName, destDB)
}

// RetrieveDBName retrieves the destination database name.
func (c *ImportCredentials) RetrieveDBName(org, db, branch string) (string, error) {
	return c.Retrieve(org, db, branch, CredentialTypeDBName)
}

// ClearImportCredentials clears all import credentials for a branch.
func (c *ImportCredentials) ClearImportCredentials(org, db, branch string) error {
	types := []CredentialType{
		CredentialTypeSource,
		CredentialTypeRoleID,
		CredentialTypePublication,
		CredentialTypeSubscription,
		CredentialTypeDBName,
	}

	var errs []error
	for _, credType := range types {
		if err := c.Delete(org, db, branch, credType); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to clear some credentials: %v", errs)
	}
	return nil
}

// HasImportCredentials checks if import credentials exist for a branch.
func (c *ImportCredentials) HasImportCredentials(org, db, branch string) bool {
	_, err := c.Retrieve(org, db, branch, CredentialTypeSource)
	return err == nil
}

// GetAllImportInfo retrieves all stored import information for a branch.
type ImportInfo struct {
	SourceConnStr    string
	RoleID           string
	PublicationName  string
	SubscriptionName string
	DBName           string
}

// GetImportInfo retrieves all import information for a branch.
func (c *ImportCredentials) GetImportInfo(org, db, branch string) (*ImportInfo, error) {
	info := &ImportInfo{}

	var err error
	info.SourceConnStr, err = c.RetrieveSourceCredentials(org, db, branch)
	if err != nil {
		return nil, err
	}

	// These are optional, don't error if not found
	info.RoleID, _ = c.RetrieveRoleID(org, db, branch)
	info.PublicationName, _ = c.RetrievePublicationName(org, db, branch)
	info.SubscriptionName, _ = c.RetrieveSubscriptionName(org, db, branch)
	info.DBName, _ = c.RetrieveDBName(org, db, branch)

	// Default to postgres if not stored
	if info.DBName == "" {
		info.DBName = "postgres"
	}

	return info, nil
}
