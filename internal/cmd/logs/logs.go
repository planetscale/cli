package logs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

const (
	defaultLimit  = 100
	defaultPeriod = "1h"
)

var validPeriods = map[string]bool{
	"15m": true,
	"1h":  true,
	"3h":  true,
	"6h":  true,
	"12h": true,
	"1d":  true,
	"7d":  true,
	"8d":  true,
}

var validLevels = map[string]bool{
	"ERROR":   true,
	"DEBUG":   true,
	"INFO":    true,
	"WARNING": true,
}

type flags struct {
	query   string
	period  string
	from    string
	to      string
	levels  []string
	servers []string
	shards  []string
	limit   int
	page    int
}

// LogEntry is a parsed branch log entry.
type LogEntry struct {
	Time             string `header:"time" json:"time" csv:"time"`
	Level            string `header:"level" json:"level" csv:"level"`
	Message          string `header:"message" json:"message" csv:"message"`
	Role             string `header:"role" json:"role,omitempty" csv:"role"`
	Shard            string `header:"shard" json:"shard,omitempty" csv:"shard"`
	Container        string `header:"container" json:"container" csv:"container"`
	AvailabilityZone string `header:"availability zone" json:"availability_zone" csv:"availability_zone"`
	Pod              string `header:"pod" json:"pod" csv:"pod"`
	StreamID         string `header:"stream id" json:"stream_id" csv:"stream_id"`
	RawMessage       string `header:"raw message" json:"raw_message" csv:"raw_message"`
}

type outerLog struct {
	Time             string `json:"_time"`
	StreamID         string `json:"_stream_id"`
	Message          string `json:"_msg"`
	Level            string `json:"planetscale.level"`
	Role             string `json:"planetscale.role"`
	Shard            string `json:"planetscale.shard"`
	Container        string `json:"planetscale.container"`
	AvailabilityZone string `json:"planetscale.availability_zone"`
	Pod              string `json:"planetscale.pod"`
}

type innerLog struct {
	Message string `json:"message"`
}

// LogsCmd queries logs for a PostgreSQL or Neki branch.
func LogsCmd(ch *cmdutil.Helper) *cobra.Command {
	f := &flags{}

	cmd := &cobra.Command{
		Use:   "logs <database> <branch>",
		Short: "Query logs for a PostgreSQL or Neki branch",
		Long: `Query recent logs for a PostgreSQL or Neki branch.

The command obtains a signed logs URL from the PlanetScale API, builds a LogsQL
query, and prints parsed log entries. By default it returns up to 100 logs from
the primary server over the last hour, newest first.`,
		Example: `  # Recent primary logs
  pscale logs <database> <branch> --org <org> --format json

  # Errors from a selected pod over the last six hours
  pscale logs <database> <branch> --org <org> --format json --period 6h --level ERROR --server <pod>

  # Logs from a custom ISO 8601 time range
  pscale logs <database> <branch> --org <org> --format json \
    --from 2026-08-20T16:00:00Z --to 2026-08-20T18:00:00Z

  # Search and filter with LogsQL syntax
  pscale logs <database> <branch> --org <org> --format json --query 'connection refused'`,
		Args:              cmdutil.RequiredArgs("database", "branch"),
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateFlags(f, cmd.Flags().Changed("period")); err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			database, branch := args[0], args[1]
			signature, err := client.Logs.CreateSignature(ctx, &ps.CreateLogSignatureRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return err
			}

			body, err := client.Logs.Query(ctx, &ps.QueryLogsRequest{
				URL:   signature.URL,
				Query: createQuery(f),
				Limit: f.limit,
			})
			if err != nil {
				return err
			}
			defer body.Close()

			entries, err := parseLogs(body)
			if err != nil {
				return err
			}

			if len(entries) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No logs found for %s in %s.\n", printer.BoldBlue(branch), printer.BoldBlue(database))
				return nil
			}
			if ch.Printer.Format() == printer.Human {
				printHumanLogs(ch, entries)
				return nil
			}

			return ch.Printer.PrintResource(entries)
		},
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization,
		"The organization for the current user")
	cmd.Flags().StringVar(&f.query, "query", "", "Text or LogsQL expression to search for")
	cmd.Flags().StringVar(&f.period, "period", defaultPeriod,
		"Time period to query: 15m, 1h, 3h, 6h, 12h, 1d, 7d, or 8d")
	cmd.Flags().StringVar(&f.from, "from", "", "Start of a custom time range as an ISO 8601 timestamp")
	cmd.Flags().StringVar(&f.to, "to", "", "End of a custom time range as an ISO 8601 timestamp")
	cmd.Flags().StringSliceVar(&f.levels, "level", nil,
		"Log levels to include (comma-separated or repeatable): DEBUG, INFO, WARNING, ERROR")
	cmd.Flags().StringSliceVar(&f.servers, "server", []string{"primary"},
		"Servers to include (comma-separated or repeatable); primary selects the primary role, other values select pod names")
	cmd.Flags().StringSliceVar(&f.shards, "shard", nil,
		"Neki shards to include (comma-separated or repeatable)")
	cmd.Flags().IntVar(&f.limit, "limit", defaultLimit, "Maximum number of logs to return")
	cmd.Flags().IntVar(&f.page, "page", 1, "Page number to fetch")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	return cmd
}

