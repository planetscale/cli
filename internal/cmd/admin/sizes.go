package admin

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func SizesCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sizes <database> <branch>",
		Short:   "List available admin sizes for a Neki database branch",
		Args:    cmdutil.ExactArgs("database", "branch"),
		Aliases: []string{"size-skus"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := args[0], args[1]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching admin sizes for %s/%s", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			skus, err := client.NekiAdmins.ListSizeSKUs(cmd.Context(), &ps.ListNekiAdminSizeSKUsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return handleError(err, database, branch)
			}
			end()

			if len(skus) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No admin sizes found for %s/%s.\n", printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(toAdminSKUs(skus))
		},
	}

	return cmd
}

type adminSKUDisplay struct {
	Name   string `header:"name" json:"name"`
	CPU    string `header:"cpu" json:"cpu"`
	Memory string `header:"memory" json:"memory"`

	orig *ps.NekiAdminSKU
}

func (s *adminSKUDisplay) MarshalJSON() ([]byte, error) { return json.Marshal(s.orig) }
func (s *adminSKUDisplay) MarshalCSVValue() interface{} { return []*adminSKUDisplay{s} }

func toAdminSKU(sku *ps.NekiAdminSKU) *adminSKUDisplay {
	name := sku.DisplayName
	if name == "" {
		name = cmdutil.ToClusterSizeSlug(sku.Name)
	}

	return &adminSKUDisplay{
		Name:   name,
		CPU:    sku.CPU,
		Memory: cmdutil.FormatParts(sku.RAM).IntString(),
		orig:   sku,
	}
}

func toAdminSKUs(skus []*ps.NekiAdminSKU) []*adminSKUDisplay {
	out := make([]*adminSKUDisplay, 0, len(skus))
	for _, sku := range skus {
		out = append(out, toAdminSKU(sku))
	}
	return out
}
