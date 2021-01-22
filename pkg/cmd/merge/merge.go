package merge

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/planetscale/cli/cmdutil"
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

func MergeCmd(cfg *config.Config) *cobra.Command {
	mergeInto := ""

	cmd := &cobra.Command{
		Use:   "merge [database] [from_branch]",
		Short: "Merge the branch from the database",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if len(args) != 2 {
				return cmd.Usage()
			}

			database := args[0]
			fromBranch := args[1]

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			// end := cmdutil.PrintProgress(fmt.Sprintf("Fetching branch %s for %s", cmdutil.BoldBlue(fromBranch), cmdutil.BoldBlue(database)))
			// defer end()

			// Check that the branch actually exists.
			branch, err := client.DatabaseBranches.Get(ctx, cfg.Organization, database, fromBranch)
			if err != nil {
				return err
			}
			// end()

			// If mergeInto is blank, then we need to prompt the user to select a
			// branch.
			if mergeInto == "" {
				end := cmdutil.PrintProgress(fmt.Sprintf("Fetching branches for %s", cmdutil.BoldBlue(database)))
				defer end()

				branches, err := client.DatabaseBranches.List(ctx, cfg.Organization, database)
				if err != nil {
					return err
				}
				end()

				if len(branches) == 1 && branches[0].Name == branch.Name {
					return fmt.Errorf("There are no other branches to merge %s into", branch.Name)
				}

				branchNames := make([]string, 0, len(branches)-1)
				for _, b := range branches {
					if b.Name == fromBranch {
						continue
					}
					branchNames = append(branchNames, b.Name)
				}

				prompt := &survey.Select{
					Message: fmt.Sprintf("Select a branch to merge %s into: ", cmdutil.BoldBlue(branch.Name)),
					Options: branchNames,
				}

				err = survey.AskOne(prompt, &mergeInto)
				if err != nil {
					return err
				}
			}

			if mergeInto == branch.Name {
				return errors.New("Cannot merge a branch into itself")
			}

			// TODO(iheanyi): Call branch merge request here.
			fmt.Printf("Will merge %s into %s\n", cmdutil.BoldBlue(branch.Name), cmdutil.BoldBlue(mergeInto))

			return nil
		},
	}

	cmd.Flags().StringVar(&mergeInto, "into", "", "the branch to be merged into")
	cmd.PersistentFlags().StringVar(&cfg.Organization, "org", cfg.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	return cmd
}
