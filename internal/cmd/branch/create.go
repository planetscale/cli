package branch

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		wait          bool
		dataBranching bool
		region        string
		parentBranch  string
		clusterSize   string
		backupID      string
		restorePoint  string
		majorVersion  string
		replicas      int
		minStorage    int64
		maxStorage    int64
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

			if flags.backupID != "" && flags.parentBranch != "" && flags.restorePoint == "" {
				return fmt.Errorf("--from and --restore cannot be used together")
			}
			if cmd.Flags().Changed("replicas") && flags.backupID == "" && flags.restorePoint == "" {
				return fmt.Errorf("--replicas can only be used with a PostgreSQL backup restore or point-in-time recovery")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating branch from %s...", printer.BoldBlue(source)))
			defer end()

			db, err := client.Databases.Get(ctx, &ps.GetDatabaseRequest{
				Organization: ch.Config.Organization,
				Database:     source,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					if _, orgErr := client.Organizations.Get(ctx, &ps.GetOrganizationRequest{
						Organization: ch.Config.Organization,
					}); cmdutil.ErrCode(orgErr) == ps.ErrNotFound {
						return cmdutil.HandleNotFoundWithServiceTokenCheck(
							ctx, cmd, ch.Config, ch.Client, err, "read_branch",
							"organization %s does not exist",
							printer.BoldBlue(ch.Config.Organization))
					}
					return cmdutil.HandleNotFoundWithServiceTokenCheck(
						ctx, cmd, ch.Config, ch.Client, err, "read_branch",
						"source database %s does not exist in organization %s",
						printer.BoldBlue(source), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			clusterSize := flags.clusterSize
			if clusterSize == "" {
				if flags.backupID != "" || flags.restorePoint != "" || flags.dataBranching {
					clusterSize = "PS-10"
				} else {
					clusterSize = "PS_DEV"
				}
			}

			if db.Kind == "mysql" {
				if cmd.Flags().Changed("replicas") {
					return fmt.Errorf("--replicas is only supported for PostgreSQL backup restores and point-in-time recovery")
				}
				if cmd.Flags().Changed("min-storage") || cmd.Flags().Changed("max-storage") {
					return fmt.Errorf("--min-storage and --max-storage are only supported for PostgreSQL databases")
				}
				if flags.restorePoint != "" {
					return fmt.Errorf("--restore-point is only supported for PostgreSQL databases")
				}

				createReq := &ps.CreateDatabaseBranchRequest{
					Organization: ch.Config.Organization,
					Database:     source,
					Name:         branch,
					Region:       flags.region,
					ClusterSize:  clusterSize,
					ParentBranch: flags.parentBranch,
					BackupID:     flags.backupID,
				}

				if flags.dataBranching {
					createReq.SeedData = "last_successful_backup"
				}

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
					dbBranch, err = waitUntilReady(ctx, client, ch.Printer, ch.Debug(), getReq)
					if err != nil {
						return err
					}
					end()
				}
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("Branch %s was successfully created.\n\nView this branch in the browser: %s\n", printer.BoldBlue(dbBranch.Name), printer.BoldBlue(dbBranch.HtmlURL))
					return nil
				}

				return ch.Printer.PrintResource(ToDatabaseBranch(dbBranch))
			} else {
				if flags.restorePoint != "" {
					if flags.backupID == "" {
						if flags.parentBranch == "" {
							return fmt.Errorf("--from is required when using --restore-point without --restore")
						}

						backupID, err := cmdutil.BackupIDForRestorePoint(ctx, client, ch.Config.Organization, source, flags.parentBranch, flags.restorePoint)
						if err != nil {
							return err
						}
						flags.backupID = backupID
					}
				}

				createReq := &ps.CreatePostgresBranchRequest{
					Organization: ch.Config.Organization,
					Database:     source,
					Name:         branch,
					Region:       flags.region,
					ClusterName:  clusterSize,
					ParentBranch: flags.parentBranch,
					BackupID:     flags.backupID,
					RestorePoint: flags.restorePoint,
					MajorVersion: flags.majorVersion,
				}

				if cmd.Flags().Changed("replicas") {
					replicas := flags.replicas
					createReq.Replicas = &replicas
				}

				if cmd.Flags().Changed("min-storage") || cmd.Flags().Changed("max-storage") {
					createReq.Storage = &ps.StorageConfig{}
					if cmd.Flags().Changed("min-storage") {
						createReq.Storage.MinimumStorageBytes = &flags.minStorage
					}
					if cmd.Flags().Changed("max-storage") {
						createReq.Storage.MaximumStorageBytes = &flags.maxStorage
					}
				}

				dbBranch, err := client.PostgresBranches.Create(cmd.Context(), createReq)
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
					getReq := &ps.GetPostgresBranchRequest{
						Organization: ch.Config.Organization,
						Database:     source,
						Branch:       branch,
					}
					dbBranch, err = waitUntilPostgresReady(ctx, client, ch.Printer, ch.Debug(), getReq)
					if err != nil {
						return err
					}
					end()
				}
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("Branch %s was successfully created in %s.\n", printer.BoldBlue(dbBranch.Name), printer.BoldBlue(source))
					return nil
				}

				return ch.Printer.PrintResource(ToPostgresBranch(dbBranch))
			}
		},
	}

	cmd.Flags().StringVar(&flags.parentBranch, "from", "", "Parent branch to create the new branch from. Cannot be used with --restore unless --restore-point is set.")
	cmd.Flags().StringVar(&flags.region, "region", "", "Region for the branch to be created in.")
	cmd.Flags().StringVar(&flags.backupID, "restore", "", "ID of Backup to restore into branch.")
	cmd.Flags().StringVar(&flags.restorePoint, "restore-point", "", "For PostgreSQL databases, restore from a point-in-time recovery timestamp (e.g. 2023-01-01T00:00:00Z). Requires --restore or --from.")
	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "", "Cluster size for the branch. Defaults to PS_DEV for regular branches, or PS-10 for branches created from a backup or with seed-data. Use 'pscale size cluster list' to see the valid sizes.")
	cmd.Flags().BoolVar(&flags.dataBranching, "seed-data", false, "Add seed data using the Data Branching™ feature. This branch will be created with the same resources as the base branch.")
	cmd.Flags().BoolVar(&flags.wait, "wait", false, "Wait until the branch is ready")
	cmd.Flags().StringVar(&flags.majorVersion, "major-version", "", "For PostgreSQL databases, the PostgreSQL major version to use for the branch. Defaults to the major version of the parent branch if it exists or the database's default branch major version. Ignored for branches restored from backups.")
	cmd.Flags().IntVar(&flags.replicas, "replicas", 0, "Number of additional replicas for a PostgreSQL restore. 0 creates a single-node branch; omit to use the target cluster size default.")
	cmd.Flags().Int64Var(&flags.minStorage, "min-storage", 0, "Minimum storage size in bytes")
	cmd.Flags().Int64Var(&flags.maxStorage, "max-storage", 0, "Maximum storage size in bytes for autoscaling")

	cmd.MarkFlagsMutuallyExclusive("restore", "seed-data")
	cmd.MarkFlagsMutuallyExclusive("restore-point", "seed-data")

	cmd.RegisterFlagCompletionFunc("region", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.RegionsCompletionFunc(ch, cmd, args, toComplete)
	})

	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.ClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	cmd.RegisterFlagCompletionFunc("major-version", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return []cobra.Completion{
			cobra.CompletionWithDesc("17", "PostgreSQL 17"),
		}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

// waitUntilReady waits until the given database branch is ready. It times out after 10 minutes.
func waitUntilReady(ctx context.Context, client *ps.Client, printer *printer.Printer, debug bool, getReq *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	startTime := time.Now()
	var ticker *time.Ticker

	// Start with 5-second interval for the first minute
	ticker = time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("branch creation timed out")
		case <-ticker.C:
			resp, err := client.DatabaseBranches.Get(ctx, getReq)
			if err != nil {
				if debug {
					printer.Printf("fetching database branch %s/%s failed: %s", getReq.Database, getReq.Branch, err)
				}
				continue
			}

			if resp.Ready {
				return resp, nil
			}

			elapsed := time.Since(startTime)
			if elapsed > time.Minute {
				// Switch to 10-second interval after 1 minute
				ticker.Stop()
				ticker = time.NewTicker(10 * time.Second)
			}
		}
	}
}

func waitUntilPostgresReady(ctx context.Context, client *ps.Client, printer *printer.Printer, debug bool, getReq *ps.GetPostgresBranchRequest) (*ps.PostgresBranch, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	startTime := time.Now()
	var ticker *time.Ticker

	// Start with 5-second interval for the first minute
	ticker = time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("branch creation timed out")
		case <-ticker.C:
			resp, err := client.PostgresBranches.Get(ctx, getReq)
			if err != nil {
				if debug {
					printer.Printf("fetching database branch %s/%s failed: %s", getReq.Database, getReq.Branch, err)
				}
				continue
			}

			if resp.Ready {
				return resp, nil
			}

			elapsed := time.Since(startTime)
			if elapsed > time.Minute {
				// Switch to 10-second interval after 1 minute
				ticker.Stop()
				ticker = time.NewTicker(10 * time.Second)
			}
		}
	}
}
