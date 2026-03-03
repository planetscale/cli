package postgres

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	exec "golang.org/x/sys/execabs"
)

// PsqlOptions contains options for executing psql commands.
type PsqlOptions struct {
	// ConnString is the PostgreSQL connection string
	ConnString string
	// Query is the SQL query to execute (mutually exclusive with File)
	Query string
	// File is the path to a SQL file to execute (mutually exclusive with Query)
	File string
	// Variables are psql variables to set (--set name=value)
	Variables map[string]string
	// SingleTransaction wraps commands in a single transaction
	SingleTransaction bool
	// Quiet suppresses output messages
	Quiet bool
}

// PgDumpOptions contains options for executing pg_dump commands.
type PgDumpOptions struct {
	// ConnString is the PostgreSQL connection string
	ConnString string
	// SchemaOnly dumps only the schema, not data
	SchemaOnly bool
	// DataOnly dumps only the data, not schema
	DataOnly bool
	// Tables is a list of tables to dump (empty means all)
	Tables []string
	// ExcludeTables is a list of tables to exclude
	ExcludeTables []string
	// Schemas is a list of schemas to dump (empty means all)
	Schemas []string
	// ExcludeSchemas is a list of schemas to exclude
	ExcludeSchemas []string
	// NoOwner omits owner statements
	NoOwner bool
	// NoPrivileges omits privilege statements
	NoPrivileges bool
	// NoTablespaces omits tablespace assignments
	NoTablespaces bool
	// NoComments omits comments
	NoComments bool
	// NoPublications omits publications
	NoPublications bool
	// NoSubscriptions omits subscriptions
	NoSubscriptions bool
	// Format is the output format (plain, custom, directory, tar)
	Format string
	// IfExists adds IF EXISTS to DROP statements
	IfExists bool
	// Clean adds DROP statements before CREATE
	Clean bool
	// Create adds CREATE DATABASE statement
	Create bool
}

var (
	psqlVersionRegex   = regexp.MustCompile(`psql \(PostgreSQL\) (\d+)\.?(\d*)`)
	pgDumpVersionRegex = regexp.MustCompile(`pg_dump \(PostgreSQL\) (\d+)\.?(\d*)`)
)

