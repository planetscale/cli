package cmd

import (
	"os"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(completionCmd)
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script for your shell",
	Long: `To load completions:

Bash:

  $ source <(pscale completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ pscale completion bash > /etc/bash_completion.d/pscale
  # macOS:
  $ pscale completion bash > /usr/local/etc/bash_completion.d/pscale

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ pscale completion zsh > "${fpath[1]}/_yourprogram"

  # You will need to start a new shell for this setup to take effect.

fish:

  $ pscale completion fish | source

  # To load completions for each session, execute once:
  $ pscale completion fish > ~/.config/fish/completions/pscale.fish

PowerShell:

  PS> pscale completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> pscale completion powershell > pscale.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cmdutil.RequiredArgs("shell"),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}
