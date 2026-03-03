// Package postgres provides PostgreSQL connection utilities.
package postgres

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
	Options  map[string]string
}

// ParseConnectionURI supports both URI and keyword/value formats.
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

	if portStr := u.Port(); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid port: %w", err)
		}
		cfg.Port = port
	}

	if u.User != nil {
		cfg.User = u.User.Username()
		cfg.Password, _ = u.User.Password()
	}

	// Database name from path without leading /
	cfg.Database = strings.TrimPrefix(u.Path, "/")

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

func quoteValue(s string) string {
	if strings.ContainsAny(s, " '\"\\") {
		return "'" + strings.ReplaceAll(s, "'", "\\'") + "'"
	}
	return s
}

// OpenConnection opens a PostgreSQL connection with sensible defaults.
func OpenConnection(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

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
