package deployrequest

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// ReviewCmd is the command for reviewing (approve, comment, etc.) a deploy
// request.
func ReviewCmd(cfg *config.Config) *cobra.Command {
	var flags struct {
		approve bool
		comment string
	}

	cmd := &cobra.Command{
		Use:   "review <database> <number>",
		Short: "Review a deploy request (approve, comment, etc...)",
		Args:  cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !flags.approve && flags.comment == "" {
				return errors.New("neither --approve nor --comment is set")
			}

			ctx := context.Background()
			database := args[0]
			number := args[1]

			n, err := strconv.ParseUint(number, 10, 64)
			if err != nil {
				return fmt.Errorf("The argument <number> is invalid: %s", err)
			}

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			action := planetscale.ReviewComment
			if flags.approve {
				action = planetscale.ReviewApprove
			}

			_, err = client.DeployRequests.CreateReview(ctx, &planetscale.ReviewDeployRequestRequest{
				Organization: cfg.Organization,
				Database:     database,
				Number:       n,
				ReviewAction: action,
				CommentText:  flags.comment,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("deploy request '%s/%s' does not exist in organization %s\n",
						cmdutil.BoldBlue(database), cmdutil.BoldBlue(number), cmdutil.BoldBlue(cfg.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			switch action {
			case planetscale.ReviewApprove:
				fmt.Printf("Deploy request %s/%s is approved.\n",
					cmdutil.BoldBlue(database), cmdutil.BoldBlue(number))
			case planetscale.ReviewComment:
				fmt.Printf("A comment is added to the deploy request %s/%s.\n",
					cmdutil.BoldBlue(database), cmdutil.BoldBlue(number))
			}

			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&flags.approve, "approve", false, "Approve a deploy request")
	cmd.PersistentFlags().StringVar(&flags.comment, "comment", "", "Comment on a deploy request")

	return cmd
}
