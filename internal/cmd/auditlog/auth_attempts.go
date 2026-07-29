package auditlog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

var authAttemptExportPollInterval = 2 * time.Second

// AuthAttemptsCmd wraps commands for authentication-attempt exports.
func AuthAttemptsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth-attempts <command>",
		Short: "Download authentication-attempt export reports",
	}
	cmd.AddCommand(DownloadAuthAttemptsCmd(ch))
	return cmd
}

// AuthAttemptExportDownload is the result of downloading an authentication-attempt export.
type AuthAttemptExportDownload struct {
	ID      string `header:"id" json:"id"`
	State   string `header:"state" json:"state"`
	Format  string `header:"format" json:"format"`
	StartAt string `header:"start_at" json:"start_at"`
	EndAt   string `header:"end_at" json:"end_at"`
	File    string `header:"file" json:"file"`
}

func (a *AuthAttemptExportDownload) MarshalCSVValue() interface{} {
	return []*AuthAttemptExportDownload{a}
}

// DownloadAuthAttemptsCmd generates an authentication-attempt export, waits for it to finish, and downloads the ZIP file.
func DownloadAuthAttemptsCmd(ch *cmdutil.Helper) *cobra.Command {
	flags := downloadAuthAttemptsOptions{}

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download an authentication-attempt export",
		Long:  "Generate an authentication-attempt export for a UTC time window, wait for it to be ready, and download its ZIP artifact. Use --since with a positive duration (24h, 7d, or 2w), or use --start-at and optionally --end-at. Named values and zone-less timestamps use the local timezone; requests are sent in UTC.",
		Example: `  # Export the last 24 hours.
	  pscale audit-log auth-attempts download --since 24h

	  # Export from local midnight yesterday through now.
	  pscale audit-log auth-attempts download --start-at yesterday

	  # Export a date-only window.
	  pscale audit-log auth-attempts download --start-at 2026-07-28 --end-at 2026-07-29

	  # Zone-less local ISO timestamps can use T or a space.
	  pscale audit-log auth-attempts download --start-at '2026-07-29 16:00' --end-at '2026-07-29 17:00'

	  # RFC3339 timestamps may include any explicit offset.
	  pscale audit-log auth-attempts download \
	    --start-at 2026-07-29T00:00:00Z --end-at 2026-07-29T01:00:00Z

	  # Export denied attempts from two addresses during a UTC window.
	  pscale audit-log auth-attempts download \
	    --start-at 2026-07-29T00:00:00Z --end-at 2026-07-29T01:00:00Z \
	    --source-ip 203.0.113.0/24 --source-ip 2001:db8::/32`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDownloadAuthAttempts(cmd, ch, flags)
		},
	}

	cmd.Flags().StringVar(&flags.since, "since", "", "Export window ending now: positive duration such as 24h, 7d, or 2w")
	cmd.Flags().StringVar(&flags.startAt, "start-at", "", "Inclusive start: now, today, yesterday, YYYY-MM-DD, local ISO, or RFC3339")
	cmd.Flags().StringVar(&flags.endAt, "end-at", "", "Exclusive end: now, today, yesterday, YYYY-MM-DD, local ISO, or RFC3339; defaults to now")
	cmd.Flags().StringSliceVar(&flags.sourceIPs, "source-ip", nil, "Source IP address or CIDR range (repeat or comma-separate)")
	cmd.Flags().StringSliceVar(&flags.branches, "branch", nil, "Branch public ID or database/branch name (repeat or comma-separate)")
	cmd.Flags().StringSliceVar(&flags.outcomes, "outcome", nil, fmt.Sprintf("Authentication outcome: %s", strings.Join(authAttemptExportOutcomes, " or ")))
	cmd.Flags().StringArrayVar(&flags.usernames, "username", nil, "Authentication username (repeatable; commas are part of the name, not separators)")
	cmd.Flags().StringArrayVar(&flags.startupDatabases, "startup-database", nil, "Startup database name (repeatable)")
	cmd.Flags().StringSliceVar(&flags.failureReasons, "failure-reason", nil, fmt.Sprintf("Failure reason: %s", strings.Join(authAttemptExportFailureReasons, ", ")))
	cmd.Flags().StringSliceVar(&flags.backendRoutes, "backend-route", nil, fmt.Sprintf("Backend route: %s", strings.Join(authAttemptExportBackendRoutes, ", ")))
	cmd.Flags().StringVar(&flags.exportFormat, "export-format", "", fmt.Sprintf("Artifact format: %s; defaults from --format (human/csv -> csv, json -> JSONL; parquet is explicit)", strings.Join(authAttemptExportFormats, ", ")))
	cmd.Flags().StringVar(&flags.output, "output", "", "Output file name, or - to write raw ZIP bytes to stdout")

	for _, enum := range []struct {
		name   string
		values []string
	}{
		{name: "outcome", values: authAttemptExportOutcomes},
		{name: "failure-reason", values: authAttemptExportFailureReasons},
		{name: "backend-route", values: authAttemptExportBackendRoutes},
		{name: "export-format", values: authAttemptExportFormats},
	} {
		_ = cmd.RegisterFlagCompletionFunc(enum.name, cobra.FixedCompletions(enum.values, cobra.ShellCompDirectiveNoFileComp))
	}
	cmd.MarkFlagsMutuallyExclusive("since", "start-at")
	cmd.MarkFlagsMutuallyExclusive("since", "end-at")

	return cmd
}

