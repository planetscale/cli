package keyspace

import (
	"fmt"
	"slices"
	"strings"

	"github.com/planetscale/cli/internal/cmd/vitessparams"
	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

var parameterNamespaces = []string{"vttablet", "mysqld"}

// ParametersCmd lists and changes the VTTablet and MySQL parameters of a keyspace.
func ParametersCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		namespace string
	}

	long := `List the VTTablet and MySQL parameters of a keyspace, including their current and default values.

To change parameters, use 'pscale keyspace parameters set <database> <branch> <keyspace> --parameters namespace.name=value'.`

	run := func(cmd *cobra.Command, args []string) error {
		database, branch, keyspace := args[0], args[1], args[2]

		if flags.namespace != "" && !slices.Contains(parameterNamespaces, flags.namespace) {
			return fmt.Errorf("invalid --namespace %q, must be one of: %s", flags.namespace, strings.Join(parameterNamespaces, ", "))
		}

		client, err := ch.Client()
		if err != nil {
			return err
		}

		end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching parameters for keyspace %s in %s/%s...", printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch)))
		defer end()

		parameters, err := client.Keyspaces.ListParameters(cmd.Context(), &ps.ListKeyspaceParametersRequest{
			Organization: ch.Config.Organization,
			Database:     database,
			Branch:       branch,
			Keyspace:     keyspace,
		})
		if err != nil {
			switch cmdutil.ErrCode(err) {
			case ps.ErrNotFound:
				return fmt.Errorf("database %s or branch %s does not exist in organization %s", printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
			default:
				return cmdutil.HandleError(err)
			}
		}
		end()

		if parameters == nil {
			return keyspaceNotFoundError(ch, database, branch, keyspace)
		}

		var selected []*ps.VitessParameter
		if flags.namespace == "" || flags.namespace == "vttablet" {
			selected = append(selected, parameters.VTTablet...)
		}
		if flags.namespace == "" || flags.namespace == "mysqld" {
			selected = append(selected, parameters.MySQL...)
		}

		return ch.Printer.PrintResource(vitessparams.ToParameters(selected))
	}

	registerFlags := func(cmd *cobra.Command) {
		cmd.Flags().StringVar(&flags.namespace, "namespace", "", "Only show parameters in this namespace: vttablet or mysqld.")
	}

	cmd := &cobra.Command{
		Use:     "parameters <database> <branch> <keyspace>",
		Aliases: []string{"params"},
		Short:   "List and change the VTTablet and MySQL parameters of a keyspace",
		Long:    long,
		Args:    cmdutil.RequiredArgs("database", "branch", "keyspace"),
		RunE:    run,
	}
	registerFlags(cmd)

	listCmd := &cobra.Command{
		Use:   "list <database> <branch> <keyspace>",
		Short: "List the VTTablet and MySQL parameters of a keyspace",
		Long:  long,
		Args:  cmdutil.RequiredArgs("database", "branch", "keyspace"),
		RunE:  run,
	}
	registerFlags(listCmd)

	cmd.AddCommand(listCmd, parametersSetCmd(ch), parametersChangesCmd(ch))
	return cmd
}

func keyspaceNotFoundError(ch *cmdutil.Helper, database, branch, keyspace string) error {
	return fmt.Errorf("keyspace %s does not exist in branch %s (database: %s, organization: %s)", printer.BoldBlue(keyspace), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
}
