package branch

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// knownVTGateSizes are the public VTGate SKU names accepted by the API.
var knownVTGateSizes = []string{
	"VTG_DEV", "VTG_5", "VTG_10", "VTG_20", "VTG_40",
	"VTG_80", "VTG_320", "VTG_640", "VTG_1280", "VTG_2560",
}

type branchVTGateResize struct {
	ID                         string `header:"id" json:"id"`
	State                      string `header:"state" json:"state"`
	VTGateSize                 string `header:"vtgate_size" json:"vtgate_name"`
	PreviousVTGateSize         string `header:"previous_vtgate_size" json:"previous_vtgate_name"`
	VTGateCount                int    `header:"vtgate_count" json:"vtgate_count"`
	PreviousVTGateCount        int    `header:"previous_vtgate_count" json:"previous_vtgate_count"`
	VTGateMaxCount             *int   `header:"vtgate_max_count,n/a" json:"vtgate_max_count"`
	VTGateAutoscaling          bool   `header:"vtgate_autoscaling" json:"vtgate_autoscaling"`
	VTGateTargetCPUUtilization *int   `header:"vtgate_target_cpu,n/a" json:"vtgate_target_cpu_utilization"`
	CreatedAt                  int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	StartedAt                  *int64 `header:"started_at,timestamp(ms|utc|human)" json:"started_at"`
	CompletedAt                *int64 `header:"completed_at,timestamp(ms|utc|human)" json:"completed_at"`

	orig *ps.BranchResizeRequest
}

// VtgateCmd manages Vitess branch VTGate configuration.
func VtgateCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vtgate <command>",
		Short: "Manage VTGate size for a Vitess branch",
	}

	cmd.AddCommand(VtgateResizeCmd(ch))
	return cmd
}

// VtgateResizeCmd resizes VTGates on a Vitess production branch.
func VtgateResizeCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		vtgateSize                 string
		vtgateCount                int
		vtgateMaxCount             int
		vtgateAutoscaling          bool
		vtgateTargetCPUUtilization int
	}

	cmd := &cobra.Command{
		Use:   "resize <database> <branch>",
		Short: "Resize VTGates for a Vitess production branch",
		Long: `Resize the VTGate SKU, count, and/or autoscaling for a Vitess production branch.

Development branches cannot be resized. Use "pscale branch vtgate resize status"
to track a resize and "pscale branch vtgate resize cancel" to cancel one while
queued.`,
		Example: `  pscale branch vtgate resize mydb main --vtgate-size VTG_320
  pscale branch vtgate resize mydb main --vtgate-size VTG_1280 --vtgate-count 2
  pscale branch vtgate resize mydb main --vtgate-size VTG_320 --vtgate-autoscaling --vtgate-count 2 --vtgate-max-count 8 --vtgate-target-cpu-utilization 50`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			changedSize := flags.vtgateSize != ""
			changedCount := cmd.Flags().Changed("vtgate-count")
			changedMaxCount := cmd.Flags().Changed("vtgate-max-count")
			changedAutoscaling := cmd.Flags().Changed("vtgate-autoscaling")
			changedTargetCPU := cmd.Flags().Changed("vtgate-target-cpu-utilization")

			if !changedSize && !changedCount && !changedMaxCount && !changedAutoscaling && !changedTargetCPU {
				return errors.New("nothing to change: pass at least one of --vtgate-size, --vtgate-count, --vtgate-max-count, --vtgate-autoscaling, or --vtgate-target-cpu-utilization")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			resizeReq := &ps.ResizeBranchRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			}
			if changedSize {
				resizeReq.VTGateSize = strings.ReplaceAll(flags.vtgateSize, "-", "_")
			}
			if changedCount {
				count := flags.vtgateCount
				resizeReq.VTGateCount = &count
			}
			if changedMaxCount {
				maxCount := flags.vtgateMaxCount
				resizeReq.VTGateMaxCount = &maxCount
			}
			if changedAutoscaling {
				autoscaling := flags.vtgateAutoscaling
				resizeReq.VTGateAutoscaling = &autoscaling
			}
			if changedTargetCPU {
				targetCPU := flags.vtgateTargetCPUUtilization
				resizeReq.VTGateTargetCPUUtilization = &targetCPU
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Requesting VTGate resize for branch %s in %s", printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			resize, err := client.DatabaseBranches.Resize(ctx, resizeReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s or branch %s does not exist in organization %s", printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Successfully requested VTGate resize for branch %s (state: %s).\n", printer.BoldBlue(branch), printer.BoldBlue(resize.State))
				return nil
			}

			return ch.Printer.PrintResource(toBranchVTGateResize(resize))
		},
	}

	cmd.Flags().StringVar(&flags.vtgateSize, "vtgate-size", "", "VTGate size SKU (e.g. VTG_320, VTG_1280)")
	cmd.Flags().IntVar(&flags.vtgateCount, "vtgate-count", 0, "Number of VTGates per availability zone (minimum when autoscaling is enabled)")
	cmd.Flags().IntVar(&flags.vtgateMaxCount, "vtgate-max-count", 0, "Maximum VTGates per availability zone when autoscaling is enabled")
	cmd.Flags().BoolVar(&flags.vtgateAutoscaling, "vtgate-autoscaling", false, "Enable or disable VTGate autoscaling (use --vtgate-autoscaling=false to disable)")
	cmd.Flags().IntVar(&flags.vtgateTargetCPUUtilization, "vtgate-target-cpu-utilization", 0, "Target CPU utilization percent when autoscaling is enabled")

	cmd.RegisterFlagCompletionFunc("vtgate-size", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		completions := make([]cobra.Completion, 0, len(knownVTGateSizes))
		upper := strings.ToUpper(strings.ReplaceAll(toComplete, "-", "_"))
		for _, size := range knownVTGateSizes {
			if strings.Contains(size, upper) {
				completions = append(completions, size)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
	})

	cmd.AddCommand(VtgateResizeStatusCmd(ch))
	cmd.AddCommand(VtgateResizeCancelCmd(ch))

	return cmd
}

func VtgateResizeStatusCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <database> <branch>",
		Short: "Show the latest VTGate resize for a Vitess branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching VTGate resize status for branch %s in %s", printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			resize, err := client.DatabaseBranches.ResizeStatus(ctx, &ps.BranchResizeStatusRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("no VTGate resize found for database %s branch %s in organization %s", printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toBranchVTGateResize(resize))
		},
	}

	return cmd
}

func VtgateResizeCancelCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel <database> <branch>",
		Short: "Cancel a queued VTGate resize for a Vitess branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Canceling VTGate resize for branch %s in %s", printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			err = client.DatabaseBranches.CancelResize(ctx, &ps.CancelBranchResizeRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
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

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Canceled VTGate resize for branch %s.\n", printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result": "canceled",
				"branch": branch,
			})
		},
	}

	return cmd
}

func toBranchVTGateResize(r *ps.BranchResizeRequest) *branchVTGateResize {
	return &branchVTGateResize{
		ID:                         r.ID,
		State:                      r.State,
		VTGateSize:                 r.VTGateName,
		PreviousVTGateSize:         r.PreviousVTGateName,
		VTGateCount:                r.VTGateCount,
		PreviousVTGateCount:        r.PreviousVTGateCount,
		VTGateMaxCount:             r.VTGateMaxCount,
		VTGateAutoscaling:          r.VTGateAutoscaling,
		VTGateTargetCPUUtilization: r.VTGateTargetCPUUtilization,
		CreatedAt:                  cmdutil.TimeToMilliseconds(r.CreatedAt),
		StartedAt:                  printer.GetMillisecondsIfExists(r.StartedAt),
		CompletedAt:                printer.GetMillisecondsIfExists(r.CompletedAt),
		orig:                       r,
	}
}

func (r *branchVTGateResize) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(r.orig, "", "  ")
}

func (r *branchVTGateResize) MarshalCSVValue() interface{} {
	return []*branchVTGateResize{r}
}