func runDownloadAuthAttempts(cmd *cobra.Command, ch *cmdutil.Helper, flags downloadAuthAttemptsOptions) error {
	if cmd.Flags().Changed("since") && flags.since == "" {
		return fmt.Errorf("--since cannot be empty; use a positive duration such as 24h, 7d, or 2w")
	}
	if cmd.Flags().Changed("start-at") && flags.startAt == "" {
		return fmt.Errorf("--start-at cannot be empty; use now, today, yesterday, YYYY-MM-DD, local ISO, or RFC3339")
	}
	if cmd.Flags().Changed("end-at") && flags.endAt == "" {
		return fmt.Errorf("--end-at cannot be empty; use now, today, yesterday, YYYY-MM-DD, local ISO, or RFC3339")
	}

	startAt, endAt, err := resolveAuthAttemptExportWindow(flags, time.Now(), time.Local)
	if err != nil {
		return err
	}

	filters, err := authAttemptExportFilters(flags.sourceIPs, flags.branches, flags.outcomes,
		flags.usernames, flags.startupDatabases, flags.failureReasons, flags.backendRoutes, map[string]bool{
			"source-ip":      cmd.Flags().Changed("source-ip"),
			"branch":         cmd.Flags().Changed("branch"),
			"outcome":        cmd.Flags().Changed("outcome"),
			"username":       cmd.Flags().Changed("username"),
			"failure-reason": cmd.Flags().Changed("failure-reason"),
			"backend-route":  cmd.Flags().Changed("backend-route"),
		})
	if err != nil {
		return err
	}
	format, err := authAttemptExportFormat(cmd, flags.exportFormat, ch.Printer.Format())
	if err != nil {
		return err
	}

	client, err := ch.Client()
	if err != nil {
		return err
	}

	toStdout := flags.output == "-"
	endProgress := func() {}
	if !toStdout {
		endProgress = ch.Printer.PrintProgress(fmt.Sprintf("Generating authentication-attempt export for %s...",
			printer.BoldBlue(ch.Config.Organization)))
	}
	defer endProgress()

	report, exportID, err := createAndPollAuthAttemptExport(cmd.Context(), client, ch.Config.Organization,
		startAt, endAt, format, filters)
	if err != nil {
		return err
	}
	if report.State == "failed" {
		return authAttemptExportFailureError(report, exportID)
	}
	if report.State != "ready" {
		return fmt.Errorf("auth attempt export %s reached unexpected state %q", printer.BoldBlue(exportID), report.State)
	}

	path := authAttemptExportOutputPath(flags.output, ch.Config.Organization, format)
	if err := downloadAndPrintAuthAttemptExport(cmd, ch, client, exportID, format, path,
		startAt, endAt, toStdout, report.State, endProgress); err != nil {
		return err
	}
	return nil
}

func createAndPollAuthAttemptExport(ctx context.Context, client *ps.Client, organization string, startAt, endAt time.Time, format string, filters ps.AuthAttemptExportFilters) (*ps.AuthAttemptExport, string, error) {
	report, err := client.AuthAttemptExports.CreateExport(ctx, &ps.CreateAuthAttemptExportRequest{
		Organization: organization,
		StartAt:      startAt,
		EndAt:        endAt,
		Format:       format,
		Filters:      filters,
	})
	if err != nil {
		if cmdutil.ErrCode(err) == ps.ErrNotFound {
			return nil, "", authAttemptExportNotFoundError(organization, "")
		}
		return nil, "", cmdutil.HandleError(err)
	}
	exportID := report.PublicID
	for report.State == "pending" || report.State == "running" {
		interval := report.RetryAfter
		if interval <= 0 {
			interval = authAttemptExportPollInterval
		}
		if err := waitForAuthAttemptExport(ctx, interval); err != nil {
			return nil, exportID, authAttemptExportInterrupted(organization, exportID, err)
		}
		report, err = client.AuthAttemptExports.GetExport(ctx, &ps.GetAuthAttemptExportRequest{
			Organization: organization, Export: exportID,
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil, exportID, authAttemptExportInterrupted(organization, exportID, ctx.Err())
			}
			if cmdutil.ErrCode(err) == ps.ErrNotFound {
				return nil, exportID, authAttemptExportNotFoundError(organization, exportID)
			}
			return nil, exportID, cmdutil.HandleError(err)
		}
	}
	return report, exportID, nil
}

