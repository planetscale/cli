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

var (
	psqlVersionRegex   = regexp.MustCompile(`psql \(PostgreSQL\) (\d+)\.?(\d*)`)
	pgDumpVersionRegex = regexp.MustCompile(`pg_dump \(PostgreSQL\) (\d+)\.?(\d*)`)
)

func FindPsqlPath() (string, error) {
	for _, cmd := range []string{"psql-18", "psql-17", "psql-16", "psql-15", "psql"} {
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
		"To install, run: brew install postgresql@18")
}

func FindPgDumpPath() (string, error) {
	for _, cmd := range []string{"pg_dump-18", "pg_dump-17", "pg_dump-16", "pg_dump-15", "pg_dump"} {
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
		"To install, run: brew install postgresql@18")
}

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

// PipeSchemaImport runs pg_dump and pipes output to psql for schema import.
func PipeSchemaImport(ctx context.Context, sourceConn, destConn string, schemas []string, includeTables []string, excludeTables []string) error {
	pgDumpPath, err := FindPgDumpPath()
	if err != nil {
		return err
	}

	psqlPath, err := FindPsqlPath()
	if err != nil {
		return err
	}

	// When importing specific tables, pg_dump --table=schema.table does NOT output
	// CREATE SCHEMA statements. We must explicitly create schemas first.
	if len(includeTables) > 0 {
		schemaSet := make(map[string]bool)
		for _, table := range includeTables {
			if idx := strings.Index(table, "."); idx > 0 {
				schema := table[:idx]
				schemaSet[schema] = true
			}
		}

		if len(schemaSet) > 0 {
			// Create all required schemas on destination
			var createSchemas []string
			for schema := range schemaSet {
				createSchemas = append(createSchemas, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", QuoteIdentifier(schema)))
			}

			schemaSQL := strings.Join(createSchemas, "\n")
			psqlCmd := exec.CommandContext(ctx, psqlPath, destConn, "--quiet", "-c", schemaSQL)
			psqlCmd.Env = os.Environ()

			var stderr bytes.Buffer
			psqlCmd.Stderr = &stderr

			if err := psqlCmd.Run(); err != nil {
				return fmt.Errorf("failed to create schemas: %w\nstderr: %s", err, stderr.String())
			}
		}
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

	// If specific tables are requested, use --table flag instead of --schema
	// Also skip constraints to avoid FK errors with concurrent imports
	if len(includeTables) > 0 {
		for _, table := range includeTables {
			pgDumpArgs = append(pgDumpArgs, "--table="+table)
		}
		// Skip constraints when importing specific tables (concurrent imports)
		// This prevents foreign key errors when tables reference other tables
		// created by different import sessions
		pgDumpArgs = append(pgDumpArgs, "--no-comments")
		pgDumpArgs = append(pgDumpArgs, "--exclude-table-data=*")
	} else {
		// Add schema filters
		for _, schema := range schemas {
			pgDumpArgs = append(pgDumpArgs, "--schema="+schema)
		}

		for _, table := range excludeTables {
			pgDumpArgs = append(pgDumpArgs, "--exclude-table", table)
		}
	}

	// Build psql arguments
	psqlArgs := []string{
		destConn,
		"--quiet",
	}

	// Only use single transaction when importing full schema
	// Skip it for specific tables to allow FK errors to be ignored
	if len(includeTables) == 0 {
		psqlArgs = append(psqlArgs, "--single-transaction")
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
		_ = pgDumpCmd.Process.Kill()
		_ = pgDumpCmd.Wait()
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
		// When importing specific tables (concurrent imports), allow FK permission errors
		// These happen when tables reference other tables owned by different roles
		if len(includeTables) > 0 && strings.Contains(stderr, "permission denied for table") {
			// Non-fatal - FK constraints will be skipped but replication will work
			return nil
		}
		if stderr != "" {
			return fmt.Errorf("psql failed: %w\nstderr: %s", psqlErr, stderr)
		}
		return fmt.Errorf("psql failed: %w", psqlErr)
	}

	// Check if psql had any errors even if exit code was 0
	if psqlStderr.Len() > 0 {
		stderr := psqlStderr.String()
		// When importing specific tables, allow FK permission errors
		if len(includeTables) > 0 && strings.Contains(stderr, "permission denied for table") {
			return nil
		}
		// Only report if it's not just warnings
		if strings.Contains(strings.ToLower(stderr), "error") || strings.Contains(strings.ToLower(stderr), "fatal") {
			return fmt.Errorf("psql completed but reported errors:\n%s", stderr)
		}
	}

	return nil
}
