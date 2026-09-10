package configprofile

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		clusterSize  string
		replicas     int
		major, minor string
		storage      storageFlags
	}
	cmd := &cobra.Command{
		Use:   "create <database> <branch> <name>",
		Short: "Create a Neki configuration profile",
		Args:  cmdutil.ExactArgs("database", "branch", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, name := args[0], args[1], args[2]
			if cmd.Flags().Changed("postgres-minor-version") && !cmd.Flags().Changed("postgres-major-version") {
				return fmt.Errorf("--postgres-major-version is required with --postgres-minor-version")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating configuration profile %s", printer.BoldBlue(name)))
			defer end()
			profile, err := client.NekiShardConfigurationProfiles.Create(cmd.Context(), &ps.CreateNekiShardConfigurationProfileRequest{
				Organization:         ch.Config.Organization,
				Database:             database,
				Branch:               branch,
				Name:                 name,
				ClusterSize:          stringPointerIfChanged(cmd, "cluster-size", flags.clusterSize),
				Replicas:             intPointerIfChanged(cmd, "replicas", flags.replicas),
				PostgresMajorVersion: stringPointerIfChanged(cmd, "postgres-major-version", flags.major),
				PostgresMinorVersion: stringPointerIfChanged(cmd, "postgres-minor-version", flags.minor),
				Storage:              storageFromFlags(cmd, flags.storage),
			})
			if err != nil {
				return handleError(err, database, branch, "")
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Configuration profile %s was successfully created.\n", printer.BoldBlue(profile.Name))
			}
			return ch.Printer.PrintResource(toProfile(profile))
		},
	}
	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "", "Cluster size for shards in the profile")
	cmd.Flags().IntVar(&flags.replicas, "replicas", 0, "Number of replicas for shards in the profile")
	cmd.Flags().StringVar(&flags.major, "postgres-major-version", "", "PostgreSQL major version")
	cmd.Flags().StringVar(&flags.minor, "postgres-minor-version", "", "PostgreSQL minor version")
	bindStorageFlags(cmd, &flags.storage)
	return cmd
}
