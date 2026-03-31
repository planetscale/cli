package vtctld

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

var (
	plannedReparentOperationPollInterval   = time.Second
	plannedReparentOperationTimeoutBuffer  = 30 * time.Second
	plannedReparentOperationDefaultTimeout = 10 * time.Minute
)

func PlannedReparentShardCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace   string
		shard      string
		newPrimary string
		wait       bool
		id         string
	}

	cmd := &cobra.Command{
		Use:   "planned-reparent-shard <database> <branch>",
		Short: "Reparent a shard to a new primary, or check on an existing reparent operation",
		Long: `Reparent a shard to a new primary using Vitess PlannedReparentShard.
Both the old and new primaries must be up and running.

To execute a planned reparent:
  pscale branch vtctld planned-reparent-shard <db> <branch> \
    --keyspace <ks> --shard <shard> --new-primary <tablet-alias>

To check on an existing operation:
  pscale branch vtctld planned-reparent-shard <db> <branch> --id <operation-id>`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if flags.id != "" {
				return getPlannedReparentOperation(ctx, ch, client, database, branch, flags.id)
			}

			if flags.keyspace == "" || flags.shard == "" || flags.newPrimary == "" {
				return fmt.Errorf("--keyspace, --shard, and --new-primary are required when not using --id")
			}

			return runPlannedReparentShard(ctx, ch, client, database, branch, flags.keyspace, flags.shard, flags.newPrimary, flags.wait)
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace name")
	cmd.Flags().StringVar(&flags.shard, "shard", "", "Shard range (e.g., '-80', '80-', or '-' for unsharded)")
	cmd.Flags().StringVar(&flags.newPrimary, "new-primary", "", "Tablet alias to promote as the new primary")
	cmd.Flags().BoolVar(&flags.wait, "wait", true, "Wait for the operation to complete")
	cmd.Flags().StringVar(&flags.id, "id", "", "Check status of an existing planned reparent operation")

	return cmd
}

func runPlannedReparentShard(ctx context.Context, ch *cmdutil.Helper, client *ps.Client, database, branch, keyspace, shard, newPrimary string, wait bool) error {
	end := ch.Printer.PrintProgress(
		fmt.Sprintf("Executing PlannedReparentShard on %s\u2026",
			progressTarget(ch.Config.Organization, database, branch)))
	defer end()

	operation, err := client.PlannedReparentShard.Create(ctx, &ps.PlannedReparentShardRequest{
		Organization: ch.Config.Organization,
		Database:     database,
		Branch:       branch,
		Keyspace:     keyspace,
		Shard:        shard,
		NewPrimary:   newPrimary,
	})
	if err != nil {
		return cmdutil.HandleError(err)
	}

	if !wait {
		end()
		return ch.Printer.PrintJSON(map[string]string{"id": operation.ID})
	}

	result, err := waitForPlannedReparentResult(ctx, client, ch.Config.Organization, database, branch, operation)
	if err != nil {
		return cmdutil.HandleError(err)
	}

	end()
	return ch.Printer.PrettyPrintJSON(result)
}

func getPlannedReparentOperation(ctx context.Context, ch *cmdutil.Helper, client *ps.Client, database, branch, id string) error {
	end := ch.Printer.PrintProgress(
		fmt.Sprintf("Getting PlannedReparentShard operation on %s\u2026",
			progressTarget(ch.Config.Organization, database, branch)))
	defer end()

	operation, err := client.PlannedReparentShard.Get(ctx, &ps.GetPlannedReparentShardRequest{
		Organization: ch.Config.Organization,
		Database:     database,
		Branch:       branch,
		ID:           id,
	})
	if err != nil {
		return cmdutil.HandleError(err)
	}

	end()
	return ch.Printer.PrintJSON(operation)
}

func waitForPlannedReparentResult(ctx context.Context, client *ps.Client, organization, database, branch string, operation *ps.VtctldOperation) (json.RawMessage, error) {
	result, done, err := plannedReparentOperationResult(operation)
	if done || err != nil {
		return result, err
	}

	request := &ps.GetPlannedReparentShardRequest{
		Organization: organization,
		Database:     database,
		Branch:       branch,
		ID:           operation.ID,
	}

	pollCtx, cancel := context.WithTimeout(ctx, plannedReparentOperationTimeout(operation))
	defer cancel()
	ticker := time.NewTicker(plannedReparentOperationPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pollCtx.Done():
			if errors.Is(pollCtx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("timed out waiting for planned reparent operation %s to finish", operation.ID)
			}

			return nil, pollCtx.Err()
		case <-ticker.C:
		}

		op, err := client.PlannedReparentShard.Get(pollCtx, request)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return nil, fmt.Errorf("timed out waiting for planned reparent operation %s to finish", operation.ID)
			}

			return nil, err
		}

		result, done, err = plannedReparentOperationResult(op)
		if done || err != nil {
			return result, err
		}
	}
}

func plannedReparentOperationResult(operation *ps.VtctldOperation) (json.RawMessage, bool, error) {
	if !operation.Completed {
		return nil, false, nil
	}

	switch operation.State {
	case "completed":
		if len(operation.Result) == 0 {
			return json.RawMessage(`{}`), true, nil
		}

		return operation.Result, true, nil
	case "failed", "cancelled":
		if operation.Error != "" {
			return nil, true, errors.New(operation.Error)
		}

		return nil, true, fmt.Errorf("planned reparent operation %s ended in state %q", operation.ID, operation.State)
	default:
		return nil, true, fmt.Errorf("planned reparent operation %s reached unexpected terminal state %q", operation.ID, operation.State)
	}
}

func plannedReparentOperationTimeout(operation *ps.VtctldOperation) time.Duration {
	if operation.Timeout > 0 {
		return time.Duration(operation.Timeout)*time.Second + plannedReparentOperationTimeoutBuffer
	}

	return plannedReparentOperationDefaultTimeout
}
