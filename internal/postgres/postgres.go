// Package postgres provides PostgreSQL connection utilities.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// Config holds PostgreSQL connection parameters.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
	Options  map[string]string
}

// ParseConnectionURI parses a PostgreSQL connection URI.
// Supports URI and keyword/value formats.
func ParseConnectionURI(uri string) (*Config, error) {
	// Handle postgresql:// or postgres:// URIs
	if strings.HasPrefix(uri, "postgresql://") || strings.HasPrefix(uri, "postgres://") {
		return parseURIFormat(uri)
	}

	// Handle keyword/value format (host=localhost port=5432 ...)
	return parseKeyValueFormat(uri)
}

func parseURIFormat(uri string) (*Config, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid connection URI: %w", err)
	}

	cfg := &Config{
		Host:    u.Hostname(),
		Port:    5432,
		Options: make(map[string]string),
	}

	// Parse port if present
	if portStr := u.Port(); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid port: %w", err)
		}
		cfg.Port = port
	}

	// Parse user info
	if u.User != nil {
		cfg.User = u.User.Username()
		cfg.Password, _ = u.User.Password()
	}

	// Parse database name (path without leading /)
	cfg.Database = strings.TrimPrefix(u.Path, "/")

	// Parse query parameters
	for key, values := range u.Query() {
		if len(values) > 0 {
			switch key {
			case "sslmode":
				cfg.SSLMode = values[0]
			default:
				cfg.Options[key] = values[0]
			}
		}
	}

	if cfg.SSLMode == "" {
		cfg.SSLMode = "require"
	}

	return cfg, nil
}

func parseKeyValueFormat(connStr string) (*Config, error) {
	cfg := &Config{
		Port:    5432,
		SSLMode: "require",
		Options: make(map[string]string),
	}

	for _, pair := range strings.Fields(connStr) {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := strings.Trim(parts[1], "'\"")

		switch key {
		case "host":
			cfg.Host = value
		case "port":
			port, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid port: %w", err)
			}
			cfg.Port = port
		case "user":
			cfg.User = value
		case "password":
			cfg.Password = value
		case "dbname":
			cfg.Database = value
		case "sslmode":
			cfg.SSLMode = value
		default:
			cfg.Options[key] = value
		}
	}

	return cfg, nil
}

// BuildConnectionString builds a connection string from cfg.
func BuildConnectionString(cfg *Config) string {
	var parts []string

	if cfg.Host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", cfg.Host))
	}
	if cfg.Port != 0 {
		parts = append(parts, fmt.Sprintf("port=%d", cfg.Port))
	}
	if cfg.User != "" {
		parts = append(parts, fmt.Sprintf("user=%s", cfg.User))
	}
	if cfg.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", quoteValue(cfg.Password)))
	}
	if cfg.Database != "" {
		parts = append(parts, fmt.Sprintf("dbname=%s", cfg.Database))
	}
	if cfg.SSLMode != "" {
		parts = append(parts, fmt.Sprintf("sslmode=%s", cfg.SSLMode))
	}

	for key, value := range cfg.Options {
		parts = append(parts, fmt.Sprintf("%s=%s", key, quoteValue(value)))
	}

	return strings.Join(parts, " ")
}

// BuildConnectionURL builds a connection URL from cfg.
func BuildConnectionURL(cfg *Config) string {
	u := url.URL{
		Scheme: "postgresql",
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:   "/" + cfg.Database,
	}

	if cfg.User != "" {
		if cfg.Password != "" {
			u.User = url.UserPassword(cfg.User, cfg.Password)
		} else {
			u.User = url.User(cfg.User)
		}
	}

	q := u.Query()
	if cfg.SSLMode != "" {
		q.Set("sslmode", cfg.SSLMode)
	}
	for key, value := range cfg.Options {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

// quoteValue quotes s if needed.
func quoteValue(s string) string {
	if strings.ContainsAny(s, " '\"\\") {
		return "'" + strings.ReplaceAll(s, "'", "\\'") + "'"
	}
	return s
}

// TestConnection tests a PostgreSQL connection.
func TestConnection(ctx context.Context, connStr string, timeout time.Duration) error {
	if timeout > 0 {
		connStr = fmt.Sprintf("%s connect_timeout=%d", connStr, int(timeout.Seconds()))
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// RedactPassword redacts passwords in connStr.
func RedactPassword(connStr string) string {
	if strings.HasPrefix(connStr, "postgresql://") || strings.HasPrefix(connStr, "postgres://") {
		u, err := url.Parse(connStr)
		if err == nil && u.User != nil {
			if _, hasPass := u.User.Password(); hasPass {
				u.User = url.UserPassword(u.User.Username(), "****")
				return u.String()
			}
		}
		return connStr
	}

	var result []string
	for _, pair := range strings.Fields(connStr) {
		if strings.HasPrefix(pair, "password=") {
			result = append(result, "password=****")
		} else {
			result = append(result, pair)
		}
	}
	return strings.Join(result, " ")
}

// MergeConfig merges base and override, with override taking precedence.
func MergeConfig(base, override *Config) *Config {
	result := &Config{
		Host:     base.Host,
		Port:     base.Port,
		User:     base.User,
		Password: base.Password,
		Database: base.Database,
		SSLMode:  base.SSLMode,
		Options:  make(map[string]string),
	}

	for k, v := range base.Options {
		result.Options[k] = v
	}

	if override.Host != "" {
		result.Host = override.Host
	}
	if override.Port != 0 {
		result.Port = override.Port
	}
	if override.User != "" {
		result.User = override.User
	}
	if override.Password != "" {
		result.Password = override.Password
	}
	if override.Database != "" {
		result.Database = override.Database
	}
	if override.SSLMode != "" {
		result.SSLMode = override.SSLMode
	}
	for k, v := range override.Options {
		result.Options[k] = v
	}

	return result
}
