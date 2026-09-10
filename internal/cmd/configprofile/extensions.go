package configprofile

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ExtensionsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extensions <database> <branch> <name>",
		Short: "List extensions for a Neki configuration profile",
		Args:  cmdutil.ExactArgs("database", "branch", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, name := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching extensions for configuration profile %s", printer.BoldBlue(name)))
			defer end()
			extensions, err := client.NekiShardConfigurationProfiles.ListExtensions(cmd.Context(), &ps.ListNekiShardConfigurationProfileExtensionsRequest{
				Organization:         ch.Config.Organization,
				Database:             database,
				Branch:               branch,
				ConfigurationProfile: name,
			})
			if err != nil {
				return handleError(err, database, branch, name)
			}
			end()

			if len(extensions) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No extensions found for configuration profile %s.\n", printer.BoldBlue(name))
				return nil
			}
			return ch.Printer.PrintResource(toExtensions(extensions))
		},
	}

	cmd.AddCommand(extensionsEnableCmd(ch), extensionsDisableCmd(ch))
	return cmd
}

func extensionsEnableCmd(ch *cmdutil.Helper) *cobra.Command {
	return extensionsToggleCmd(ch, "enable", "Enable an extension for a Neki configuration profile", true)
}

func extensionsDisableCmd(ch *cmdutil.Helper) *cobra.Command {
	return extensionsToggleCmd(ch, "disable", "Disable an extension for a Neki configuration profile", false)
}

func extensionsToggleCmd(ch *cmdutil.Helper, use, short string, enabled bool) *cobra.Command {
	action := "Enabling"
	if !enabled {
		action = "Disabling"
	}

	return &cobra.Command{
		Use:   use + " <database> <branch> <name> <extension>",
		Short: short,
		Args:  cmdutil.ExactArgs("database", "branch", "name", "extension"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, name, extensionName := args[0], args[1], args[2], args[3]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("%s extension %s", action, printer.BoldBlue(extensionName)))
			defer end()
			extension, err := client.NekiShardConfigurationProfiles.UpdateExtension(cmd.Context(), &ps.UpdateNekiShardConfigurationProfileExtensionRequest{
				Organization:         ch.Config.Organization,
				Database:             database,
				Branch:               branch,
				ConfigurationProfile: name,
				Extension:            extensionName,
				Enabled:              enabled,
			})
			if err != nil {
				return handleError(err, database, branch, name)
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Extension %s on configuration profile %s was %sd.\n", printer.BoldBlue(extension.Name), printer.BoldBlue(name), use)
			}
			return ch.Printer.PrintResource(toExtension(extension))
		},
	}
}