func printHumanLogs(ch *cmdutil.Helper, entries []*LogEntry) {
	for _, entry := range entries {
		message := strings.ReplaceAll(entry.Message, "\r\n", `\n`)
		message = strings.ReplaceAll(message, "\r", `\r`)
		message = strings.ReplaceAll(message, "\n", `\n`)
		message = strings.ReplaceAll(message, "\t", `\t`)
		ch.Printer.Printf("%s %-7s %s\n", entry.Time, entry.Level, message)
	}
}

func validateFlags(f *flags, periodChanged bool) error {
	if (f.from == "") != (f.to == "") {
		return fmt.Errorf("--from and --to must be used together")
	}
	if f.from != "" && periodChanged {
		return fmt.Errorf("--period cannot be combined with --from and --to")
	}
	if !validPeriods[f.period] {
		return fmt.Errorf("invalid --period %q, must be one of: 15m, 1h, 3h, 6h, 12h, 1d, 7d, 8d", f.period)
	}
	if f.limit <= 0 {
		return fmt.Errorf("--limit must be greater than zero")
	}
	if f.page <= 0 {
		return fmt.Errorf("--page must be greater than zero")
	}

	for i, level := range f.levels {
		level = strings.ToUpper(level)
		if !validLevels[level] {
			return fmt.Errorf("invalid --level %q, must be one of: DEBUG, INFO, WARNING, ERROR", f.levels[i])
		}
		f.levels[i] = level
	}

	return nil
}

func createQuery(f *flags) string {
	parts := strings.Split(strings.TrimSpace(f.query), "|")
	filter := strings.TrimSpace(parts[0])
	if filter == "" {
		filter = "*"
	}

	timeFilter := f.period
	if f.from != "" {
		timeFilter = fmt.Sprintf("[%s, %s]", f.from, f.to)
	}
	query := []string{filter, "_time:" + timeFilter}

	if len(f.levels) > 0 {
		levels := make([]string, 0, len(f.levels))
		for _, level := range f.levels {
			levels = append(levels, "planetscale.level:"+level)
		}
		query = append(query, "("+strings.Join(levels, " OR ")+")")
	}

	if len(f.shards) > 0 {
		shards := make([]string, 0, len(f.shards))
		for _, shard := range f.shards {
			shards = append(shards, "planetscale.shard:"+shard)
		}
		query = append(query, "("+strings.Join(shards, " OR ")+")")
	}

	if len(f.servers) > 0 {
		servers := make([]string, 0, len(f.servers))
		for _, server := range f.servers {
			if server == "" {
				continue
			}
			if server == "primary" {
				servers = append(servers, "planetscale.role:primary")
			} else {
				servers = append(servers, "planetscale.pod:"+server)
			}
		}
		if len(servers) > 0 {
			query = append(query, "("+strings.Join(servers, " OR ")+")")
		}
	}

	for _, pipeline := range parts[1:] {
		pipeline = strings.TrimSpace(pipeline)
		if pipeline != "" {
			query = append(query, "| "+pipeline)
		}
	}

	offset := (f.page - 1) * f.limit
	query = append(query, fmt.Sprintf("| sort by (_time desc) | offset %d", offset))

	return strings.Join(query, " ")
}

func parseLogs(r io.Reader) ([]*LogEntry, error) {
	entries := make([]*LogEntry, 0)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

	for scanner.Scan() {
		entry, ok := parseLog(scanner.Bytes())
		if ok {
			entries = append(entries, entry)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading branch logs: %w", err)
	}

	return entries, nil
}

func parseLog(raw []byte) (*LogEntry, bool) {
	if len(raw) == 0 {
		return nil, false
	}

	var outer outerLog
	if err := json.Unmarshal(raw, &outer); err != nil || outer.Message == "" {
		return nil, false
	}

	message := outer.Message
	var inner innerLog
	if err := json.Unmarshal([]byte(outer.Message), &inner); err == nil {
		if inner.Message == "" {
			return nil, false
		}
		message = inner.Message
	}

	level := outer.Level
	if level == "" {
		level = "INFO"
	}

	return &LogEntry{
		Time:             outer.Time,
		Level:            level,
		Message:          message,
		Role:             outer.Role,
		Shard:            outer.Shard,
		Container:        outer.Container,
		AvailabilityZone: outer.AvailabilityZone,
		Pod:              outer.Pod,
		StreamID:         outer.StreamID,
		RawMessage:       outer.Message,
	}, true
}
