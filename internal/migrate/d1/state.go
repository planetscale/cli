package d1

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
)

// StateStore manages local migration state.
type StateStore struct {
	dir string
}

// NewStateStore returns the default state store location.
func NewStateStore() (*StateStore, error) {
	dir, err := xdg.ConfigFile("planetscale/import-d1")
	if err != nil {
		return nil, fmt.Errorf("state dir: %w", err)
	}
	if os.Getenv("PSCALE_TEST_MODE") == "1" {
		dir = filepath.Join(os.TempDir(), "pscale-import-d1-test")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &StateStore{dir: dir}, nil
}

func (s *StateStore) statePath(org, database, branch, migrationID string) string {
	key := fmt.Sprintf("%s_%s_%s_%s.json", sanitize(org), sanitize(database), sanitize(branch), sanitize(migrationID))
	return filepath.Join(s.dir, key)
}

func sanitize(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			out = append(out, c)
		} else {
			out = append(out, '_')
		}
	}
	return string(out)
}

// Save persists migration state.
func (s *StateStore) Save(state *MigrationState) error {
	if state.CreatedAt.IsZero() {
		state.CreatedAt = time.Now().UTC()
	}
	state.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path := s.statePath(state.Org, state.Database, state.Branch, state.MigrationID)
	return os.WriteFile(path, data, 0o600)
}

// Load retrieves migration state by ID.
func (s *StateStore) Load(org, database, branch, migrationID string) (*MigrationState, error) {
	path := s.statePath(org, database, branch, migrationID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, newMigrationError(ErrCodeNotFound, "migration state not found", "Run `import d1 start --dry-run` or `import d1 start` to create migration state")
		}
		return nil, err
	}
	var state MigrationState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// Delete removes migration state.
func (s *StateStore) Delete(org, database, branch, migrationID string) error {
	path := s.statePath(org, database, branch, migrationID)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// SaveState is a package-level helper using the default store.
func SaveState(state *MigrationState) error {
	store, err := NewStateStore()
	if err != nil {
		return err
	}
	return store.Save(state)
}

// LoadState loads state using the default store.
func LoadState(org, database, branch, migrationID string) (*MigrationState, error) {
	store, err := NewStateStore()
	if err != nil {
		return nil, err
	}
	return store.Load(org, database, branch, migrationID)
}

// SetMigrationPhase updates the phase on existing migration state.
func SetMigrationPhase(org, database, branch, migrationID, phase string) error {
	return updateMigrationState(org, database, branch, migrationID, func(state *MigrationState) {
		state.Phase = phase
	})
}

func updateMigrationState(org, database, branch, migrationID string, update func(*MigrationState)) error {
	state, err := LoadState(org, database, branch, migrationID)
	if err != nil {
		return err
	}
	update(state)
	return SaveState(state)
}

func saveImportMigrationState(opts ImportOptions, phase, sqlitePath string) error {
	state, err := LoadState(opts.Org, opts.Database, opts.Branch, opts.MigrationID)
	if err != nil {
		if me, ok := migrationErr(err); ok && me.Info.Code == ErrCodeNotFound {
			state = &MigrationState{
				MigrationID: opts.MigrationID,
				Org:         opts.Org,
				Database:    opts.Database,
				Branch:      opts.Branch,
			}
		} else {
			return err
		}
	}
	state.Phase = phase
	if opts.InputPath != "" {
		state.InputPath = opts.InputPath
	}
	if opts.Method != "" {
		state.Method = opts.Method
	}
	if sqlitePath != "" {
		state.SQLitePath = sqlitePath
	}
	return SaveState(state)
}

// Complete marks a migration as finished in local state.
func Complete(org, database, branch, migrationID string) error {
	return SetMigrationPhase(org, database, branch, migrationID, PhaseComplete)
}

// Teardown is deprecated; use Complete.
func Teardown(org, database, branch, migrationID string) error {
	return Complete(org, database, branch, migrationID)
}
