package keyspace

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func UpdateSettingsCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		replicationDurabilityConstraints *ps.ReplicationDurabilityConstraints
		vreplicationFlags                *ps.VReplicationFlags
		maxRollout                       int
		resetMaxRollout                  bool
		throttlerEnabled                 bool
		throttlerThreshold               float64
		interactive                      bool
	}

	flags.replicationDurabilityConstraints = &ps.ReplicationDurabilityConstraints{}
	flags.vreplicationFlags = &ps.VReplicationFlags{}

	cmd := &cobra.Command{
		Use:   "update-settings <database> <branch> <keyspace>",
		Short: "Update the settings for a keyspace",
		Args:  cmdutil.RequiredArgs("database", "branch", "keyspace"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, keyspace := args[0], args[1], args[2]
			maxRolloutChanged := cmd.Flags().Changed("max-rollout")
			resetMaxRolloutChanged := cmd.Flags().Changed("reset-max-rollout")
			resetMaxRolloutRequested := resetMaxRolloutChanged && flags.resetMaxRollout

			if maxRolloutChanged && resetMaxRolloutRequested {
				return fmt.Errorf("--max-rollout and --reset-max-rollout are mutually exclusive")
			}
			if flags.interactive && (maxRolloutChanged || resetMaxRolloutChanged) {
				return fmt.Errorf("--max-rollout and --reset-max-rollout cannot be used with --interactive")
			}
			if maxRolloutChanged && (flags.maxRollout < 1 || flags.maxRollout > 32) {
				return fmt.Errorf("--max-rollout must be between 1 and 32")
			}
			if cmd.Flags().Changed("throttler-threshold") && flags.throttlerThreshold < 0 {
				return errors.New("--throttler-threshold must be greater than or equal to 0")
			}

			updateReq := &ps.UpdateKeyspaceSettingsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     keyspace,
			}

			if flags.interactive {
				return updateInteractive(ctx, ch, updateReq)
			}

			// Nested VReplication and throttler updates read current settings
			// first so unspecified flags in that group can be preserved.
			rdcChanged := cmd.Flags().Changed("replication-durability-constraints-strategy")
			vrfChanged := cmd.Flags().Changed("vreplication-optimize-inserts") ||
				cmd.Flags().Changed("vreplication-enable-noblob-binlog-mode") ||
				cmd.Flags().Changed("vreplication-batch-replication-events")
			throttlerChanged := cmd.Flags().Changed("throttler-enabled") ||
				cmd.Flags().Changed("throttler-threshold")

			if !rdcChanged && !vrfChanged && !throttlerChanged && !maxRolloutChanged && !resetMaxRolloutRequested {
				ch.Printer.Println("No changes were requested. No update performed.")
				return nil
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating settings for keyspace %s in %s/%s", printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			if rdcChanged {
				updateReq.ReplicationDurabilityConstraints = &ps.ReplicationDurabilityConstraints{
					Strategy: constraintsToStrategy(flags.replicationDurabilityConstraints.Strategy),
				}
			}

			if vrfChanged || throttlerChanged {
				if err := setInitialSettings(ctx, client, updateReq, false, vrfChanged, throttlerChanged); err != nil {
					return err
				}
			}

			if vrfChanged {
				if updateReq.VReplicationFlags == nil {
					updateReq.VReplicationFlags = &ps.VReplicationFlags{}
				}

				if cmd.Flags().Changed("vreplication-optimize-inserts") {
					updateReq.VReplicationFlags.OptimizeInserts = flags.vreplicationFlags.OptimizeInserts
				}

				if cmd.Flags().Changed("vreplication-enable-noblob-binlog-mode") {
					updateReq.VReplicationFlags.AllowNoBlobBinlogRowImage = flags.vreplicationFlags.AllowNoBlobBinlogRowImage
				}

				if cmd.Flags().Changed("vreplication-batch-replication-events") {
					updateReq.VReplicationFlags.VPlayerBatching = flags.vreplicationFlags.VPlayerBatching
				}
			}

			if throttlerChanged {
				if updateReq.Throttler == nil {
					updateReq.Throttler = &ps.KeyspaceThrottler{}
				}

				if cmd.Flags().Changed("throttler-enabled") {
					updateReq.Throttler.Enabled = &flags.throttlerEnabled
				}

				if cmd.Flags().Changed("throttler-threshold") {
					updateReq.Throttler.Threshold = &flags.throttlerThreshold
				}
			}

			if maxRolloutChanged {
				maxRollout := &flags.maxRollout
				updateReq.MaxRollout = &maxRollout
			} else if resetMaxRolloutRequested {
				var maxRollout *int
				updateReq.MaxRollout = &maxRollout
			}

			k, err := updateKeyspaceSettings(ctx, client, updateReq)
			if err != nil {
				return err
			}

			end()

			return ch.Printer.PrintResource(toKeyspaceSettings(k))
		},
	}

	cmd.Flags().StringVar(&flags.replicationDurabilityConstraints.Strategy, "replication-durability-constraints-strategy", "maximum", "By default, replication is configured to maximize safety and data integrity. This setting may be relaxed to favor increased performance and reduced replication lag. Options: maximum, dynamic, minimum")
	cmd.Flags().BoolVar(&flags.vreplicationFlags.OptimizeInserts, "vreplication-optimize-inserts", true, "When enabled, skips sending INSERT events for rows that have yet to be replicated.")
	cmd.Flags().BoolVar(&flags.vreplicationFlags.AllowNoBlobBinlogRowImage, "vreplication-enable-noblob-binlog-mode", true, "When enabled, omits changed BLOB and TEXT columns from replication events, which reduces binlog sizes.")
	cmd.Flags().BoolVar(&flags.vreplicationFlags.VPlayerBatching, "vreplication-batch-replication-events", false, "When enabled, sends fewer queries to MySQL to improve performance.")
	cmd.Flags().IntVar(&flags.maxRollout, "max-rollout", 1, "Maximum number of concurrent shard rollouts (1-32). The effective service cap is 32.")
	cmd.Flags().BoolVar(&flags.resetMaxRollout, "reset-max-rollout", false, "Reset the configured maximum concurrent shard rollouts to the default (1).")
	cmd.Flags().BoolVar(&flags.throttlerEnabled, "throttler-enabled", true, "Pause schema migrations and VReplication workflows when replication lag rises above the threshold.")
	cmd.Flags().Float64Var(&flags.throttlerThreshold, "throttler-threshold", 5, "Replication lag in seconds above which migrations and workflows are paused.")
	cmd.Flags().BoolVarP(&flags.interactive, "interactive", "i", false, "Run the command in interactive mode")

	return cmd
}

