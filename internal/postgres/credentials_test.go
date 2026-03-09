package postgres

import (
	"os"
	"testing"
)

func TestStoreAndRetrieveImportCredentials(t *testing.T) {
	// Enable test mode to use file backend
	os.Setenv("PSCALE_TEST_MODE", "1")
	defer os.Unsetenv("PSCALE_TEST_MODE")

	creds, err := NewImportCredentials()
	if err != nil {
		t.Fatalf("NewImportCredentials() error = %v", err)
	}

	org := "testorg"
	db := "testdb"
	branch := "main"
	subscription := "test_sub_123"

	// Test data
	info := &ImportInfo{
		SourceConnStr:    "postgresql://user:pass@localhost:5432/db",
		RoleID:           "role_abc123",
		RoleUsername:     "pscale_user",
		RolePassword:     "secret_password",
		RoleHost:         "db.example.com",
		PublicationName:  "test_publication",
		SubscriptionName: subscription,
		DBName:           "postgres",
	}

	// Store the import info
	err = creds.StoreImportForSubscription(org, db, branch, subscription, info)
	if err != nil {
		t.Fatalf("StoreImportForSubscription() error = %v", err)
	}

	// Retrieve the import info
	retrieved, err := creds.GetImportInfoForSubscription(org, db, branch, subscription)
	if err != nil {
		t.Fatalf("GetImportInfoForSubscription() error = %v", err)
	}

	// Verify all fields match
	if retrieved.SourceConnStr != info.SourceConnStr {
		t.Errorf("SourceConnStr = %v, want %v", retrieved.SourceConnStr, info.SourceConnStr)
	}
	if retrieved.RoleID != info.RoleID {
		t.Errorf("RoleID = %v, want %v", retrieved.RoleID, info.RoleID)
	}
	if retrieved.RoleUsername != info.RoleUsername {
		t.Errorf("RoleUsername = %v, want %v", retrieved.RoleUsername, info.RoleUsername)
	}
	if retrieved.RolePassword != info.RolePassword {
		t.Errorf("RolePassword = %v, want %v", retrieved.RolePassword, info.RolePassword)
	}
	if retrieved.RoleHost != info.RoleHost {
		t.Errorf("RoleHost = %v, want %v", retrieved.RoleHost, info.RoleHost)
	}
	if retrieved.PublicationName != info.PublicationName {
		t.Errorf("PublicationName = %v, want %v", retrieved.PublicationName, info.PublicationName)
	}
	if retrieved.DBName != info.DBName {
		t.Errorf("DBName = %v, want %v", retrieved.DBName, info.DBName)
	}

	// Clean up
	err = creds.ClearSubscriptionCredentials(org, db, branch, subscription)
	if err != nil {
		t.Errorf("ClearSubscriptionCredentials() error = %v", err)
	}
}

func TestListStoredSubscriptions(t *testing.T) {
	// Enable test mode to use file backend
	os.Setenv("PSCALE_TEST_MODE", "1")
	defer os.Unsetenv("PSCALE_TEST_MODE")

	creds, err := NewImportCredentials()
	if err != nil {
		t.Fatalf("NewImportCredentials() error = %v", err)
	}

	org := "testorg"
	db := "testdb"
	branch := "main"

	// Store multiple subscriptions
	subscriptions := []string{"sub1", "sub2", "sub3"}
	for _, sub := range subscriptions {
		info := &ImportInfo{
			SourceConnStr:    "postgresql://localhost:5432/db",
			RoleID:           "role_" + sub,
			SubscriptionName: sub,
		}
		err = creds.StoreImportForSubscription(org, db, branch, sub, info)
		if err != nil {
			t.Fatalf("StoreImportForSubscription() error = %v", err)
		}
	}

	// List stored subscriptions
	stored, err := creds.ListStoredSubscriptions(org, db, branch)
	if err != nil {
		t.Fatalf("ListStoredSubscriptions() error = %v", err)
	}

	if len(stored) != len(subscriptions) {
		t.Errorf("ListStoredSubscriptions() returned %d subscriptions, want %d", len(stored), len(subscriptions))
	}

	// Verify all subscriptions are present
	subMap := make(map[string]bool)
	for _, sub := range stored {
		subMap[sub] = true
	}
	for _, sub := range subscriptions {
		if !subMap[sub] {
			t.Errorf("Subscription %s not found in list", sub)
		}
	}

	// Clean up
	for _, sub := range subscriptions {
		creds.ClearSubscriptionCredentials(org, db, branch, sub)
	}
}

func TestClearSubscriptionCredentials(t *testing.T) {
	// Enable test mode to use file backend
	os.Setenv("PSCALE_TEST_MODE", "1")
	defer os.Unsetenv("PSCALE_TEST_MODE")

	creds, err := NewImportCredentials()
	if err != nil {
		t.Fatalf("NewImportCredentials() error = %v", err)
	}

	org := "testorg"
	db := "testdb"
	branch := "main"
	subscription := "test_sub"

	// Store credentials with all fields
	info := &ImportInfo{
		SourceConnStr:    "postgresql://localhost:5432/db",
		RoleID:           "role123",
		RoleUsername:     "user",
		RolePassword:     "pass",
		RoleHost:         "host",
		PublicationName:  "pub",
		SubscriptionName: subscription,
		DBName:           "postgres",
	}
	err = creds.StoreImportForSubscription(org, db, branch, subscription, info)
	if err != nil {
		t.Fatalf("StoreImportForSubscription() error = %v", err)
	}

	// Clear credentials (ignore errors about missing files)
	_ = creds.ClearSubscriptionCredentials(org, db, branch, subscription)

	// Verify credentials are gone
	_, err = creds.GetImportInfoForSubscription(org, db, branch, subscription)
	if err == nil {
		t.Error("Expected error when retrieving cleared credentials, got nil")
	}
}
