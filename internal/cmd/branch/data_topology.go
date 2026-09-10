package branch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func DataTopologyCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "data-topology <command>",
		Short: "Fetch or update the data topology of a Neki branch",
	}

	cmd.AddCommand(GetDataTopologyCmd(ch))
	cmd.AddCommand(ListDataTopologyCmd(ch))
	cmd.AddCommand(UpdateDataTopologyCmd(ch))

	return cmd
}

func ListDataTopologyCmd(ch *cmdutil.Helper) *cobra.Command {
	var shards bool

	cmd := &cobra.Command{
		Use:   "ls <database> <branch>",
		Short: "Show the relationships in a Neki branch's data topology",
		Example: `  pscale branch data-topology ls mydb main --org my-org --format json
  pscale branch data-topology ls mydb main --org my-org --shards --format json`,
		Args: cmdutil.ExactArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			dataTopology, err := client.DatabaseBranches.DataTopology(ctx, &planetscale.BranchDataTopologyRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return handleDataTopologyError(err, database, branch, ch.Config.Organization)
			}

			switch ch.Printer.Format() {
			case printer.Human:
				var tree *dataTopologyTreeOutput
				if shards {
					tree, err = buildDataTopologyShardsTree(ch.Config.Organization, database, branch, dataTopology)
				} else {
					tree, err = buildDataTopologyTree(ch.Config.Organization, database, branch, dataTopology)
				}
				if err != nil {
					return err
				}
				ch.Printer.Print(renderDataTopologyTree(tree.Tree))
				return nil
			case printer.JSON:
				if shards {
					output, err := buildDataTopologyShardsOutput(ch.Config.Organization, database, branch, dataTopology)
					if err != nil {
						return err
					}
					return ch.Printer.PrintResource(output)
				}
				output, err := buildDataTopologyListOutput(ch.Config.Organization, database, branch, dataTopology)
				if err != nil {
					return err
				}
				return ch.Printer.PrintResource(output)
			default:
				return errors.New("data-topology ls supports human and json output")
			}
		},
	}

	cmd.Flags().BoolVar(&shards, "shards", false, "Group data by the physical shards that host it")

	return cmd
}

func GetDataTopologyCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "get <database> <branch>",
		Short: "Show the data topology of a Neki branch",
		Args:  cmdutil.ExactArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			dataTopology, err := client.DatabaseBranches.DataTopology(ctx, &planetscale.BranchDataTopologyRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return handleDataTopologyError(err, database, branch, ch.Config.Organization)
			}

			return printDataTopology(ch, dataTopology)
		},
	}
}

func UpdateDataTopologyCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:     "update <database> <branch>",
		Short:   "Update the data topology of a Neki branch from standard input",
		Example: `  pscale branch data-topology update mydb main --org my-org --format json < data-topology.json`,
		Args:    cmdutil.ExactArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			data, err := readDataTopology(cmd)
			if err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			dataTopology, err := client.DatabaseBranches.UpdateDataTopology(ctx, &planetscale.UpdateBranchDataTopologyRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				DataTopology: data,
			})
			if err != nil {
				return handleDataTopologyError(err, database, branch, ch.Config.Organization)
			}

			return printDataTopology(ch, dataTopology)
		},
	}
}

func readDataTopology(cmd *cobra.Command) (json.RawMessage, error) {
	stdin := cmd.InOrStdin()
	if stdin == os.Stdin {
		stdinFile, err := os.Stdin.Stat()
		if err != nil {
			return nil, err
		}
		if (stdinFile.Mode() & os.ModeCharDevice) != 0 {
			return nil, errors.New("no data topology provided; pipe a JSON object to standard input")
		}
	}

	data, err := io.ReadAll(stdin)
	if err != nil {
		return nil, err
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, errors.New("no data topology provided; pipe a JSON object to standard input")
	}

	var topology map[string]json.RawMessage
	if err := json.Unmarshal(data, &topology); err != nil {
		return nil, fmt.Errorf("invalid data topology JSON: %w", err)
	}
	if topology == nil {
		return nil, errors.New("invalid data topology JSON: expected an object")
	}

	return json.RawMessage(data), nil
}

func handleDataTopologyError(err error, database, branch, organization string) error {
	if cmdutil.ErrCode(err) == planetscale.ErrNotFound {
		return fmt.Errorf("branch %s does not exist or is not a Neki branch in database %s (organization: %s)",
			printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(organization))
	}

	return cmdutil.HandleError(err)
}

func printDataTopology(ch *cmdutil.Helper, dataTopology *planetscale.DataTopology) error {
	if ch.Printer.Format() != printer.Human {
		return ch.Printer.PrintResource(dataTopology)
	}

	if err := ch.Printer.PrettyPrintJSON(dataTopology.DataTopology); err != nil {
		return fmt.Errorf("reading data topology: %w", err)
	}

	return nil
}
