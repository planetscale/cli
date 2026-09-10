package configprofile

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
		name, clusterSize string
		replicas          int
		parameters        []string
		major, minor      string
		storage           storageFlags
	}
	cmd := &cobra.Command{
		Use:   "update <database> <branch> <name>",
		Short: "Update a Neki configuration profile",
		Args:  cmdutil.ExactArgs("database", "branch", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, profileName := args[0], args[1], args[2]
			changed := false
			for _, name := range []string{"name", "cluster-size", "replicas", "parameters", "postgres-major-version", "postgres-minor-version"} {
				changed = changed || cmd.Flags().Changed(name)
			}
			changed = changed || storageFlagChanged(cmd)
			if !changed {
				return fmt.Errorf("at least one update flag is required")
			}
			if cmd.Flags().Changed("postgres-minor-version") && !cmd.Flags().Changed("postgres-major-version") {
				return fmt.Errorf("--postgres-major-version is required with --postgres-minor-version")
			}

			parameters, err := parseParameters(flags.parameters)
			if err != nil {
				return err
			}
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating configuration profile %s", printer.BoldBlue(profileName)))
			defer end()
			profile, err := client.NekiShardConfigurationProfiles.Update(cmd.Context(), &ps.UpdateNekiShardConfigurationProfileRequest{
				Organization:         ch.Config.Organization,
				Database:             database,
				Branch:               branch,
				ConfigurationProfile: profileName,
				Name:                 stringPointerIfChanged(cmd, "name", flags.name),
				ClusterSize:          stringPointerIfChanged(cmd, "cluster-size", cmdutil.ToSizeSKUName(flags.clusterSize)),
				Replicas:             intPointerIfChanged(cmd, "replicas", flags.replicas),
				Parameters:           parameters,
				PostgresMajorVersion: stringPointerIfChanged(cmd, "postgres-major-version", flags.major),
				PostgresMinorVersion: stringPointerIfChanged(cmd, "postgres-minor-version", flags.minor),
				Storage:              storageFromFlags(cmd, flags.storage),
			})
			if err != nil {
				return handleError(err, database, branch, profileName)
			}
			end()
			return ch.Printer.PrintResource(toProfile(profile))
		},
	}
	cmd.Flags().StringVar(&flags.name, "name", "", "New name for the configuration profile")
	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "", "New cluster size for shards in the profile")
	cmd.Flags().IntVar(&flags.replicas, "replicas", 0, "New number of replicas for shards in the profile")
	cmd.Flags().StringArrayVar(&flags.parameters, "parameters", nil, "Set a parameter as namespace.name=value; repeatable")
	cmd.Flags().StringVar(&flags.major, "postgres-major-version", "", "PostgreSQL major version")
	cmd.Flags().StringVar(&flags.minor, "postgres-minor-version", "", "PostgreSQL minor version")
	bindStorageFlags(cmd, &flags.storage)
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
