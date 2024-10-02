package keyspace

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

type KeyspaceResizeRequest struct {
	ID    string `json:"id"`
	State string `json:"state"`

	ClusterSize string `header:"cluster_size" json:"cluster_rate_name"`

	Replicas      uint `json:"replicas"`
	ExtraReplicas uint `json:"extra_replicas"`

	CreatedAt   int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	StartedAt   *int64 `header:"started_at,timestamp(ms|utc|human)" json:"started_at"`
	CompletedAt *int64 `header:"completed_at,timestamp(ms|utc|human)" json:"completed_at"`

	orig *planetscale.KeyspaceResizeRequest
}

// ResizeCmd encapsulates the command for resizing a keyspace within a branch.
func ResizeCmd(ch *cmdutil.Helper) *cobra.Command {
	resizeReq := &planetscale.ResizeKeyspaceRequest{}

	var flags struct {
		additionalReplicas int
		clusterSize        string
	}

	cmd := &cobra.Command{
		Use:   "resize <database> <branch> <keyspace>",
		Short: "Resize a keyspace within a branch",
		Args:  cmdutil.RequiredArgs("database", "branch", "keyspace"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, keyspace := args[0], args[1], args[2]

			resizeReq.Organization = ch.Config.Organization
			resizeReq.Database = database
			resizeReq.Branch = branch
			resizeReq.Keyspace = keyspace

			if cmd.Flags().Changed("additional-replicas") {
				additionalReplicas := uint(flags.additionalReplicas)
				resizeReq.ExtraReplicas = &additionalReplicas
			}

			if cmd.Flags().Changed("cluster-size") {
				size := planetscale.ClusterSize(flags.clusterSize)
				resizeReq.ClusterSize = &size
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Saving changes to keyspace %s in %s/%s", keyspace, database, branch))
			defer end()

			krr, err := client.Keyspaces.Resize(ctx, resizeReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("database %s or branch %s or keyspace %s does not exist in organization %s", printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(keyspace), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Successfully saved changes to keyspace %s.\n", printer.BoldBlue(keyspace))
				return nil
			}

			return ch.Printer.PrintResource(toKeyspaceResizeRequest(krr))
		},
	}

	cmd.Flags().IntVar(&flags.additionalReplicas, "additional-replicas", 0, "Number of additional additionalReplicas to add to the keyspace")
	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "", "The cluster size to use for the keyspace")

	return cmd
}

func toKeyspaceResizeRequest(krr *planetscale.KeyspaceResizeRequest) *KeyspaceResizeRequest {
	return &KeyspaceResizeRequest{
		ID:            krr.ID,
		State:         krr.State,
		ClusterSize:   cmdutil.ToClusterSizeSlug(krr.ClusterSize),
		Replicas:      krr.Replicas,
		ExtraReplicas: krr.ExtraReplicas,
		CreatedAt:     cmdutil.TimeToMilliseconds(krr.CreatedAt),
		StartedAt:     printer.GetMillisecondsIfExists(krr.StartedAt),
		CompletedAt:   printer.GetMillisecondsIfExists(krr.CompletedAt),
		orig:          krr,
	}
}
