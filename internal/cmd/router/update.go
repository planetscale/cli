package router

import (
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func UpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		size                 string
		replicasPerCell      int
		autoscaling          bool
		maxReplicasPerCell   int
		targetCPUUtilization int
		parameters           []string
	}

	cmd := &cobra.Command{
		Use:   "update <database> <branch> <name>",
		Short: "Update a router for a Neki database branch",
		Long:  "Update a router for a Neki database branch.\n\nChanges are requested and applied asynchronously; the router state shows progress.",
		Args:  cmdutil.ExactArgs("database", "branch", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, name := args[0], args[1], args[2]

			changed := false
			for _, flag := range []string{"size", "replicas-per-cell", "autoscaling", "max-replicas-per-cell", "target-cpu-utilization", "parameters"} {
				changed = changed || cmd.Flags().Changed(flag)
			}
			if !changed {
				return fmt.Errorf("at least one update flag is required")
			}

			parameters, err := parseParameters(flags.parameters)
			if err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating router %s", printer.BoldBlue(name)))
			defer end()

			router, err := client.NekiRouters.Update(cmd.Context(), &ps.UpdateNekiRouterRequest{
				Organization:         ch.Config.Organization,
				Database:             database,
				Branch:               branch,
				Router:               name,
				RouterSize:           stringPointerIfChanged(cmd, "size", cmdutil.ToSizeSKUName(flags.size)),
				ReplicasPerCell:      intPointerIfChanged(cmd, "replicas-per-cell", flags.replicasPerCell),
				Autoscaling:          boolPointerIfChanged(cmd, "autoscaling", flags.autoscaling),
				MaxReplicasPerCell:   intPointerIfChanged(cmd, "max-replicas-per-cell", flags.maxReplicasPerCell),
				TargetCPUUtilization: intPointerIfChanged(cmd, "target-cpu-utilization", flags.targetCPUUtilization),
				Parameters:           parameters,
			})
			if err != nil {
				return handleError(err, database, branch, name)
			}
			end()

			return ch.Printer.PrintResource(toRouter(router))
		},
	}

	cmd.Flags().StringVar(&flags.size, "size", "", "New router size SKU (e.g. NKR-5)")
	cmd.Flags().IntVar(&flags.replicasPerCell, "replicas-per-cell", 0, "New number of replicas in each cell")
	cmd.Flags().BoolVar(&flags.autoscaling, "autoscaling", false, "Whether the router scales horizontally within each cell")
	cmd.Flags().IntVar(&flags.maxReplicasPerCell, "max-replicas-per-cell", 0, "Maximum number of replicas in each cell when autoscaling")
	cmd.Flags().IntVar(&flags.targetCPUUtilization, "target-cpu-utilization", 0, "Target average CPU utilization percentage when autoscaling")
	cmd.Flags().StringArrayVar(&flags.parameters, "parameters", nil, "Set a parameter as namespace.name=value; repeatable")

	return cmd
}

func parseParameters(values []string) (map[string]map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make(map[string]map[string]string)
	for _, value := range values {
		key, setting, ok := strings.Cut(value, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --parameters %q: expected namespace.name=value", value)
		}
		namespace, name, ok := strings.Cut(key, ".")
		if !ok || namespace == "" || name == "" {
			return nil, fmt.Errorf("invalid --parameters %q: expected namespace.name=value", value)
		}
		if out[namespace] == nil {
			out[namespace] = make(map[string]string)
		}
		out[namespace][name] = setting
	}
	return out, nil
}

func stringPointerIfChanged(cmd *cobra.Command, name, value string) *string {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	return &value
}

func intPointerIfChanged(cmd *cobra.Command, name string, value int) *int {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	return &value
}

func boolPointerIfChanged(cmd *cobra.Command, name string, value bool) *bool {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	return &value
}
