package config

import (
	"errors"
	"sync"
	"testing"
)

func TestAccessTokenLookupCached(t *testing.T) {
	lookupErr := errors.New("keychain access denied")
	for _, tt := range []struct {
		name  string
		token string
		err   error
	}{
		{name: "token", token: "stored-token"},
		{name: "empty"},
		{name: "failure", err: lookupErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			cfg := New()
			cfg.loadAccessToken = sync.OnceValues(func() (string, error) {
				calls++
				return tt.token, tt.err
			})
			if calls != 0 {
				t.Fatal("constructor loaded credentials")
			}
			for range 2 {
				if err := cfg.LoadAccessToken(); !errors.Is(err, tt.err) {
					t.Fatalf("LoadAccessToken error = %v, want %v", err, tt.err)
				}
				if _, err := cfg.NewClientFromConfig(); tt.err != nil && !errors.Is(err, tt.err) {
					t.Fatalf("NewClientFromConfig error = %v, want %v", err, tt.err)
				}
				if err := cfg.IsAuthenticated(); tt.err != nil && !errors.Is(err, tt.err) {
					t.Fatalf("IsAuthenticated error = %v, want %v", err, tt.err)
				}
			}
			if calls != 1 {
				t.Fatalf("credential lookups = %d, want 1", calls)
			}
			if cfg.AccessToken != tt.token {
				t.Fatalf("access token = %q, want %q", cfg.AccessToken, tt.token)
			}
		})
	}
}

func TestExplicitCredentialsSkipAccessTokenLookup(t *testing.T) {
	for _, tt := range []struct {
		name, token, serviceID, serviceToken string
		invalid                              bool
	}{
		{name: "api token", token: "explicit-token"},
		{name: "service token", serviceID: "id", serviceToken: "secret"},
		{name: "missing service secret", serviceID: "id", invalid: true},
		{name: "missing service id", serviceToken: "secret", invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := New()
			cfg.AccessToken, cfg.ServiceTokenID, cfg.ServiceToken = tt.token, tt.serviceID, tt.serviceToken
			cfg.loadAccessToken = func() (string, error) {
				t.Fatal("explicit credentials opened the keychain")
				return "", nil
			}
			if err := cfg.LoadAccessToken(); err != nil {
				t.Fatal(err)
			}
			if err := cfg.IsAuthenticated(); (err != nil) != tt.invalid {
				t.Fatalf("IsAuthenticated error = %v, invalid = %v", err, tt.invalid)
			}
			if _, err := cfg.NewClientFromConfig(); (err != nil) != tt.invalid {
				t.Fatalf("NewClientFromConfig error = %v, invalid = %v", err, tt.invalid)
			}
		})
	}
}

func TestStoredAccessTokenLoadsWithServiceToken(t *testing.T) {
	cfg := New()
	cfg.ServiceTokenID, cfg.ServiceToken = "id", "secret"
	cfg.loadAccessToken = func() (string, error) { return "stored-token", nil }
	if err := cfg.LoadAccessToken(); err != nil {
		t.Fatal(err)
	}
	if cfg.AccessToken != "" {
		t.Fatalf("LoadAccessToken loaded %q despite service token", cfg.AccessToken)
	}
	if err := cfg.LoadStoredAccessToken(); err != nil {
		t.Fatal(err)
	}
	if cfg.AccessToken != "stored-token" {
		t.Fatalf("access token = %q, want stored-token", cfg.AccessToken)
	}
}
