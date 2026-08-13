package deployrequest

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

// OperationsCmd lists schema operations for a deploy request.
func OperationsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operations <database> <number>",
		Short: "List deploy operations for a deploy request",
		Args:  cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			numberStr := args[1]

			number, err := strconv.ParseUint(numberStr, 10, 64)
			if err != nil {
				return fmt.Errorf("the argument <number> is invalid: %s", err)
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching operations for deploy request %s/%s",
				printer.BoldBlue(database), printer.BoldBlue(number)))
			defer end()

			ops, err := client.DeployRequests.GetDeployOperations(ctx, &planetscale.GetDeployOperationsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Number:       number,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("deploy request '%s/%s' does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(number), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(ops) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No deploy operations for %s/%s.\n", printer.BoldBlue(database), printer.BoldBlue(number))
				return nil
			}

			return ch.Printer.PrintResource(toDeployOperations(ops))
		},
	}

	return cmd
}

type DeployOperationRow struct {
	ID                 string `header:"id" json:"id"`
	State              string `header:"state" json:"state"`
	Keyspace           string `header:"keyspace" json:"keyspace_name"`
	Table              string `header:"table" json:"table_name"`
	Operation          string `header:"operation" json:"operation_name"`
	ProgressPercentage uint64 `header:"progress_%" json:"progress_percentage"`
	ETASeconds         int64  `header:"eta_seconds" json:"eta_seconds"`

	orig *planetscale.DeployOperation
}

func (d *DeployOperationRow) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.orig, "", "  ")
}

func (d *DeployOperationRow) MarshalCSVValue() interface{} {
	return []*DeployOperationRow{d}
}

func toDeployOperation(op *planetscale.DeployOperation) *DeployOperationRow {
	return &DeployOperationRow{
		ID:                 op.ID,
		State:              op.State,
		Keyspace:           op.Keyspace,
		Table:              op.Table,
		Operation:          op.Operation,
		ProgressPercentage: op.ProgressPercentage,
		ETASeconds:         op.ETASeconds,
		orig:               op,
	}
}

func toDeployOperations(ops []*planetscale.DeployOperation) []*DeployOperationRow {
	rows := make([]*DeployOperationRow, 0, len(ops))
	for _, op := range ops {
		rows = append(rows, toDeployOperation(op))
	}
	return rows
}
