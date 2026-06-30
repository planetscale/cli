package d1

import "testing"

const pgloaderOrganizationsOK = `
2026-06-29T17:07:37.780572-04:00 LOG report summary reset
             table name     errors       rows      bytes      total time
-----------------------  ---------  ---------  ---------  --------------
                  fetch          0          0                     0.000s
        fetch meta data          0          1                     0.021s
               Truncate          0          1                     0.053s
          organizations          0         28     2.7 kB          0.775s
      Total import time          ✓         28     2.7 kB          1.770s
`

const pgloaderTeamMembersZero = `
2026-06-29T17:10:50.659297-04:00 LOG report summary reset
             table name     errors       rows      bytes      total time
-----------------------  ---------  ---------  ---------  --------------
                  fetch          0          0                     0.000s
        fetch meta data          0          0                     0.012s
      Total import time          ✓          0                     0.966s
`

func TestPgloaderFetchMetaDataTableCount(t *testing.T) {
	if got := pgloaderFetchMetaDataTableCount(pgloaderOrganizationsOK); got != 1 {
		t.Fatalf("organizations meta = %d, want 1", got)
	}
	if got := pgloaderFetchMetaDataTableCount(pgloaderTeamMembersZero); got != 0 {
		t.Fatalf("team_members meta = %d, want 0", got)
	}
	if got := pgloaderFetchMetaDataTableCount("no summary"); got != -1 {
		t.Fatalf("missing meta = %d, want -1", got)
	}
}

func TestPgloaderRowsCopied(t *testing.T) {
	rows, ok := pgloaderRowsCopied(pgloaderOrganizationsOK, "organizations")
	if !ok || rows != 28 {
		t.Fatalf("organizations rows = (%d, %v), want (28, true)", rows, ok)
	}
	rows, ok = pgloaderRowsCopied(pgloaderTeamMembersZero, "team_members")
	if ok || rows != 0 {
		t.Fatalf("team_members rows = (%d, %v), want (0, false)", rows, ok)
	}
}

func TestValidatePgloaderTableLoad(t *testing.T) {
	if err := validatePgloaderTableLoad(pgloaderOrganizationsOK, "organizations", 28); err != nil {
		t.Fatalf("expected ok load: %v", err)
	}
	if err := validatePgloaderTableLoad(pgloaderOrganizationsOK, "organizations", 0); err != nil {
		t.Fatalf("expected skip for empty expected: %v", err)
	}
	if err := validatePgloaderTableLoad(pgloaderTeamMembersZero, "team_members", 700); err == nil {
		t.Fatal("expected error for 0-row load")
	} else if me, ok := err.(*MigrationError); !ok || me.Info.Code != ErrCodeImportFailed {
		t.Fatalf("error = %#v", err)
	}
}
