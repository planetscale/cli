package branch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// ResizeCmd changes a Postgres branch's cluster: its size, replica count,
// and/or configuration parameters. All changes are queued as a single change
// request via the branch changes API.
func ResizeCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		clusterSize string
		replicas    int
		parameters  []string
		wait        bool
		waitTimeout time.Duration
	}

	cmd := &cobra.Command{
		Use:   "resize <database> <branch>",
		Short: "Change a Postgres branch's cluster size, replicas, or parameters",
		Long: `Change a Postgres branch's cluster size, replica count, and/or configuration
parameters. All requested changes are combined into a single change request
that is applied asynchronously. Use "pscale branch resize status" to track it
and "pscale branch resize cancel" to cancel it while queued.`,
		Example: `  pscale branch resize mydb main --cluster-size PS_10_GCP_X86
  pscale branch resize mydb main --parameters pgconf.max_connections=200
  pscale branch resize mydb main --cluster-size PS_20_GCP_X86 --replicas 2 --parameters pgconf.max_connections=500 --wait`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			if flags.clusterSize == "" && !cmd.Flags().Changed("replicas") && len(flags.parameters) == 0 {
				return errors.New("nothing to change: pass at least one of --cluster-size, --replicas, or --parameters")
			}

			parameters, err := parseParameterSets(flags.parameters)
			if err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			db, err := client.Databases.Get(ctx, &ps.GetDatabaseRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s", printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			if db.Kind == "mysql" {
				return fmt.Errorf("branch resize is only supported for PostgreSQL databases. To resize a MySQL keyspace, use %s. To resize VTGates, use %s", printer.BoldBlue("pscale keyspace resize"), printer.BoldBlue("pscale branch vtgate resize"))
			}

			restartParams, err := preflightParameters(ctx, client, ch.Config.Organization, database, branch, parameters)
			if err != nil {
				return err
			}

			resizeReq := &ps.ResizePostgresBranchRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				ClusterSize:  flags.clusterSize,
				Parameters:   parameters,
			}
			if cmd.Flags().Changed("replicas") {
				replicas := flags.replicas
				resizeReq.Replicas = &replicas
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Requesting changes to branch %s in %s...", printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			change, err := client.PostgresBranches.Resize(ctx, resizeReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s or branch %s does not exist in organization %s", printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			// A nil change request means the branch already matches the
			// requested configuration (the API responded 204 No Content).
			if change == nil {
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("Branch %s already matches the requested configuration; no changes applied.\n", printer.BoldBlue(branch))
					return nil
				}
				return ch.Printer.PrintResource(map[string]string{
					"result": "no_change",
					"branch": branch,
				})
			}

			if flags.wait {
				change, err = waitForChange(ctx, ch, client, database, branch, change, flags.waitTimeout)
				if err != nil {
					return err
				}
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Change to branch %s %s (state: %s).\n", printer.BoldBlue(branch), changeVerb(change.State), printer.BoldBlue(change.State))
				// Only warn about pending restarts: once the change request
				// has finished, any required restart already happened (or the
				// change was canceled and no restart will occur).
				if len(restartParams) > 0 && !change.Finished() {
					ch.Printer.Printf("Note: %s require a database restart to take effect.\n", printer.BoldBlue(strings.Join(restartParams, ", ")))
				}
				return nil
			}

			return ch.Printer.PrintResource(toPostgresBranchResize(change))
		},
	}

	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "", "New cluster size for the branch (a fully-qualified SKU name, e.g. PS_10_GCP_X86). Use 'pscale size cluster list --engine postgresql' to see the valid sizes.")
	cmd.Flags().IntVar(&flags.replicas, "replicas", 0, "Desired number of replicas for the branch.")
	cmd.Flags().StringArrayVar(&flags.parameters, "parameters", nil, "Set a configuration parameter as namespace.name=value (e.g. pgconf.max_connections=200). Repeatable. Use 'pscale branch parameters list' to see available parameters.")
	cmd.Flags().BoolVar(&flags.wait, "wait", false, "Wait for the change request to complete before returning.")
	cmd.Flags().DurationVar(&flags.waitTimeout, "wait-timeout", 10*time.Minute, "Maximum time to wait for the change request to complete with --wait.")

	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return cmdutil.PostgresBranchClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	cmd.AddCommand(ResizeStatusCmd(ch))
	cmd.AddCommand(ResizeCancelCmd(ch))

	return cmd
}

// parseParameterSets parses repeated --parameters flags of the form
// namespace.name=value into the nested map the API expects.
func parseParameterSets(sets []string) (map[string]map[string]string, error) {
	if len(sets) == 0 {
		return nil, nil
	}

	parameters := make(map[string]map[string]string)
	for _, set := range sets {
		key, value, found := strings.Cut(set, "=")
		if !found {
			return nil, fmt.Errorf("invalid --parameters %q: expected namespace.name=value (e.g. pgconf.max_connections=200)", set)
		}

		namespace, name, found := strings.Cut(key, ".")
		if !found || namespace == "" || name == "" {
			return nil, fmt.Errorf("invalid --parameters %q: parameter must be prefixed with its namespace, e.g. pgconf.%s=%s. Run 'pscale branch parameters list <database> <branch>' to see namespaces", set, key, value)
		}

		if parameters[namespace] == nil {
			parameters[namespace] = make(map[string]string)
		}
		parameters[namespace][name] = value
	}

	return parameters, nil
}

