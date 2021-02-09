package version

import (
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/config"

	"github.com/spf13/cobra"
)

// VersionCmd encapsulates the commands for showing a version
func VersionCmd(cfg *config.Config, ver, commit, buildDate string) *cobra.Command {
	cmd := &cobra.Command{
		Use: "version <command>",
		// we can also show the version via `--version`, hence this doesn't
		// need to be displayed.
		Hidden: true, //
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(Format(ver, commit, buildDate))
		},
	}

	return cmd
}

// Format formats a version string with the given information.
func Format(ver, commit, buildDate string) string {
	if ver == "" && buildDate == "" && commit == "" {
		return "pscale version (built from source)"
	}

	ver = strings.TrimPrefix(ver, "v")

	return fmt.Sprintf("pscale version %s (build date: %s commit: %s)\n", ver, buildDate, commit)
}