func setInitialSettings(ctx context.Context, client *ps.Client, req *ps.UpdateKeyspaceSettingsRequest, includeDurability, includeVReplication, includeThrottler bool) error {
	organization := req.Organization
	database := req.Database
	branch := req.Branch
	keyspace := req.Keyspace

	ks, err := client.Keyspaces.Get(ctx, &ps.GetKeyspaceRequest{
		Organization: organization,
		Database:     database,
		Branch:       branch,
		Keyspace:     keyspace,
	})
	if err != nil {
		switch cmdutil.ErrCode(err) {
		case ps.ErrNotFound:
			return fmt.Errorf("keyspace %s does not exist in branch %s (database: %s, organization: %s)", printer.BoldBlue(keyspace), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(organization))
		default:
			return cmdutil.HandleError(err)
		}
	}

	if includeDurability && ks.ReplicationDurabilityConstraints != nil {
		req.ReplicationDurabilityConstraints = ks.ReplicationDurabilityConstraints
	}

	if includeVReplication && ks.VReplicationFlags != nil {
		vreplicationFlags := *ks.VReplicationFlags
		req.VReplicationFlags = &vreplicationFlags
	}

	if includeThrottler && ks.Throttler != nil {
		throttler := *ks.Throttler
		req.Throttler = &throttler
	}

	return nil
}

