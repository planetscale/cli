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

// StorageCheckCmd checks whether a deploy request has enough storage.
func StorageCheckCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage-check <database> <number>",
		Short: "Check storage readiness for a deploy request",
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

			end := ch.Printer.PrintProgress(fmt.Sprintf("Checking storage for deploy request %s/%s",
				printer.BoldBlue(database), printer.BoldBlue(number)))
			defer end()

			check, err := client.DeployRequests.CheckStorage(ctx, &planetscale.CheckDeployRequestStorageRequest{
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

			return ch.Printer.PrintResource(toStorageCheck(check))
		},
	}

	return cmd
}

type StorageCheckRow struct {
	EnoughStorage      bool  `header:"enough_storage" json:"enough_storage"`
	Upgradeable        bool  `header:"upgradeable" json:"upgradeable"`
	StorageBytesNeeded int64 `header:"storage_bytes_needed" json:"storage_bytes_needed"`

	orig *planetscale.DeployRequestStorageCheck
}

func (s *StorageCheckRow) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(s.orig, "", "  ")
}

func (s *StorageCheckRow) MarshalCSVValue() interface{} {
	return []*StorageCheckRow{s}
}

func toStorageCheck(check *planetscale.DeployRequestStorageCheck) *StorageCheckRow {
	return &StorageCheckRow{
		EnoughStorage:      check.EnoughStorage,
		Upgradeable:        check.Upgradeable,
		StorageBytesNeeded: check.StorageBytesNeeded,
		orig:               check,
	}
}
