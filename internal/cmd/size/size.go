package size

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func main() {
	fmt.Println("vim-go")
}

func SizeCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "size <command>",
		Short:             "Lists the sizes for various components within PlanetScale",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization,
		"The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint: errcheck

	cmd.AddCommand(ClusterCmd(ch))

	return cmd
}
