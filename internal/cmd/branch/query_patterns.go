package branch

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// queryPatternsPollInterval is how often the download command polls the API
// while a report is generating. It's a variable so tests can shorten it.
var queryPatternsPollInterval = 2 * time.Second

// QueryPatternsCmd wraps commands for a branch's query patterns reports.
func QueryPatternsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-patterns <command>",
		Short: "List, show, delete, and download query pattern reports for a branch",
	}

	cmd.AddCommand(ListQueryPatternsCmd(ch))
	cmd.AddCommand(ShowQueryPatternsCmd(ch))
	cmd.AddCommand(DeleteQueryPatternsCmd(ch))
	cmd.AddCommand(DownloadQueryPatternsCmd(ch))

	return cmd
}

// QueryPatternsDownload is the result of downloading a query patterns report.
type QueryPatternsDownload struct {
	ID    string `header:"id" json:"id"`
	State string `header:"state" json:"state"`
	File  string `header:"file" json:"file"`
}

// DownloadQueryPatternsCmd generates a query patterns report for a branch,
// waits for it to finish, and downloads the resulting CSV file.
func DownloadQueryPatternsCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		output string
	}

	cmd := &cobra.Command{
		Use:   "download <database> <branch>",
		Short: "Download a CSV report of the query patterns for a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]
			toStdout := flags.output == "-"

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := func() {}
			if !toStdout {
				end = ch.Printer.PrintProgress(fmt.Sprintf("Generating query patterns report for %s in %s...",
					printer.BoldBlue(branch), printer.BoldBlue(database)))
			}
			defer end()

			report, err := client.QueryPatterns.CreateReport(ctx, &ps.CreateQueryPatternsReportRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return queryPatternsError(ch, err, database, branch)
			}

			ticker := time.NewTicker(queryPatternsPollInterval)
			defer ticker.Stop()

			for report.State == "pending" {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
					report, err = client.QueryPatterns.GetReport(ctx, &ps.GetQueryPatternsReportRequest{
						Organization: ch.Config.Organization,
						Database:     database,
						Branch:       branch,
						Report:       report.PublicID,
					})
					if err != nil {
						return queryPatternsError(ch, err, database, branch)
					}
				}
			}

			if report.State != "completed" {
				return fmt.Errorf("query patterns report %s for branch %s failed to generate, please try again",
					printer.BoldBlue(report.PublicID), printer.BoldBlue(branch))
			}

			path := flags.output
			if path == "" {
				path = fmt.Sprintf("query-patterns-%s-%s-%s-%s.csv",
					ch.Config.Organization, database, branch,
					time.Now().UTC().Format("20060102T150405Z"))
			}

			body, err := client.QueryPatterns.DownloadReport(ctx, &ps.DownloadQueryPatternsReportRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Report:       report.PublicID,
			})
			if err != nil {
				return queryPatternsError(ch, err, database, branch)
			}
			defer body.Close()

			if toStdout {
				if _, err := io.Copy(cmd.OutOrStdout(), body); err != nil {
					return fmt.Errorf("writing query patterns report to stdout: %w", err)
				}
				return nil
			}

			f, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("creating file %s: %w", path, err)
			}

			_, err = io.Copy(f, body)
			if closeErr := f.Close(); err == nil {
				err = closeErr
			}
			if err != nil {
				return fmt.Errorf("writing query patterns report to %s: %w", path, err)
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Successfully downloaded query patterns report to %s\n", printer.BoldBlue(path))
				return nil
			}

			return ch.Printer.PrintResource(&QueryPatternsDownload{
				ID:    report.PublicID,
				State: report.State,
				File:  path,
			})
		},
	}

	cmd.Flags().StringVar(&flags.output, "output", "",
		"Output file name, or - to write to stdout. Defaults to query-patterns-<organization>-<database>-<branch>-<timestamp>.csv.")

	return cmd
}

func ListQueryPatternsCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		limit         int
		startingAfter string
	}

	cmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Short:   "List query pattern reports for a branch",
		Aliases: []string{"ls"},
		Args:    cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching query pattern reports for %s/%s...",
				printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			opts := []ps.ListOption{ps.WithLimit(flags.limit)}
			if flags.startingAfter != "" {
				opts = append(opts, ps.WithStartingAfter(flags.startingAfter))
			}

			reports, err := client.QueryPatterns.ListReports(ctx, &ps.ListQueryPatternsReportsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			}, opts...)
			if err != nil {
				return queryPatternsError(ch, err, database, branch)
			}
			end()

			if len(reports) == 0 && ch.Printer.Format() == printer.Human {
				if flags.startingAfter != "" {
					ch.Printer.Println("No query pattern reports found on this page.")
				} else {
					ch.Printer.Printf("No query pattern reports exist for %s/%s.\n",
						printer.BoldBlue(database), printer.BoldBlue(branch))
				}
				return nil
			}

			return ch.Printer.PrintResource(toQueryPatternsReports(reports))
		},
	}

	cmd.Flags().IntVar(&flags.limit, "limit", 25, "Number of results to return (max 100)")
	cmd.Flags().StringVar(&flags.startingAfter, "starting-after", "", "Cursor to fetch the next page of reports")
	return cmd
}

func ShowQueryPatternsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <branch> <report-id>",
		Short: "Show a query pattern report",
		Args:  cmdutil.RequiredArgs("database", "branch", "report-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, reportID := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching query pattern report %s for %s/%s...",
				printer.BoldBlue(reportID), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			report, err := client.QueryPatterns.GetReport(ctx, &ps.GetQueryPatternsReportRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Report:       reportID,
			})
			if err != nil {
				return queryPatternsReportError(ch, err, database, branch, reportID)
			}
			end()

			return ch.Printer.PrintResource(toQueryPatternsReport(report))
		},
	}

	return cmd
}

func DeleteQueryPatternsCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <database> <branch> <report-id>",
		Short:   "Delete a query pattern report",
		Aliases: []string{"rm"},
		Args:    cmdutil.RequiredArgs("database", "branch", "report-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, reportID := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !force {
				if err := ch.Printer.ConfirmCommand(reportID, "delete query pattern report", "deletion of query pattern report"); err != nil {
					return err
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting query pattern report %s from %s/%s...",
				printer.BoldBlue(reportID), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			err = client.QueryPatterns.DeleteReport(ctx, &ps.DeleteQueryPatternsReportRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Report:       reportID,
			})
			if err != nil {
				return queryPatternsReportError(ch, err, database, branch, reportID)
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Deleted query pattern report %s from %s/%s.\n",
					printer.BoldBlue(reportID), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result": "query pattern report deleted",
				"id":     reportID,
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete the report without confirmation")
	return cmd
}

type queryPatternsReport struct {
	ID         string `header:"id" json:"id"`
	State      string `header:"state" json:"state"`
	Actor      string `header:"actor" json:"actor"`
	CreatedAt  int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	FinishedAt *int64 `header:"finished_at,timestamp(ms|utc|human)" json:"finished_at"`

	orig *ps.QueryPatternsReport
}

func toQueryPatternsReports(reports []*ps.QueryPatternsReport) []*queryPatternsReport {
	out := make([]*queryPatternsReport, 0, len(reports))
	for _, report := range reports {
		out = append(out, toQueryPatternsReport(report))
	}
	return out
}

func toQueryPatternsReport(report *ps.QueryPatternsReport) *queryPatternsReport {
	actor := "-"
	if report.Actor != nil && report.Actor.Name != "" {
		actor = report.Actor.Name
	}
	out := &queryPatternsReport{
		ID:         report.PublicID,
		State:      report.State,
		Actor:      actor,
		CreatedAt:  printer.GetMilliseconds(report.CreatedAt),
		FinishedAt: printer.GetMillisecondsIfExists(report.FinishedAt),
		orig:       report,
	}
	return out
}

func (r *queryPatternsReport) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(r.orig, "", "  ")
}

func (r *queryPatternsReport) MarshalCSVValue() interface{} {
	return []*queryPatternsReport{r}
}

func queryPatternsReportError(ch *cmdutil.Helper, err error, database, branch, reportID string) error {
	switch cmdutil.ErrCode(err) {
	case ps.ErrNotFound:
		return fmt.Errorf("query pattern report %s does not exist for branch %s in database %s (organization: %s) or query insights is not enabled for the database",
			printer.BoldBlue(reportID), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
	default:
		return cmdutil.HandleError(err)
	}
}

// queryPatternsError maps a not-found API error to a message explaining both
// causes: a missing branch, or query insights being disabled for the database.
func queryPatternsError(ch *cmdutil.Helper, err error, database, branch string) error {
	switch cmdutil.ErrCode(err) {
	case ps.ErrNotFound:
		return fmt.Errorf("branch %s does not exist in database %s (organization: %s) or query insights is not enabled for the database",
			printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
	default:
		return cmdutil.HandleError(err)
	}
}
