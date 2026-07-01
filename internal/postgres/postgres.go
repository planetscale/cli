// Package postgres provides PostgreSQL connection utilities.
package postgres

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
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

// BuildConnectionURI returns a postgresql:// URI suitable for pgloader.
func BuildConnectionURI(cfg *Config) string {
	host := cfg.Host
	if cfg.Port != 0 {
		host = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	}

	u := &url.URL{
		Scheme: "postgresql",
		Host:   host,
		Path:   "/" + cfg.Database,
	}

	if cfg.User != "" {
		if cfg.Password != "" {
			u.User = url.UserPassword(cfg.User, cfg.Password)
		} else {
			u.User = url.User(cfg.User)
		}
	}

	q := url.Values{}
	if cfg.SSLMode != "" {
		q.Set("sslmode", cfg.SSLMode)
	}
	for key, value := range cfg.Options {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

func quoteValue(s string) string {
	if strings.ContainsAny(s, " '\"\\") {
		return "'" + strings.ReplaceAll(s, "'", "\\'") + "'"
	}
	return s
}

// OpenConnection opens a PostgreSQL connection with sensible defaults.
func OpenConnection(connStr string) (*sql.DB, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// QuoteIdentifier escapes a PostgreSQL identifier.
func QuoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
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

	return redactKeywordPassword(connStr)
}

func redactKeywordPassword(connStr string) string {
	var parts []string
	i := 0
	for i < len(connStr) {
		for i < len(connStr) && (connStr[i] == ' ' || connStr[i] == '\t') {
			i++
		}
		if i >= len(connStr) {
			break
		}
		keyStart := i
		for i < len(connStr) && connStr[i] != '=' && connStr[i] != ' ' && connStr[i] != '\t' {
			i++
		}
		key := connStr[keyStart:i]
		if i >= len(connStr) || connStr[i] != '=' {
			parts = append(parts, strings.TrimSpace(connStr[keyStart:]))
			break
		}
		i++
		if strings.EqualFold(key, "password") {
			_, next := readPasswordConnValue(connStr, i)
			i = next
			parts = append(parts, key+"=****")
			continue
		}
		val, next := readKeywordConnValue(connStr, i)
		i = next
		parts = append(parts, key+"="+val)
	}
	return strings.Join(parts, " ")
}

var connParamKeywords = []string{
	"host", "hostaddr", "port", "user", "password", "dbname", "database",
	"sslmode", "application_name", "connect_timeout", "options",
	"fallback_application_name", "client_encoding", "target_session_attrs",
	"replication", "gssencmode", "sslcert", "sslkey", "sslrootcert",
	"requirepeer", "krbsrvname", "gsslib", "service",
}

func readPasswordConnValue(connStr string, start int) (value string, next int) {
	if start >= len(connStr) {
		return "", start
	}
	if connStr[start] == '\'' || connStr[start] == '"' {
		return readKeywordConnValue(connStr, start)
	}
	end := start + nextConnKeywordAssignIndex(connStr[start:])
	return strings.TrimSpace(connStr[start:end]), end
}

func nextConnKeywordAssignIndex(s string) int {
	if s == "" {
		return 0
	}
	lower := strings.ToLower(s)
	best := len(s)
	for _, kw := range connParamKeywords {
		token := " " + kw + "="
		if idx := strings.Index(lower, token); idx >= 0 && idx < best {
			best = idx
		}
	}
	return best
}

func readKeywordConnValue(connStr string, start int) (value string, next int) {
	if start >= len(connStr) {
		return "", start
	}
	switch connStr[start] {
	case '\'':
		var b strings.Builder
		i := start + 1
		for i < len(connStr) {
			if connStr[i] == '\'' {
				if i+1 < len(connStr) && connStr[i+1] == '\'' {
					b.WriteByte('\'')
					i += 2
					continue
				}
				return b.String(), i + 1
			}
			b.WriteByte(connStr[i])
			i++
		}
		return b.String(), len(connStr)
	case '"':
		var b strings.Builder
		i := start + 1
		for i < len(connStr) {
			if connStr[i] == '"' {
				if i+1 < len(connStr) && connStr[i+1] == '"' {
					b.WriteByte('"')
					i += 2
					continue
				}
				return b.String(), i + 1
			}
			b.WriteByte(connStr[i])
			i++
		}
		return b.String(), len(connStr)
	default:
		i := start
		for i < len(connStr) && connStr[i] != ' ' && connStr[i] != '\t' {
			i++
		}
		return connStr[start:i], i
	}
}
