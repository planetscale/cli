package keyspace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// CreateCmd encapsulates the command for creating a new keyspace within a branch.
func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	createReq := &planetscale.CreateKeyspaceRequest{}

	var flags struct {
		shards             int
		clusterSize        string
		additionalReplicas int
		wait               bool
	}

	cmd := &cobra.Command{
		Use:   "create <database> <branch> <keyspace>",
		Short: "Create a new keyspace within a branch",
		Args:  cmdutil.RequiredArgs("database", "branch", "keyspace"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, keyspace := args[0], args[1], args[2]

			createReq.Organization = ch.Config.Organization
			createReq.Database = database
			createReq.Branch = branch
			createReq.Name = keyspace
			createReq.Shards = flags.shards
			createReq.ExtraReplicas = flags.additionalReplicas
			createReq.ClusterSize = flags.clusterSize

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating keyspace %s in %s/%s", printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			k, err := client.Keyspaces.Create(ctx, createReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("database %s or branch %s does not exist in organization %s", printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if flags.wait {
				end := ch.Printer.PrintProgress(fmt.Sprintf("Waiting until keyspace %s is ready...", printer.BoldBlue(keyspace)))
				defer end()

				getReq := &planetscale.GetKeyspaceRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Branch:       branch,
					Keyspace:     keyspace,
				}
				if err := waitUntilReady(ctx, client, ch.Printer, ch.Debug(), getReq); err != nil {
					return err
				}
				end()
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Keyspace %s was successfully created.\n", printer.BoldBlue(k.Name))
				return nil
			}

			return ch.Printer.PrintResource(toKeyspace(k))
		},
	}
	cmd.Flags().IntVar(&flags.shards, "shards", 1, "Number of shards in the keyspace")
	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "PS-10", "cluster size for the keyspace. Use `pscale size cluster list` to get a list of valid sizes.")
	cmd.Flags().IntVar(&flags.additionalReplicas, "additional-replicas", 0, "number of additional replicas per shard. By default, each production cluster includes 2 replicas.")
	cmd.Flags().BoolVar(&flags.wait, "wait", false, "Wait until the keyspace is ready")

	cmd.MarkFlagRequired("cluster-size")

	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return cmdutil.BranchClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}

func waitUntilReady(ctx context.Context, client *planetscale.Client, printer *printer.Printer, debug bool, getReq *planetscale.GetKeyspaceRequest) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ctx.Done():
			return errors.New("keyspace creation timed out")
		case <-ticker.C:
			resp, err := client.Keyspaces.Get(ctx, getReq)
			if err != nil {
				if debug {
					printer.Printf("fetching keyspace %s/%s %s failed: %s", getReq.Database, getReq.Branch, getReq.Keyspace, err)
				}
				continue
			}

			if resp.Ready {
				return nil
			}
		}
	}
}
