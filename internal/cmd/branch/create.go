package branch

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	createReq := &ps.CreateDatabaseBranchRequest{}

	var flags struct {
		wait          bool
		dataBranching bool
	}

	cmd := &cobra.Command{
		Use:     "create <source-database> <branch> [options]",
		Short:   "Create a new branch from a database",
		Args:    cmdutil.RequiredArgs("source-database", "branch"),
		Aliases: []string{"b"},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			client, err := ch.Client()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			org := ch.Config.Organization // --org flag
			if org == "" {
				cfg, err := ch.ConfigFS.DefaultConfig()
				if err != nil {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}

				org = cfg.Organization
			}

			databases, err := client.Databases.List(cmd.Context(), &ps.ListDatabasesRequest{
				Organization: org,
			})
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			candidates := make([]string, 0, len(databases))
			for _, db := range databases {
				candidates = append(candidates, db.Name)
			}

			return candidates, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			source := args[0]
			branch := args[1]

			if flags.dataBranching {
				createReq.SeedData = "last_successful_backup"
			}

			createReq.Database = source
			createReq.Name = branch
			createReq.Organization = ch.Config.Organization

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating branch from %s...", printer.BoldBlue(source)))
			defer end()
			dbBranch, err := client.DatabaseBranches.Create(cmd.Context(), createReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					if createReq.ParentBranch != "" {
						return cmdutil.HandleNotFoundWithServiceTokenCheck(
							ctx, cmd, ch.Config, ch.Client, err, "create_branch",
							"source branch %s or database %s does not exist (organization: %s)",
							printer.BoldBlue(createReq.ParentBranch), printer.BoldBlue(source), printer.BoldBlue(ch.Config.Organization))
					} else {
						return cmdutil.HandleNotFoundWithServiceTokenCheck(
							ctx, cmd, ch.Config, ch.Client, err, "create_branch",
							"source database %s does not exist in organization %s",
							printer.BoldBlue(source), printer.BoldBlue(ch.Config.Organization))
					}
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			// wait and check until the DB is ready
			if flags.wait {
				end := ch.Printer.PrintProgress(fmt.Sprintf("Waiting until branch %s is ready...", printer.BoldBlue(branch)))
				defer end()
				getReq := &ps.GetDatabaseBranchRequest{
					Organization: ch.Config.Organization,
					Database:     source,
					Branch:       branch,
				}
				if err := waitUntilReady(ctx, client, ch.Printer, ch.Debug(), getReq); err != nil {
					return err
				}
				end()
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Branch %s was successfully created.\n\nView this branch in the browser: %s\n", printer.BoldBlue(dbBranch.Name), printer.BoldBlue(dbBranch.HtmlURL))
				return nil
			}

			return ch.Printer.PrintResource(ToDatabaseBranch(dbBranch))
		},
	}

	cmd.Flags().StringVar(&createReq.ParentBranch, "from", "", "Parent branch to create the new branch from. Cannot be used with --restore")
	cmd.Flags().StringVar(&createReq.Region, "region", "", "Region for the branch to be created in.")
	cmd.Flags().StringVar(&createReq.BackupID, "restore", "", "ID of Backup to restore into branch.")
	cmd.Flags().StringVar(&createReq.ClusterSize, "cluster-size", "PS-10", "Cluster size for branches being created from a backup or seeded with data. Use 'pscale size cluster list' to see the valid sizes.")
	cmd.Flags().BoolVar(&flags.dataBranching, "seed-data", false, "Add seed data using the Data Branching™ feature. This branch will be created with the same resources as the base branch.")
	cmd.Flags().BoolVar(&flags.wait, "wait", false, "Wait until the branch is ready")
	cmd.MarkFlagsMutuallyExclusive("from", "restore")
	cmd.MarkFlagsMutuallyExclusive("restore", "seed-data")

	cmd.RegisterFlagCompletionFunc("region", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.RegionsCompletionFunc(ch, cmd, args, toComplete)
	})

	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.ClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}

// waitUntilReady waits until the given database branch is ready. It times out after 3 minutes.
func waitUntilReady(ctx context.Context, client *ps.Client, printer *printer.Printer, debug bool, getReq *ps.GetDatabaseBranchRequest) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ctx.Done():
			return errors.New("branch creation timed out")
		case <-ticker.C:
			resp, err := client.DatabaseBranches.Get(ctx, getReq)
			if err != nil {
				if debug {
					printer.Printf("fetching database branch %s/%s failed: %s", getReq.Database, getReq.Branch, err)
				}
				continue
			}

			if resp.Ready {
				return nil
			}
		}
	}
}