func downloadAndPrintAuthAttemptExport(cmd *cobra.Command, ch *cmdutil.Helper, client *ps.Client, exportID, format, path string, startAt, endAt time.Time, toStdout bool, state string, endProgress func()) error {
	body, err := client.AuthAttemptExports.DownloadExport(cmd.Context(), &ps.DownloadAuthAttemptExportRequest{
		Organization: ch.Config.Organization,
		Export:       exportID,
	})
	if err != nil {
		if cmd.Context().Err() != nil {
			return authAttemptExportInterrupted(ch.Config.Organization, exportID, cmd.Context().Err())
		}
		if cmdutil.ErrCode(err) == ps.ErrNotFound {
			return authAttemptExportNotFoundError(ch.Config.Organization, exportID)
		}
		return cmdutil.HandleError(err)
	}

	if toStdout {
		defer body.Close()
		if _, err := io.Copy(cmd.OutOrStdout(), body); err != nil {
			if cmd.Context().Err() != nil {
				return authAttemptExportInterrupted(ch.Config.Organization, exportID, cmd.Context().Err())
			}
			return fmt.Errorf("writing auth attempt export to stdout: %w", err)
		}
		return nil
	}
	if err := publishAuthAttemptExport(cmd.Context(), body, path); err != nil {
		if cmd.Context().Err() != nil {
			return authAttemptExportInterrupted(ch.Config.Organization, exportID, cmd.Context().Err())
		}
		return fmt.Errorf("writing auth attempt export to %s: %w", path, err)
	}

	endProgress()
	if ch.Printer.Format() == printer.Human {
		ch.Printer.Printf("Successfully downloaded auth attempts (%s) to %s\n", format, printer.BoldBlue(path))
		return nil
	}
	return ch.Printer.PrintResource(&AuthAttemptExportDownload{
		ID: exportID, State: state, Format: format,
		StartAt: startAt.Format(time.RFC3339), EndAt: endAt.Format(time.RFC3339), File: path,
	})
}

func waitForAuthAttemptExport(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func authAttemptExportOutputPath(output, organization, format string) string {
	filename := fmt.Sprintf("auth-attempts-%s-%s-%s.zip", organization, format, time.Now().UTC().Format("20060102T150405Z"))
	if output == "" {
		return filename
	}
	return output
}

func publishAuthAttemptExport(ctx context.Context, body io.ReadCloser, path string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		body.Close()
		return fmt.Errorf("creating temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	_, copyErr := io.Copy(tmp, body)
	bodyCloseErr := body.Close()
	closeErr := tmp.Close()
	if copyErr == nil {
		copyErr = bodyCloseErr
	}
	if copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return copyErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming temporary file: %w", err)
	}
	return nil
}

func authAttemptExportInterrupted(organization, exportID string, err error) error {
	return fmt.Errorf("auth attempt export %s interrupted: %w; recover with pscale api organizations/%s/auth-attempt-exports/%s",
		printer.BoldBlue(exportID), err, organization, exportID)
}

func authAttemptExportFailureError(report *ps.AuthAttemptExport, exportID string) error {
	message := fmt.Sprintf("auth attempt export %s failed (%s)", printer.BoldBlue(exportID), report.FailureReason)
	if report.FailureDetail != "" {
		message += ": " + report.FailureDetail
	}
	if report.RecoveryHint != "" {
		message += "\n" + report.RecoveryHint
	}
	return errors.New(message)
}

func authAttemptExportNotFoundError(organization, exportID string) error {
	if exportID == "" {
		return fmt.Errorf("auth-attempt exports are unavailable for organization %s", printer.BoldBlue(organization))
	}
	return fmt.Errorf("auth-attempt export %s was not found and may have expired", printer.BoldBlue(exportID))
}
