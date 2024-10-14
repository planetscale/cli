package keyspace

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

type KeyspaceResizeRequest struct {
	ID    string `json:"id"`
	State string `header:"state" json:"state"`

	ClusterSize         string `header:"cluster_size" json:"cluster_rate_name"`
	PreviousClusterSize string `header:"previous_cluster_size" json:"previous_cluster_rate_name"`

	PreviousReplicas uint `header:"previous_replicas" json:"previous_replicas"`
	Replicas         uint `header:"replicas" json:"replicas"`
	ExtraReplicas    uint `header:"extra_replicas" json:"extra_replicas"`

	CreatedAt   int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	StartedAt   *int64 `header:"started_at,timestamp(ms|utc|human)" json:"started_at"`
	CompletedAt *int64 `header:"completed_at,timestamp(ms|utc|human)" json:"completed_at"`

	orig *ps.KeyspaceResizeRequest
}

// ResizeCmd encapsulates the command for resizing a keyspace within a branch.
func ResizeCmd(ch *cmdutil.Helper) *cobra.Command {
	resizeReq := &ps.ResizeKeyspaceRequest{}

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
				size := ps.ClusterSize(flags.clusterSize)
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
				case ps.ErrNotFound:
					return fmt.Errorf("database %s, branch %s, or keyspace %s does not exist in organization %s", printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(keyspace), printer.BoldBlue(ch.Config.Organization))
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

	cmd.Flags().IntVar(&flags.additionalReplicas, "additional-replicas", 0, "number of additional replicas per shard. By default, each production cluster comes with 2 replicas.")
	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "", "cluster size for the keyspace: Options: PS_10, PS_20, PS_40, PS_80, PS_160, PS_320, PS_400")

	cmd.AddCommand(ResizeStatusCmd(ch))
	cmd.AddCommand(ResizeCancelCmd(ch))

	return cmd
}

func toKeyspaceResizeRequest(krr *ps.KeyspaceResizeRequest) *KeyspaceResizeRequest {
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

func (k *KeyspaceResizeRequest) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(k.orig, "", "  ")
}

func (k *KeyspaceResizeRequest) MarshalCSVValue() interface{} {
	return []*KeyspaceResizeRequest{k}
}
