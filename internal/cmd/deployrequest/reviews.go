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

// ReviewsCmd lists reviews for a deploy request.
func ReviewsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reviews <database> <number>",
		Short: "List reviews for a deploy request",
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

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching reviews for deploy request %s/%s",
				printer.BoldBlue(database), printer.BoldBlue(number)))
			defer end()

			reviews, err := client.DeployRequests.ListReviews(ctx, &planetscale.ListDeployRequestReviewsRequest{
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

			if len(reviews) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No reviews for %s/%s.\n", printer.BoldBlue(database), printer.BoldBlue(number))
				return nil
			}

			return ch.Printer.PrintResource(toDeployRequestReviews(reviews))
		},
	}

	return cmd
}

type DeployRequestReviewRow struct {
	ID        string `header:"id" json:"id"`
	State     string `header:"state" json:"state"`
	Actor     string `header:"actor" json:"actor"`
	Body      string `header:"body" json:"body"`
	CreatedAt string `header:"created_at" json:"created_at"`

	orig *planetscale.DeployRequestReview
}

func (d *DeployRequestReviewRow) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.orig, "", "  ")
}

func (d *DeployRequestReviewRow) MarshalCSVValue() interface{} {
	return []*DeployRequestReviewRow{d}
}

func toDeployRequestReview(r *planetscale.DeployRequestReview) *DeployRequestReviewRow {
	return &DeployRequestReviewRow{
		ID:        r.ID,
		State:     r.State,
		Actor:     r.Actor.Name,
		Body:      r.Body,
		CreatedAt: formatTimestampRequired(r.CreatedAt),
		orig:      r,
	}
}

func toDeployRequestReviews(reviews []*planetscale.DeployRequestReview) []*DeployRequestReviewRow {
	rows := make([]*DeployRequestReviewRow, 0, len(reviews))
	for _, r := range reviews {
		rows = append(rows, toDeployRequestReview(r))
	}
	return rows
}
