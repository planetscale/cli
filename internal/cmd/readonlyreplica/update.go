package readonlyreplica

import (
	"errors"
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// UpdateCmd updates a read-only replica.
func UpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		replicas    int
		clusterSize string
		parameters  []string
	}

	cmd := &cobra.Command{
		Use:   "update <database> <branch> <name>",
		Short: "Update a read-only replica",
		Long: `Update a read-only replica's cluster size, instance count, and/or
PostgreSQL configuration parameters. Parameter values must be greater than or
equal to the primary branch's corresponding values.`,
		Example: `  pscale read-only-replica update mydb main analytics --replicas 2
  pscale read-only-replica update mydb main analytics --cluster-size PS_20_GCP_X86
  pscale read-only-replica update mydb main analytics --parameters pgconf.max_connections=300`,
		Args: cmdutil.RequiredArgs("database", "branch", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, name := args[0], args[1], args[2]

			if !cmd.Flags().Changed("replicas") && flags.clusterSize == "" && len(flags.parameters) == 0 {
				return errors.New("nothing to change: pass at least one of --replicas, --cluster-size, or --parameters")
			}

			parameters, err := parseParameters(flags.parameters)
			if err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}
			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "read-only replicas"); err != nil {
				return err
			}

			req := &ps.UpdatePostgresReadOnlyReplicaRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Replica:      name,
				ClusterSize:  flags.clusterSize,
				Parameters:   parameters,
			}
			if cmd.Flags().Changed("replicas") {
				req.Replicas = &flags.replicas
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating read-only replica %s on %s/%s", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			replica, err := client.PostgresReadOnlyReplicas.Update(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("read-only replica %s does not exist on %s/%s (organization: %s)",
						printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Update requested for read-only replica %s on %s/%s (state: %s).\n",
					printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(replica.State))
				return nil
			}
			return ch.Printer.PrintResource(toReadOnlyReplica(replica))
		},
	}

	cmd.Flags().IntVar(&flags.replicas, "replicas", 0, "Desired number of instances serving reads")
	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "", "New cluster size SKU")
	cmd.Flags().StringArrayVar(&flags.parameters, "parameters", nil, "Set a parameter as namespace.name=value (for example pgconf.max_connections=300); repeatable")

	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return cmdutil.PostgresBranchClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}

func parseParameters(values []string) (map[string]map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	parameters := make(map[string]map[string]string)
	for _, value := range values {
		key, parameterValue, found := strings.Cut(value, "=")
		if !found {
			return nil, fmt.Errorf("invalid --parameters %q: expected namespace.name=value (for example pgconf.max_connections=300)", value)
		}
		namespace, name, found := strings.Cut(key, ".")
		if !found || namespace == "" || name == "" {
			return nil, fmt.Errorf("invalid --parameters %q: parameter must include its namespace (for example pgconf.max_connections=300)", value)
		}
		if parameters[namespace] == nil {
			parameters[namespace] = make(map[string]string)
		}
		parameters[namespace][name] = parameterValue
	}
	return parameters, nil
}
