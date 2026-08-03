package auditlog

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

var authAttemptExportFormats = []string{"jsonl", "csv", "parquet"}
var authAttemptExportOutcomes = []string{"allow", "deny"}
var authAttemptExportFailureReasons = []string{"bad_password", "unknown_user", "authorization_failed", "ip_not_allowed", "other"}
var authAttemptExportBackendRoutes = []string{"postgres", "pgbouncer", "unknown"}

type downloadAuthAttemptsOptions struct {
	since            string
	startAt          string
	endAt            string
	sourceIPs        []string
	branches         []string
	outcomes         []string
	usernames        []string
	startupDatabases []string
	failureReasons   []string
	backendRoutes    []string
	exportFormat     string
	output           string
}

func authAttemptExportFormat(cmd *cobra.Command, requested string, output printer.Format) (string, error) {
	if requested == "" {
		if cmd.Flags().Changed("export-format") {
			return "", fmt.Errorf("--export-format cannot be empty")
		}
		if output == printer.JSON {
			return "jsonl", nil
		}
		return "csv", nil
	}
	if !slices.Contains(authAttemptExportFormats, requested) {
		return "", fmt.Errorf("invalid --export-format %q, must be one of: %s", requested, strings.Join(authAttemptExportFormats, ", "))
	}
	return requested, nil
}

func resolveAuthAttemptExportWindow(flags downloadAuthAttemptsOptions, now time.Time, location *time.Location) (time.Time, time.Time, error) {
	switch {
	case flags.since != "" && (flags.startAt != "" || flags.endAt != ""):
		return time.Time{}, time.Time{}, fmt.Errorf("--since is mutually exclusive with --start-at and --end-at")
	case flags.since == "" && flags.startAt == "" && flags.endAt == "":
		return time.Time{}, time.Time{}, fmt.Errorf("a time window is required: use --since or --start-at with optional --end-at")
	case flags.since == "" && flags.startAt == "":
		return time.Time{}, time.Time{}, fmt.Errorf("--start-at is required when --since is not set")
	}

	if flags.since != "" {
		duration, err := parseAuthAttemptExportSince(flags.since)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		return now.Add(-duration).UTC(), now.UTC(), nil
	}

	startAt, err := parseAuthAttemptExportTime("start-at", flags.startAt, now, location)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	endAt := now
	if flags.endAt != "" {
		endAt, err = parseAuthAttemptExportTime("end-at", flags.endAt, now, location)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	if !startAt.Before(endAt) {
		return time.Time{}, time.Time{}, fmt.Errorf("--start-at must be before --end-at")
	}
	return startAt.UTC(), endAt.UTC(), nil
}

func parseAuthAttemptExportSince(value string) (time.Duration, error) {
	if value == "" {
		return 0, fmt.Errorf("--since cannot be empty; use a positive duration such as 24h, 7d, or 2w")
	}

	const week = 7 * 24 * time.Hour
	const maxDuration = time.Duration(1<<63 - 1)
	if strings.HasSuffix(value, "w") {
		weeks, err := strconv.ParseInt(value[:len(value)-1], 10, 64)
		if err != nil || weeks <= 0 || weeks > int64(maxDuration/week) {
			return 0, fmt.Errorf("invalid --since %q: use a positive duration such as 24h, 7d, or 2w", value)
		}
		return time.Duration(weeks) * week, nil
	}

	duration, err := cmdutil.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid --since %q: use a positive duration such as 24h, 7d, or 2w", value)
	}
	if strings.HasSuffix(value, "d") {
		days, parseErr := strconv.ParseInt(value[:len(value)-1], 10, 64)
		if parseErr != nil || days <= 0 || days > int64(maxDuration/(24*time.Hour)) {
			return 0, fmt.Errorf("invalid --since %q: use a positive duration such as 24h, 7d, or 2w", value)
		}
	}
	if duration <= 0 {
		return 0, fmt.Errorf("invalid --since %q: use a positive duration such as 24h, 7d, or 2w", value)
	}
	return duration, nil
}

func parseAuthAttemptExportTime(flag, value string, now time.Time, location *time.Location) (time.Time, error) {
	switch strings.ToLower(value) {
	case "now":
		return now.UTC(), nil
	case "today", "yesterday":
		localNow := now.In(location)
		midnight := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location)
		if strings.EqualFold(value, "yesterday") {
			midnight = midnight.AddDate(0, 0, -1)
		}
		return midnight.UTC(), nil
	}

	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), nil
	}
	for _, layout := range []string{"2006-01-02", "2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if parsed, err := time.ParseInLocation(layout, value, location); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid --%s %q: use now, today, yesterday, YYYY-MM-DD, local ISO, or RFC3339", flag, value)
}

func authAttemptExportFilters(cmd *cobra.Command, flags downloadAuthAttemptsOptions) (ps.AuthAttemptExportFilters, error) {
	for _, filter := range []struct {
		name    string
		values  []string
		allowed []string
	}{
		{"source-ip", flags.sourceIPs, nil},
		{"branch", flags.branches, nil},
		{"outcome", flags.outcomes, authAttemptExportOutcomes},
		{"username", flags.usernames, nil},
		{"failure-reason", flags.failureReasons, authAttemptExportFailureReasons},
		{"backend-route", flags.backendRoutes, authAttemptExportBackendRoutes},
	} {
		if err := validateAuthAttemptFilter(filter.name, filter.values, filter.allowed, cmd.Flags().Changed(filter.name)); err != nil {
			return ps.AuthAttemptExportFilters{}, err
		}
	}

	return ps.AuthAttemptExportFilters{
		SourceIPs:        flags.sourceIPs,
		Branches:         flags.branches,
		Outcomes:         flags.outcomes,
		Usernames:        flags.usernames,
		StartupDatabases: flags.startupDatabases,
		FailureReasons:   flags.failureReasons,
		BackendRoutes:    flags.backendRoutes,
	}, nil
}

func validateAuthAttemptFilter(flag string, values, allowed []string, changed bool) error {
	if changed && len(values) == 0 {
		return fmt.Errorf("--%s cannot be empty", flag)
	}
	for _, value := range values {
		if value == "" {
			return fmt.Errorf("--%s cannot be empty", flag)
		}
		if len(allowed) > 0 && !slices.Contains(allowed, value) {
			return fmt.Errorf("invalid --%s %q, must be one of: %s", flag, value, strings.Join(allowed, ", "))
		}
	}
	return nil
}
