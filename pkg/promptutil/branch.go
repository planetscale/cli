package promptutil

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AlecAivazis/survey/v2"
	ps "github.com/planetscale/planetscale-go/planetscale"
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
		return "", err
	}

	if len(branches) == 0 {
		return "", fmt.Errorf("no branch exist for database: %q", db)
	}

	// if there is only one branch, just return it
	if len(branches) == 1 {
		return branches[0].Name, nil
	}

	branchNames := make([]string, 0, len(branches)-1)
	for _, b := range branches {
		branchNames = append(branchNames, b.Name)
	}

	prompt := &survey.Select{
		Message: "Select a branch to connect to:",
		Options: branchNames,
		VimMode: true,
	}

	type result struct {
		branch string
		err    error
	}

	resp := make(chan result)

	go func() {
		var branch string
		err := survey.AskOne(prompt, &branch)
		resp <- result{
			branch: branch,
			err:    err,
		}
	}()

	// timeout so CLI is not blocked forever if the user accidently called it
	select {
	case <-time.After(time.Second * 20):
		// TODO(fatih): this is buggy. Because there is no proper cancellation
		// in the survey.AskOne() function, it holds to stdin, which causes the
		// terminal to malfunction. But the timeout is not intended for regular
		// users, it's meant to catch script invocations, so let's still use it
		return "", errors.New("pscale connect timeout: no branch is selected")
	case r := <-resp:
		return r.branch, r.err
	}
}
