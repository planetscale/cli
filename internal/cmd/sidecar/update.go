package sidecar

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
		parameters []string
	}

	cmd := &cobra.Command{
		Use:   "update <database> <branch> <sidecar>",
		Short: "Update a sidecar for a Neki database branch",
		Long:  "Update a sidecar for a Neki database branch.\n\n<sidecar> is the sidecar ID from `sidecar list`, or the configuration profile name.\nChanges are requested and applied asynchronously; the sidecar state shows progress.",
		Args:  cmdutil.ExactArgs("database", "branch", "sidecar"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, sidecar := args[0], args[1], args[2]

			if !cmd.Flags().Changed("parameters") {
				return fmt.Errorf("at least one --parameters flag is required")
			}

			parameters, err := parseParameters(flags.parameters)
			if err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating sidecar %s", printer.BoldBlue(sidecar)))
			defer end()

			resource, err := client.NekiSidecars.Update(cmd.Context(), &ps.UpdateNekiSidecarRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Sidecar:      sidecar,
				Parameters:   parameters,
			})
			if err != nil {
				return handleError(err, database, branch, sidecar)
			}
			end()

			return ch.Printer.PrintResource(toSidecar(resource))
		},
	}

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
