package keyspace

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// CreateCmd encapsulates the command for creating a new keyspace within a branch.
func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	createReq := &planetscale.CreateKeyspaceRequest{}

	var flags struct {
		shards             int
		clusterSize        string
		additionalReplicas int
		customShards       []string
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
			createReq.ExtraReplicas = flags.additionalReplicas
			createReq.ClusterSize = flags.clusterSize

			if len(flags.customShards) > 0 {
				if cmd.Flags().Changed("shards") {
					return errors.New("--shards cannot be combined with --custom-shard")
				}
				customShards, err := parseCustomShards(flags.customShards)
				if err != nil {
					return err
				}
				createReq.Shards = nil
				createReq.CustomShards = customShards
			} else {
				createReq.Shards = &flags.shards
				createReq.CustomShards = nil
			}

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
				k, err = waitUntilReady(ctx, client, ch.Printer, ch.Debug(), getReq)
				if err != nil {
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
	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "", "cluster size for the keyspace. Use `pscale size cluster list` to get a list of valid sizes.")
	cmd.Flags().IntVar(&flags.additionalReplicas, "additional-replicas", 0, "number of additional replicas per shard. By default, each production cluster includes 2 replicas.")
	cmd.Flags().StringArrayVar(&flags.customShards, "custom-shard", nil, customShardFlagHelp)
	cmd.Flags().BoolVar(&flags.wait, "wait", false, "Wait until the keyspace is ready")

	cmd.MarkFlagRequired("cluster-size")

	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return cmdutil.BranchClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}

const customShardFlagHelp = "key range and optional cluster size for one shard as key-range=<range>[,cluster-size=<size>]. Repeatable; requires at least two shards whose ranges cover the whole keyspace. Shards without a cluster-size use --cluster-size. Cannot be combined with --shards."

// parseCustomShards parses repeatable --custom-shard flags.
func parseCustomShards(raw []string) ([]planetscale.CreateKeyspaceCustomShard, error) {
	if len(raw) < 2 {
		return nil, fmt.Errorf("--custom-shard requires at least two shards, got %d", len(raw))
	}
	out := make([]planetscale.CreateKeyspaceCustomShard, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, s := range raw {
		entry, err := parseCustomShard(s)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[entry.KeyRange]; ok {
			return nil, fmt.Errorf("duplicate --custom-shard key-range %q", entry.KeyRange)
		}
		seen[entry.KeyRange] = struct{}{}
		out = append(out, entry)
	}
	return out, nil
}

func parseCustomShard(s string) (planetscale.CreateKeyspaceCustomShard, error) {
	var entry planetscale.CreateKeyspaceCustomShard
	seen := make(map[string]struct{}, 2)
	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			return entry, fmt.Errorf("invalid --custom-shard %q: expected key-range=<range>[,cluster-size=<size>]", s)
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "key-range", "cluster-size":
		default:
			return entry, fmt.Errorf("unknown --custom-shard field %q in %q: valid fields are key-range, cluster-size", k, s)
		}
		if _, dup := seen[k]; dup {
			return entry, fmt.Errorf("duplicate --custom-shard field %q in %q", k, s)
		}
		seen[k] = struct{}{}
		if v == "" {
			return entry, fmt.Errorf("invalid --custom-shard %q: %s must not be empty", s, k)
		}
		if k == "key-range" {
			entry.KeyRange = v
		} else {
			entry.ClusterSize = cmdutil.ToSizeSKUName(v)
		}
	}
	if entry.KeyRange == "" {
		return entry, fmt.Errorf("--custom-shard %q is missing required field key-range", s)
	}
	return entry, nil
}

func waitUntilReady(ctx context.Context, client *planetscale.Client, printer *printer.Printer, debug bool, getReq *planetscale.GetKeyspaceRequest) (*planetscale.Keyspace, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("keyspace creation timed out")
		case <-ticker.C:
			resp, err := client.Keyspaces.Get(ctx, getReq)
			if err != nil {
				if debug {
					printer.Printf("fetching keyspace %s/%s %s failed: %s", getReq.Database, getReq.Branch, getReq.Keyspace, err)
				}
				continue
			}

			if resp.Ready {
				return resp, nil
			}
		}
	}
}