// preflightParameters validates the requested parameters against the branch's
// parameter catalog before submitting the change request, and returns the
// names of requested parameters that require a database restart. Catalog
// lookup failures are not fatal: the API validates parameters authoritatively
// on submission.
func preflightParameters(ctx context.Context, client *ps.Client, organization, database, branch string, parameters map[string]map[string]string) ([]string, error) {
	if len(parameters) == 0 {
		return nil, nil
	}

	catalog, err := client.PostgresBranches.ListParameters(ctx, &ps.ListPostgresParametersRequest{
		Organization: organization,
		Database:     database,
		Branch:       branch,
	})
	if err != nil {
		return nil, nil
	}

	known := make(map[string]*ps.PostgresParameter, len(catalog))
	for _, param := range catalog {
		known[param.Namespace+"."+param.Name] = param
	}

	// Validate in sorted order so errors and restart notes are deterministic
	// regardless of map iteration order.
	var keys []string
	for namespace, params := range parameters {
		for name := range params {
			keys = append(keys, namespace+"."+name)
		}
	}
	slices.Sort(keys)

	var restartParams []string
	for _, key := range keys {
		param, ok := known[key]
		if !ok {
			return nil, fmt.Errorf("unknown parameter %s. Run 'pscale branch parameters list %s %s' to see available parameters", printer.BoldBlue(key), database, branch)
		}
		if param.Immutable {
			return nil, fmt.Errorf("parameter %s cannot be changed", printer.BoldBlue(key))
		}
		if param.Restart {
			restartParams = append(restartParams, key)
		}
	}

	return restartParams, nil
}

// waitForChange polls the change request until it reaches a terminal state
// (completed or canceled) or the timeout elapses.
func waitForChange(ctx context.Context, ch *cmdutil.Helper, client *ps.Client, database, branch string, change *ps.PostgresBranchClusterResizeRequest, timeout time.Duration) (*ps.PostgresBranchClusterResizeRequest, error) {
	end := ch.Printer.PrintProgress(fmt.Sprintf("Waiting for change %s to complete...", printer.BoldBlue(change.ID)))
	defer end()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		if change.Finished() {
			end()
			return change, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("change %s did not complete within %s (current state: %s). Check progress with 'pscale branch resize status %s %s'", change.ID, timeout, change.State, database, branch)
		case <-ticker.C:
		}

		updated, err := client.PostgresBranches.GetChange(ctx, &ps.GetPostgresBranchChangeRequest{
			Organization: ch.Config.Organization,
			Database:     database,
			Branch:       branch,
			ID:           change.ID,
		})
		if err != nil {
			// Retry transient poll failures until the timeout, matching the
			// --wait behavior of the other commands (e.g. deploy-request
			// deploy, database create).
			continue
		}
		change = updated
	}
}

func changeVerb(state string) string {
	switch state {
	case ps.PostgresBranchChangeStateCompleted:
		return "completed"
	case ps.PostgresBranchChangeStateCanceled:
		return "was canceled"
	default:
		return "started"
	}
}

type postgresBranchResize struct {
	ID                  string `header:"id" json:"id"`
	State               string `header:"state" json:"state"`
	ClusterSize         string `header:"cluster_size" json:"cluster_size"`
	PreviousClusterSize string `header:"previous_cluster_size" json:"previous_cluster_size"`
	Replicas            int    `header:"replicas" json:"replicas"`
	Parameters          string `header:"parameters" json:"parameters"`

	CreatedAt   int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	CompletedAt *int64 `header:"completed_at,timestamp(ms|utc|human)" json:"completed_at"`

	orig *ps.PostgresBranchClusterResizeRequest
}

func toPostgresBranchResize(c *ps.PostgresBranchClusterResizeRequest) *postgresBranchResize {
	return &postgresBranchResize{
		ID:                  c.ID,
		State:               c.State,
		ClusterSize:         c.ClusterDisplayName,
		PreviousClusterSize: c.PreviousClusterDisplayName,
		Replicas:            c.Replicas,
		Parameters:          formatChangeParameters(c.Parameters),
		CreatedAt:           cmdutil.TimeToMilliseconds(c.CreatedAt),
		CompletedAt:         printer.GetMillisecondsIfExists(c.CompletedAt),
		orig:                c,
	}
}

// formatChangeParameters renders a change request's parameters as a compact
// namespace.name=value list for table output.
func formatChangeParameters(parameters map[string]map[string]any) string {
	var parts []string
	for namespace, params := range parameters {
		for name, value := range params {
			parts = append(parts, fmt.Sprintf("%s.%s=%s", namespace, name, formatParameterValue(value)))
		}
	}
	slices.Sort(parts)
	return strings.Join(parts, ", ")
}

func (p *postgresBranchResize) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(p.orig, "", "  ")
}

func (p *postgresBranchResize) MarshalCSVValue() interface{} {
	return []*postgresBranchResize{p}
}
