package d1

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	execabs "golang.org/x/sys/execabs"
)

// ExportOptions configures D1 export via wrangler.
type ExportOptions struct {
	D1Database string
	Output     string
	Remote     bool
	Table      string
	NoData     bool
}

// Export runs wrangler d1 export.
func Export(ctx context.Context, opts ExportOptions) (*ExportResult, error) {
	if opts.D1Database == "" {
		return nil, newMigrationError(ErrCodeInvalidInput, "d1 database name is required", "Pass --d1-database")
	}
	if opts.Output == "" {
		opts.Output = fmt.Sprintf("d1-export-%s.sql", opts.D1Database)
	}

	bin, prefix, err := FindWrangler()
	if err != nil {
		return nil, err
	}

	args := append(prefix, "d1", "export", opts.D1Database, "--output", opts.Output)
	if opts.Remote {
		args = append(args, "--remote")
	}
	if opts.Table != "" {
		args = append(args, "--table="+opts.Table)
	}
	if opts.NoData {
		args = append(args, "--no-data=true")
	}

	cmd := execabs.CommandContext(ctx, bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return nil, newMigrationError(
			ErrCodeImportFailed,
			fmt.Sprintf("wrangler export failed: %v", err),
			"Ensure CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID are set, or "+wranglerMissingRemediation,
		)
	}

	size, _ := FileSize(opts.Output)
	return &ExportResult{
		OutputPath: opts.Output,
		Remote:     opts.Remote,
		Database:   opts.D1Database,
		ExportedAt: time.Now().UTC(),
		SizeBytes:  size,
	}, nil
}

// DefaultSQLitePath returns a sqlite path adjacent to the dump.
func DefaultSQLitePath(dumpPath string) string {
	base := filepath.Base(dumpPath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	return filepath.Join(filepath.Dir(dumpPath), name+".sqlite")
}
