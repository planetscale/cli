package admin

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
		size       string
		parameters []string
	}

	cmd := &cobra.Command{
		Use:   "update <database> <branch>",
		Short: "Update the admin for a Neki database branch",
		Long:  "Update the admin for a Neki database branch.\n\nChanges are requested and applied asynchronously; the admin state shows progress.",
		Args:  cmdutil.ExactArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := args[0], args[1]

			changed := false
			for _, flag := range []string{"size", "parameters"} {
				changed = changed || cmd.Flags().Changed(flag)
			}
			if !changed {
				return fmt.Errorf("at least one of --size or --parameters is required")
			}

			parameters, err := parseParameters(flags.parameters)
			if err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating admin for %s/%s", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			resource, err := client.NekiAdmins.Update(cmd.Context(), &ps.UpdateNekiAdminRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				AdminSize:    stringPointerIfChanged(cmd, "size", cmdutil.ToSizeSKUName(flags.size)),
				Parameters:   parameters,
			})
			if err != nil {
				return handleError(err, database, branch)
			}
			end()

			return ch.Printer.PrintResource(toAdmin(resource))
		},
	}

	cmd.Flags().StringVar(&flags.size, "size", "", "New admin size SKU (e.g. NKA-0)")
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