// FindPsqlPath locates the psql binary.
func FindPsqlPath() (string, error) {
	for _, cmd := range []string{"psql-17", "psql-16", "psql-15", "psql"} {
		path, err := exec.LookPath(cmd)
		if err == nil {
			cmd := exec.Command(path, "--version")
			out, err := cmd.Output()
			if err != nil {
				continue
			}

			// Check if it's PostgreSQL (not just any psql)
			if strings.Contains(string(out), "PostgreSQL") {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("couldn't find the 'psql' command-line tool required for PostgreSQL imports.\n" +
		"To install, run: brew install postgresql@17")
}

// FindPgDumpPath locates the pg_dump binary.
func FindPgDumpPath() (string, error) {
	for _, cmd := range []string{"pg_dump-17", "pg_dump-16", "pg_dump-15", "pg_dump"} {
		path, err := exec.LookPath(cmd)
		if err == nil {
			// Verify it's a working pg_dump
			cmd := exec.Command(path, "--version")
			out, err := cmd.Output()
			if err != nil {
				continue
			}

			if strings.Contains(string(out), "pg_dump") {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("couldn't find the 'pg_dump' command-line tool required for PostgreSQL imports.\n" +
		"To install, run: brew install postgresql@17")
}

// CheckPsqlVersion returns the psql version.
func CheckPsqlVersion(minMajor int) (major, minor int, err error) {
	path, err := FindPsqlPath()
	if err != nil {
		return 0, 0, err
	}

	cmd := exec.Command(path, "--version")
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get psql version: %w", err)
	}

	matches := psqlVersionRegex.FindStringSubmatch(string(out))
	if len(matches) < 2 {
		return 0, 0, fmt.Errorf("could not parse psql version from: %s", string(out))
	}

	major, err = strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse psql major version: %w", err)
	}

	if len(matches) > 2 && matches[2] != "" {
		minor, _ = strconv.Atoi(matches[2])
	}

	if major < minMajor {
		return major, minor, fmt.Errorf("psql version %d.%d is too old, minimum required is %d", major, minor, minMajor)
	}

	return major, minor, nil
}

// CheckPgDumpVersion returns the pg_dump version.
func CheckPgDumpVersion(minMajor int) (major, minor int, err error) {
	path, err := FindPgDumpPath()
	if err != nil {
		return 0, 0, err
	}

	cmd := exec.Command(path, "--version")
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get pg_dump version: %w", err)
	}

	matches := pgDumpVersionRegex.FindStringSubmatch(string(out))
	if len(matches) < 2 {
		return 0, 0, fmt.Errorf("could not parse pg_dump version from: %s", string(out))
	}

	major, err = strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse pg_dump major version: %w", err)
	}

	if len(matches) > 2 && matches[2] != "" {
		minor, _ = strconv.Atoi(matches[2])
	}

	if major < minMajor {
		return major, minor, fmt.Errorf("pg_dump version %d.%d is too old, minimum required is %d", major, minor, minMajor)
	}

	return major, minor, nil
}

// ExecutePsql executes a psql command and returns stdout and stderr.
func ExecutePsql(ctx context.Context, opts PsqlOptions) (stdout, stderr string, err error) {
	path, err := FindPsqlPath()
	if err != nil {
		return "", "", err
	}

	args := []string{opts.ConnString}

	if opts.SingleTransaction {
		args = append(args, "--single-transaction")
	}

	if opts.Quiet {
		args = append(args, "--quiet")
	}

	for name, value := range opts.Variables {
		args = append(args, "--set", fmt.Sprintf("%s=%s", name, value))
	}

	if opts.Query != "" {
		args = append(args, "-c", opts.Query)
	} else if opts.File != "" {
		args = append(args, "-f", opts.File)
	}

	cmd := exec.CommandContext(ctx, path, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	cmd.Env = os.Environ()

	err = cmd.Run()
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	if err != nil {
		return stdout, stderr, fmt.Errorf("psql error: %w\nstderr: %s", err, stderr)
	}

	return stdout, stderr, nil
}

// ExecutePgDump executes a pg_dump command and returns stdout and stderr.
func ExecutePgDump(ctx context.Context, opts PgDumpOptions) (stdout, stderr string, err error) {
	path, err := FindPgDumpPath()
	if err != nil {
		return "", "", err
	}

	args := []string{opts.ConnString}

	if opts.SchemaOnly {
		args = append(args, "--schema-only")
	}
	if opts.DataOnly {
		args = append(args, "--data-only")
	}
	if opts.NoOwner {
		args = append(args, "--no-owner")
	}
	if opts.NoPrivileges {
		args = append(args, "--no-privileges")
	}
	if opts.NoTablespaces {
		args = append(args, "--no-tablespaces")
	}
	if opts.NoComments {
		args = append(args, "--no-comments")
	}
	if opts.NoPublications {
		args = append(args, "--no-publications")
	}
	if opts.NoSubscriptions {
		args = append(args, "--no-subscriptions")
	}
	if opts.IfExists {
		args = append(args, "--if-exists")
	}
	if opts.Clean {
		args = append(args, "--clean")
	}
	if opts.Create {
		args = append(args, "--create")
	}
	if opts.Format != "" {
		args = append(args, "--format", opts.Format)
	}

	for _, table := range opts.Tables {
		args = append(args, "--table", table)
	}
	for _, table := range opts.ExcludeTables {
		args = append(args, "--exclude-table", table)
	}
	for _, schema := range opts.Schemas {
		args = append(args, "--schema", schema)
	}
	for _, schema := range opts.ExcludeSchemas {
		args = append(args, "--exclude-schema", schema)
	}

	cmd := exec.CommandContext(ctx, path, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	cmd.Env = os.Environ()

	err = cmd.Run()
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	if err != nil {
		return stdout, stderr, fmt.Errorf("pg_dump error: %w\nstderr: %s", err, stderr)
	}

	return stdout, stderr, nil
}

// PipeSchemaImport runs pg_dump and pipes output to psql for schema import.
func PipeSchemaImport(ctx context.Context, sourceConn, destConn string, schemas []string, excludeTables []string) error {
	pgDumpPath, err := FindPgDumpPath()
	if err != nil {
		return err
	}

	psqlPath, err := FindPsqlPath()
	if err != nil {
		return err
	}

	// Build pg_dump arguments
	pgDumpArgs := []string{
		sourceConn,
		"--schema-only",
		"--no-owner",
		"--no-privileges",
		"--no-tablespaces",
		"--no-publications",
		"--no-subscriptions",
	}

	// Add schema filters
	for _, schema := range schemas {
		pgDumpArgs = append(pgDumpArgs, "--schema="+schema)
	}

	for _, table := range excludeTables {
		pgDumpArgs = append(pgDumpArgs, "--exclude-table", table)
	}

	// Build psql arguments
	psqlArgs := []string{
		destConn,
		"--single-transaction",
		"--quiet",
	}

	// Create the commands
	pgDumpCmd := exec.CommandContext(ctx, pgDumpPath, pgDumpArgs...)
	psqlCmd := exec.CommandContext(ctx, psqlPath, psqlArgs...)

	// Get pg_dump stdout
	pgDumpOut, err := pgDumpCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create pg_dump pipe: %w", err)
	}

	// Get psql stdin
	psqlIn, err := psqlCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create psql pipe: %w", err)
	}

	// Capture stderr from both
	var pgDumpStderr, psqlStderr bytes.Buffer
	pgDumpCmd.Stderr = &pgDumpStderr
	psqlCmd.Stderr = &psqlStderr

	// Set environment
	pgDumpCmd.Env = os.Environ()
	psqlCmd.Env = os.Environ()

	// Start pg_dump
	if err := pgDumpCmd.Start(); err != nil {
		return fmt.Errorf("failed to start pg_dump: %w", err)
	}

	// Start psql
	if err := psqlCmd.Start(); err != nil {
		pgDumpCmd.Process.Kill()
		return fmt.Errorf("failed to start psql: %w", err)
	}

	// Filter pg_dump output and write to psql
	filterErr := make(chan error, 1)
	go func() {
		defer psqlIn.Close()
		scanner := bufio.NewScanner(pgDumpOut)
		for scanner.Scan() {
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)

			// Skip \restrict and \unrestrict lines (Supabase-specific)
			if strings.HasPrefix(trimmed, "\\restrict") || strings.HasPrefix(trimmed, "\\unrestrict") {
				continue
			}

			// Replace CREATE SCHEMA with CREATE SCHEMA IF NOT EXISTS
			if strings.HasPrefix(trimmed, "CREATE SCHEMA ") && strings.HasSuffix(trimmed, ";") {
				line = strings.Replace(line, "CREATE SCHEMA ", "CREATE SCHEMA IF NOT EXISTS ", 1)
			}

			if _, err := fmt.Fprintln(psqlIn, line); err != nil {
				filterErr <- err
				return
			}
		}
		if err := scanner.Err(); err != nil {
			filterErr <- err
			return
		}
		filterErr <- nil
	}()

	// Wait for pg_dump
	pgDumpErr := pgDumpCmd.Wait()

	// Wait for psql
	psqlErr := psqlCmd.Wait()

	// Check filter error
	if err := <-filterErr; err != nil {
		return fmt.Errorf("failed to filter pg_dump output: %w", err)
	}

	// Check for errors with detailed stderr output
	if pgDumpErr != nil {
		stderr := pgDumpStderr.String()
		if stderr != "" {
			return fmt.Errorf("pg_dump failed: %w\nstderr: %s", pgDumpErr, stderr)
		}
		return fmt.Errorf("pg_dump failed: %w", pgDumpErr)
	}
	if psqlErr != nil {
		stderr := psqlStderr.String()
		if stderr != "" {
			return fmt.Errorf("psql failed: %w\nstderr: %s", psqlErr, stderr)
		}
		return fmt.Errorf("psql failed: %w", psqlErr)
	}

	// Check if psql had any errors even if exit code was 0
	if psqlStderr.Len() > 0 {
		stderr := psqlStderr.String()
		// Only report if it's not just warnings
		if strings.Contains(strings.ToLower(stderr), "error") || strings.Contains(strings.ToLower(stderr), "fatal") {
			return fmt.Errorf("psql completed but reported errors:\n%s", stderr)
		}
	}

	return nil
}
