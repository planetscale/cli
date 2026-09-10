package configprofile

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ParametersCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		namespace string
		extension bool
		internal  bool
	}

	cmd := &cobra.Command{
		Use:     "parameters <database> <branch> <name>",
		Short:   "List parameters for a Neki configuration profile",
		Args:    cmdutil.ExactArgs("database", "branch", "name"),
		Aliases: []string{"params"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, name := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			request := &ps.ListNekiShardConfigurationProfileParametersRequest{
				Organization:         ch.Config.Organization,
				Database:             database,
				Branch:               branch,
				ConfigurationProfile: name,
			}
			if cmd.Flags().Changed("extension") {
				request.Extension = &flags.extension
			}
			if cmd.Flags().Changed("internal") {
				request.Internal = &flags.internal
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching parameters for configuration profile %s", printer.BoldBlue(name)))
			defer end()
			parameters, err := client.NekiShardConfigurationProfiles.ListParameters(cmd.Context(), request)
			if err != nil {
				return handleError(err, database, branch, name)
			}
			end()

			if flags.namespace != "" {
				filtered := make([]*ps.NekiParameter, 0, len(parameters))
				for _, parameter := range parameters {
					if parameter.Namespace == flags.namespace {
						filtered = append(filtered, parameter)
					}
				}
				parameters = filtered
			}

			if len(parameters) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No parameters found for configuration profile %s.\n", printer.BoldBlue(name))
				return nil
			}
			return ch.Printer.PrintResource(toParameters(parameters))
		},
	}

	cmd.Flags().StringVar(&flags.namespace, "namespace", "", "Only show parameters in this namespace")
	cmd.Flags().BoolVar(&flags.extension, "extension", false, "Only show extension parameters")
	cmd.Flags().BoolVar(&flags.internal, "internal", false, "Only show internal parameters")
	return cmd
}
