package token

import (
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func UpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var name string
	var ttl string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "update a service token's name or expiry",
		Long:  "Update a service token's name or expiry. This does not rotate the token secret or change its id or grants.",
		Args:  cmdutil.RequiredArgs("id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := ch.Client()
			if err != nil {
				return err
			}

			id := args[0]
			nameSet := cmd.Flags().Changed("name")
			ttlSet := cmd.Flags().Changed("ttl")
			if !nameSet && !ttlSet {
				return fmt.Errorf("at least one of --name or --ttl must be provided")
			}

			req := &planetscale.UpdateServiceTokenRequest{
				Organization: ch.Config.Organization,
				ID:           id,
			}
			if nameSet {
				req.Name = &name
			}
			if ttlSet {
				parsed, err := serviceTokenTTL(ttl)
				if err != nil {
					return err
				}
				req.TTL = parsed
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating service token %s in org %s",
				printer.BoldBlue(id), printer.BoldBlue(ch.Config.Organization)))
			defer end()

			token, err := client.ServiceTokens.Update(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("service token %s does not exist in organization %s",
						printer.BoldBlue(id), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			return ch.Printer.PrintResource(toServiceTokenDetails(token))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "new name for the service token. Omit to leave the name unchanged")
	cmd.Flags().StringVar(&ttl, "ttl", "", `time to live in seconds, or "never" to remove the expiry. Omit to leave the expiry unchanged`)

	return cmd
}

// serviceTokenTTL parses the same seconds unit as service-token create.
// "never" clears the expiry. Create uses 0 for that and omits the field;
// update cannot, because an omitted --ttl leaves the expiry alone.
func serviceTokenTTL(v string) (*planetscale.UpdateServiceTokenTTL, error) {
	if v == "never" {
		return &planetscale.UpdateServiceTokenTTL{Never: true}, nil
	}
	seconds, err := strconv.Atoi(v)
	if err != nil {
		return nil, fmt.Errorf(`--ttl must be a number of seconds or "never"`)
	}
	return &planetscale.UpdateServiceTokenTTL{Seconds: seconds}, nil
}
