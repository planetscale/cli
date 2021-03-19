package cmdutil

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// RequiredArgs returns a short and actionable error message if the given
// required arguments are not available.
func RequiredArgs(reqArgs ...string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		// show just the usage line and any examples if they exist
		var usageTemplate = `Usage:{{if .Runnable}} {{.UseLine}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}
`
		cmd.SetUsageTemplate(usageTemplate)

		n := len(reqArgs)
		if len(args) >= n {
			return nil
		}

		missing := reqArgs[len(args):]

		a := fmt.Sprintf("arguments <%s>", strings.Join(missing, ", "))
		if len(missing) == 1 {
			a = fmt.Sprintf("argument <%s>", missing[0])
		}

		return fmt.Errorf("missing %s \n\n%s", a, cmd.UsageString())
	}
}
