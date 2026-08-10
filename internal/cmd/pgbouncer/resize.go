package pgbouncer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// ResizeCmd changes a dedicated PgBouncer's size, replicas, target, or parameters.
func ResizeCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		size            string
		replicasPerCell int
		target          string
		parameters      []string
		wait            bool
		waitTimeout     time.Duration
	}

	cmd := &cobra.Command{
		Use:   "resize <database> <branch> <name>",
		Short: "Change a dedicated PgBouncer's size, replicas, target, or parameters",
		Long: `Change a dedicated PgBouncer's size, replicas-per-cell, traffic target, and/or
pgbouncer configuration parameters. Changes are queued as an asynchronous
resize request. Use "pscale pgbouncer resize status" to track it and
"pscale pgbouncer resize cancel" to cancel it while unfinished.

This is distinct from "pscale branch resize" pgbouncer.* parameters, which
configure the local PgBouncer on database nodes.`,
		Example: `  pscale pgbouncer resize mydb main read-pool --size PGB_10
  pscale pgbouncer resize mydb main read-pool --replicas-per-cell 2 --wait
  pscale pgbouncer resize mydb main read-pool --parameters pgbouncer.default_pool_size=100`,
		Args: cmdutil.RequiredArgs("database", "branch", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, name := args[0], args[1], args[2]

			if flags.size == "" && !cmd.Flags().Changed("replicas-per-cell") && flags.target == "" && len(flags.parameters) == 0 {
				return errors.New("nothing to change: pass at least one of --size, --replicas-per-cell, --target, or --parameters")
			}

			parameters, err := parseBouncerParameters(flags.parameters)
			if err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "PgBouncers"); err != nil {
				return err
			}

			req := &ps.ResizePostgresBouncerRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Bouncer:      name,
				BouncerSize:  flags.size,
				Target:       flags.target,
				Parameters:   parameters,
			}
			if cmd.Flags().Changed("replicas-per-cell") {
				req.ReplicasPerCell = &flags.replicasPerCell
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Requesting changes to PgBouncer %s on %s/%s...", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			resize, err := client.PostgresBouncers.Resize(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("PgBouncer %s does not exist on %s/%s (organization: %s)",
						printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if flags.wait {
				resize, err = waitForBouncerResize(ctx, ch, client, database, branch, name, resize, flags.waitTimeout)
				if err != nil {
					return err
				}
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Change to PgBouncer %s %s (state: %s).\n",
					printer.BoldBlue(name), resizeVerb(resize.State), printer.BoldBlue(resize.State))
				return nil
			}

			return ch.Printer.PrintResource(toPgBouncerResize(resize))
		},
	}

	cmd.Flags().StringVar(&flags.size, "size", "", "PgBouncer size SKU (e.g. PGB_10)")
	cmd.Flags().IntVar(&flags.replicasPerCell, "replicas-per-cell", 1, "Number of PgBouncer replica servers per cell")
	cmd.Flags().StringVar(&flags.target, "target", "", "Traffic target: primary, replica, or replica_az_affinity")
	cmd.Flags().StringArrayVar(&flags.parameters, "parameters", nil, "Set a PgBouncer parameter as namespace.name=value (e.g. pgbouncer.default_pool_size=100). Repeatable.")
	cmd.Flags().BoolVar(&flags.wait, "wait", false, "Wait for the resize request to complete before returning")
	cmd.Flags().DurationVar(&flags.waitTimeout, "wait-timeout", 10*time.Minute, "Maximum time to wait for the resize request to complete with --wait")

	cmd.AddCommand(ResizeStatusCmd(ch))
	cmd.AddCommand(ResizeCancelCmd(ch))

	return cmd
}

// ResizeStatusCmd shows the most recent resize request for a PgBouncer.
func ResizeStatusCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <database> <branch> <name>",
		Short: "Show the latest resize request for a dedicated PgBouncer",
		Args:  cmdutil.RequiredArgs("database", "branch", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, name := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "PgBouncers"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching resize requests for PgBouncer %s on %s/%s...", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			resizes, err := client.PostgresBouncers.ListResizes(ctx, &ps.ListPostgresBouncerResizesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Bouncer:      name,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("PgBouncer %s does not exist on %s/%s (organization: %s)",
						printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(resizes) == 0 {
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("PgBouncer %s has no resize requests.\n", printer.BoldBlue(name))
					return nil
				}
				return ch.Printer.PrintResource([]*PgBouncerResize{})
			}

			return ch.Printer.PrintResource(toPgBouncerResize(resizes[0]))
		},
	}

	return cmd
}

