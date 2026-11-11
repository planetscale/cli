package oidctrust

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func DeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <trust-id>",
		Short: "Delete an OIDC trust",
		Args:  cmdutil.RequiredArgs("trust-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting OIDC trust %s", printer.BoldBlue(id)))
			defer end()

			err = client.OIDCTrusts.Delete(cmd.Context(), &ps.DeleteOIDCTrustRequest{
				Organization: ch.Config.Organization,
				ID:           id,
			})
			if err != nil {
				return notFoundOrHandle(ch, id, err)
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("OIDC trust %s was successfully deleted.\n", printer.BoldBlue(id))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result": "oidc trust deleted",
			})
		},
	}

	return cmd
}

func notFoundOrHandle(ch *cmdutil.Helper, id string, err error) error {
	if cmdutil.ErrCode(err) == ps.ErrNotFound {
		return fmt.Errorf("OIDC trust %s does not exist in organization %s",
			printer.BoldBlue(id), printer.BoldBlue(ch.Config.Organization))
	}
	return cmdutil.HandleError(err)
}
