package promptutil

import (
	"context"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pkg/errors"
	"github.com/planetscale/planetscale-go/planetscale"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
)

// GetBranch returns the database branch. If there is only one branch, it
// returns the branch immediately. Otherwise the user is prompted to select the
// branch from a list of existing branches.
func GetBranch(ctx context.Context, client *ps.Client, org, db string) (string, error) {
	branches, err := client.DatabaseBranches.List(ctx, &ps.ListDatabaseBranchesRequest{
		Organization: org,
		Database:     db,
	})
	if err != nil {
		switch cmdutil.ErrCode(err) {
		case planetscale.ErrResponseMalformed:
			return "", cmdutil.MalformedError(err)
		default:
			return "", errors.Wrap(err, "error listing branches")
		}
	}

	if len(branches) == 0 {
		return "", fmt.Errorf("no branch exist for database: %q", db)
	}

	// if there is only one branch, just return it
	if len(branches) == 1 {
		return branches[0].Name, nil
	}

	if printer.IsTTY {
		branchNames := make([]string, 0, len(branches)-1)
		for _, b := range branches {
			branchNames = append(branchNames, b.Name)
		}

		prompt := &survey.Select{
			Message: "Select a branch to connect to:",
			Options: branchNames,
			VimMode: true,
		}

		var branch string
		err = survey.AskOne(prompt, &branch)
		return branch, err
	}

	return "", fmt.Errorf("more than one branch exists for database %q", db)
}