func updateInteractive(ctx context.Context, ch *cmdutil.Helper, updateReq *ps.UpdateKeyspaceSettingsRequest) error {
	client, err := ch.Client()
	if err != nil {
		return err
	}

	if err := setInitialSettings(ctx, client, updateReq, true, true, true); err != nil {
		return err
	}

	if updateReq.ReplicationDurabilityConstraints == nil {
		updateReq.ReplicationDurabilityConstraints = &ps.ReplicationDurabilityConstraints{}
	}

	if updateReq.VReplicationFlags == nil {
		updateReq.VReplicationFlags = &ps.VReplicationFlags{}
	}

	if updateReq.Throttler == nil {
		updateReq.Throttler = &ps.KeyspaceThrottler{}
	}

	throttlerEnabled := true
	if updateReq.Throttler.Enabled != nil {
		throttlerEnabled = *updateReq.Throttler.Enabled
	}

	throttlerThreshold := "5"
	if updateReq.Throttler.Threshold != nil {
		throttlerThreshold = strconv.FormatFloat(*updateReq.Throttler.Threshold, 'g', -1, 64)
	}

	form := huh.NewForm(
		// Replication Durability Constraints
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Replication durability constraints").
				Description("By default, replication is configured to maximize safety and data integrity. This setting may be relaxed to favor increased performance and reduced replication lag.").
				Value(&updateReq.ReplicationDurabilityConstraints.Strategy).
				Options(
					huh.NewOption("Maximum - Uses the highest durability constraints to maximize safety and data integrity.", constraintsToStrategy("maximum")),
					huh.NewOption("Dynamic - Uses maximum durability constraints when replication lag is less than 5 seconds. Automatically reduces durability settings when replication exceeds 5 seconds.", constraintsToStrategy("dynamic")),
					huh.NewOption("Minimum - Optimizes for low lag and replica performance, but has the highest risk of data loss on crashed instances.", constraintsToStrategy("minimum")),
				),
		),

		// VReplication Flags
		huh.NewGroup(
			huh.NewNote().Title("VReplication").
				Description("Options for improving performance during deploy requests and workflows."),

			huh.NewConfirm().
				Title("Skip INSERT events during replication catch-up?").
				Description("When enabled, skips sending INSERT events for rows that have yet to be replicated.").
				Value(&updateReq.VReplicationFlags.OptimizeInserts),

			huh.NewConfirm().
				Title("Enable NOBLOB binlog mode?").
				Description("When enabled, omits changed BLOB and TEXT columns from replication events, which reduces binlog sizes.").
				Value(&updateReq.VReplicationFlags.AllowNoBlobBinlogRowImage),

			huh.NewConfirm().
				Title("Batch VPlayer events?").
				Description("When enabled, sends fewer queries to MySQL to improve performance.").
				Value(&updateReq.VReplicationFlags.VPlayerBatching),
		).Title("VReplication").Description("Options for improving performance during deploy requests and workflows"),

		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable the throttler?").
				Description("Pauses schema migrations and VReplication workflows when replication lag rises above the threshold.").
				Value(&throttlerEnabled),

			huh.NewInput().
				Title("Replication lag threshold (seconds)").
				Description("Migrations and workflows are paused while replication lag is above this value.").
				Value(&throttlerThreshold).
				Validate(func(s string) error {
					v, err := strconv.ParseFloat(s, 64)
					if err != nil {
						return errors.New("threshold must be a number")
					}
					if v < 0 {
						return errors.New("threshold must be greater than or equal to 0")
					}
					return nil
				}),
		).Title("Throttler"),
	).WithTheme(huh.ThemeBase16())

	if err := form.Run(); err != nil {
		return err
	}

	threshold, err := strconv.ParseFloat(throttlerThreshold, 64)
	if err != nil {
		return err
	}
	updateReq.Throttler.Enabled = &throttlerEnabled
	updateReq.Throttler.Threshold = &threshold

	ks, err := updateKeyspaceSettings(ctx, client, updateReq)
	if err != nil {
		return err
	}

	ch.Printer.Printf("Updated settings for keyspace %s in %s/%s\n", printer.BoldBlue(ks.Name), printer.BoldBlue(updateReq.Database), printer.BoldBlue(updateReq.Branch))

	return nil
}

func updateKeyspaceSettings(ctx context.Context, client *ps.Client, updateReq *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
	k, err := client.Keyspaces.UpdateSettings(ctx, updateReq)
	if err != nil {
		switch cmdutil.ErrCode(err) {
		case ps.ErrNotFound:
			return nil, fmt.Errorf("keyspace %s does not exist in branch %s (database: %s, organization: %s)", printer.BoldBlue(updateReq.Keyspace), printer.BoldBlue(updateReq.Branch), printer.BoldBlue(updateReq.Database), printer.BoldBlue(updateReq.Organization))
		default:
			return nil, cmdutil.HandleError(err)
		}
	}

	return k, nil
}

// Helper function for translating the API value to a more semantic string.
func constraintsToStrategy(constraint string) string {
	switch constraint {
	case "maximum":
		return "available"
	case "minimum":
		return "always"
	case "dynamic":
		return "lag"
	default:
		return constraint
	}
}
