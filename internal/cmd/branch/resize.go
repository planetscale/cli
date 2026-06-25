package branch

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// ResizeCmd resizes a Postgres branch's cluster.
func ResizeCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		clusterSize string
	}

	cmd := &cobra.Command{
		Use:   "resize <database> <branch>",
		Short: "Resize a Postgres branch's cluster",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			if flags.clusterSize == "" {
				return errors.New("the --cluster-size flag is required")
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
				return fmt.Errorf("branch resize is only supported for PostgreSQL databases. To resize a MySQL keyspace, use %s", printer.BoldBlue("pscale keyspace resize"))
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Resizing branch %s in %s to %s...", printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(flags.clusterSize)))
			defer end()

			change, err := client.PostgresBranches.Resize(ctx, &ps.ResizePostgresBranchRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				ClusterSize:  flags.clusterSize,
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

			// A nil change request means the branch is already configured with
			// the requested cluster size (the API responded 204 No Content).
			if change == nil {
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("Branch %s is already configured with cluster size %s.\n", printer.BoldBlue(branch), printer.BoldBlue(flags.clusterSize))
				}
				return nil
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Resize of branch %s to %s started (state: %s).\n", printer.BoldBlue(branch), printer.BoldBlue(change.ClusterDisplayName), printer.BoldBlue(change.State))
				return nil
			}

			return ch.Printer.PrintResource(toPostgresBranchResize(change))
		},
	}

	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "", "New cluster size for the branch. Use 'pscale size cluster list' to see the valid sizes.")
	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return cmdutil.ClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}

type postgresBranchResize struct {
	ID                  string `header:"id" json:"id"`
	State               string `header:"state" json:"state"`
	ClusterSize         string `header:"cluster_size" json:"cluster_size"`
	PreviousClusterSize string `header:"previous_cluster_size" json:"previous_cluster_size"`
	Replicas            int    `header:"replicas" json:"replicas"`

	orig *ps.PostgresBranchClusterResizeRequest
}

func toPostgresBranchResize(c *ps.PostgresBranchClusterResizeRequest) *postgresBranchResize {
	return &postgresBranchResize{
		ID:                  c.ID,
		State:               c.State,
		ClusterSize:         c.ClusterDisplayName,
		PreviousClusterSize: c.PreviousClusterDisplayName,
		Replicas:            c.Replicas,
		orig:                c,
	}
}

func (p *postgresBranchResize) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(p.orig, "", "  ")
}