// ResizeCancelCmd cancels unfinished resize requests for a PgBouncer.
func ResizeCancelCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel <database> <branch> <name>",
		Short: "Cancel unfinished resize requests for a dedicated PgBouncer",
		Args:  cmdutil.RequiredArgs("database", "branch", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, name := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "PgBouncers"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Cancelling resize requests for PgBouncer %s on %s/%s...", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			err = client.PostgresBouncers.CancelResizes(ctx, &ps.CancelPostgresBouncerResizesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Bouncer:      name,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("PgBouncer %s does not exist on %s/%s (organization: %s)",
						printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Canceled unfinished resize requests for PgBouncer %s.\n", printer.BoldBlue(name))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result":   "canceled",
				"name":     name,
				"database": database,
				"branch":   branch,
			})
		},
	}

	return cmd
}

func parseBouncerParameters(sets []string) (map[string]map[string]string, error) {
	if len(sets) == 0 {
		return nil, nil
	}

	parameters := make(map[string]map[string]string)
	for _, set := range sets {
		key, value, found := strings.Cut(set, "=")
		if !found {
			return nil, fmt.Errorf("invalid --parameters %q: expected namespace.name=value (e.g. pgbouncer.default_pool_size=100)", set)
		}

		namespace, name, found := strings.Cut(key, ".")
		if !found || namespace == "" || name == "" {
			return nil, fmt.Errorf("invalid --parameters %q: parameter must be prefixed with its namespace, e.g. pgbouncer.%s=%s", set, key, value)
		}

		if parameters[namespace] == nil {
			parameters[namespace] = make(map[string]string)
		}
		parameters[namespace][name] = value
	}

	return parameters, nil
}

func waitForBouncerResize(ctx context.Context, ch *cmdutil.Helper, client *ps.Client, database, branch, name string, resize *ps.PostgresBouncerResizeRequest, timeout time.Duration) (*ps.PostgresBouncerResizeRequest, error) {
	end := ch.Printer.PrintProgress(fmt.Sprintf("Waiting for resize %s to complete...", printer.BoldBlue(resize.ID)))
	defer end()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		if resize.Finished() {
			end()
			return resize, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("resize %s did not complete within %s (current state: %s). Check progress with 'pscale pgbouncer resize status %s %s %s'",
				resize.ID, timeout, resize.State, database, branch, name)
		case <-ticker.C:
		}

		resizes, err := client.PostgresBouncers.ListResizes(ctx, &ps.ListPostgresBouncerResizesRequest{
			Organization: ch.Config.Organization,
			Database:     database,
			Branch:       branch,
			Bouncer:      name,
		})
		if err != nil {
			continue
		}
		for _, candidate := range resizes {
			if candidate.ID == resize.ID {
				resize = candidate
				break
			}
		}
	}
}

func resizeVerb(state string) string {
	switch state {
	case ps.PostgresBouncerResizeStateCompleted:
		return "completed"
	case ps.PostgresBouncerResizeStateCanceled:
		return "was canceled"
	default:
		return "queued"
	}
}

// PgBouncerResize is the human/JSON/CSV view of a bouncer resize request.
type PgBouncerResize struct {
	ID              string `header:"id" json:"id"`
	Bouncer         string `header:"bouncer" json:"bouncer"`
	State           string `header:"state" json:"state"`
	Size            string `header:"size" json:"size"`
	PreviousSize    string `header:"previous_size" json:"previous_size"`
	Target          string `header:"target" json:"target"`
	ReplicasPerCell int    `header:"replicas_per_cell" json:"replicas_per_cell"`
	Actor           string `header:"actor" json:"actor"`
	CreatedAt       int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt       int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.PostgresBouncerResizeRequest
}

func (r *PgBouncerResize) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(r.orig, "", "  ")
}

func (r *PgBouncerResize) MarshalCSVValue() interface{} {
	return []*PgBouncerResize{r}
}

func toPgBouncerResize(resize *ps.PostgresBouncerResizeRequest) *PgBouncerResize {
	out := &PgBouncerResize{
		ID:              resize.ID,
		Bouncer:         resize.Bouncer.Name,
		State:           resize.State,
		Target:          resize.Target,
		ReplicasPerCell: resize.ReplicasPerCell,
		Actor:           resize.Actor.Name,
		CreatedAt:       printer.GetMilliseconds(resize.CreatedAt),
		UpdatedAt:       printer.GetMilliseconds(resize.UpdatedAt),
		orig:            resize,
	}
	if resize.SKU != nil {
		if resize.SKU.DisplayName != "" {
			out.Size = resize.SKU.DisplayName
		} else {
			out.Size = resize.SKU.Name
		}
	} else {
		out.Size = "-"
	}
	if resize.PreviousSKU != nil {
		if resize.PreviousSKU.DisplayName != "" {
			out.PreviousSize = resize.PreviousSKU.DisplayName
		} else {
			out.PreviousSize = resize.PreviousSKU.Name
		}
	} else {
		out.PreviousSize = "-"
	}
	if out.Actor == "" {
		out.Actor = "-"
	}
	if out.Bouncer == "" {
		out.Bouncer = "-"
	}
	return out
}
