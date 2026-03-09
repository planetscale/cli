package postgres

import (
	"testing"
)

// Note: Testing exported functions only.
// Internal key generation functions are tested indirectly through
// the StoreImportForSubscription and GetImportInfoForSubscription methods.

func TestImportInfo(t *testing.T) {
	// Basic structure test - ensure ImportInfo struct is usable
	info := &ImportInfo{
		SourceConnStr:   "postgresql://localhost:5432/db",
		RoleID:          "role123",
		RoleUsername:    "user",
		RolePassword:    "pass",
		RoleHost:        "host",
		PublicationName: "pub",
		DBName:          "postgres",
	}

	if info.SourceConnStr == "" {
		t.Error("ImportInfo should have SourceConnStr")
	}
	if info.RoleID == "" {
		t.Error("ImportInfo should have RoleID")
	}
}
