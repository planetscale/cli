package agentguide

import (
	"fmt"

	"github.com/spf13/cobra"

	clicontent "github.com/planetscale/cli"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
)

const (
	HostedMCPURL        = "https://mcp.pscale.dev/mcp/planetscale"
	MCPDocsURL          = "https://planetscale.com/docs/connect/mcp"
	SkillsRepoURL       = "https://github.com/planetscale/skills"
	SkillsSetupCmd      = "git clone https://github.com/planetscale/skills.git && cd skills && script/setup"
	SkillsNPXInstall    = "npx skills add planetscale/skills -g -y"
	SkillsCLIAutomation = "14-pscale-cli-automation"
)

type response struct {
	Status              string   `json:"status"`
	Guide               string   `json:"guide"`
	FirstCommand        string   `json:"first_command"`
	AgentGuideCommand   string   `json:"agent_guide_command"`
	HostedMCPURL        string   `json:"hosted_mcp_url"`
	MCPDocsURL          string   `json:"mcp_docs_url"`
	SkillsRepoURL       string   `json:"skills_repo_url"`
	SkillsSetupCommand  string   `json:"skills_setup_command"`
	SkillsNPXCommand    string   `json:"skills_npx_command"`
	SkillsCLIAutomation string   `json:"skills_cli_automation_skill"`
	SupportedEngines    []string `json:"supported_engines"`
	Conventions         []string `json:"conventions"`
	NextSteps           []string `json:"next_steps"`
}

// SkillDoc renders the embedded agent guide as an installable skill file:
// YAML frontmatter (name/description trigger) followed by the guide body.
// AGENTS.md stays the single source of truth for the body.
func SkillDoc() string {
	return "---\n" +
		"name: pscale-cli\n" +
		"description: \"Automate PlanetScale with the pscale CLI. Use when the user asks to run pscale commands or manage PlanetScale databases, branches, deploy requests, or SQL from scripts or agents. Always pass --format json.\"\n" +
		"---\n\n" +
		clicontent.AgentGuide
}

// AgentGuideCmd prints the embedded guide for agents and automation.
func AgentGuideCmd(ch *cmdutil.Helper) *cobra.Command {
	var asSkill bool
	cmd := &cobra.Command{
		Use:   "agent-guide",
		Short: "Show guidance for AI agents and automation",
		Long: `Show guidance for AI agents and automation using pscale.

Use --format json for a machine-readable bootstrap response with first commands,
supported engines, hosted MCP details, and PlanetScale skills install hints.

Use --skill to print the guide as an installable agent skill file (YAML
frontmatter plus the guide), suitable for dropping into a skills directory.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if asSkill {
				// --skill dumps a raw skill file (like --version), not a
				// resource. Write straight to stdout: Printer.Print routes
				// non-human formats to io.Discard, which would exit 0 with an
				// empty file when combined with --format json.
				fmt.Print(SkillDoc())
				return nil
			}
			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(response{
					Status:              "ok",
					Guide:               clicontent.AgentGuide,
					FirstCommand:        cmdutil.AgentAuthCheckCmd(),
					AgentGuideCommand:   cmdutil.AgentGuideCmd(),
					HostedMCPURL:        HostedMCPURL,
					MCPDocsURL:          MCPDocsURL,
					SkillsRepoURL:       SkillsRepoURL,
					SkillsSetupCommand:  SkillsSetupCmd,
					SkillsNPXCommand:    SkillsNPXInstall,
					SkillsCLIAutomation: SkillsCLIAutomation,
					SupportedEngines:    []string{"mysql", "postgresql"},
					Conventions: []string{
						"Always pass --format json for automation",
						"Put --org on resource subcommands, not on pscale root",
						"Put positional arguments before flags for commands like sql and branch list",
						"Use hosted MCP for MCP clients; direct CLI automation should start with auth check",
						"Install planetscale/skills for operational workflows; this guide covers pscale invocation only",
						"Project AGENTS.md is for org/database/branch targeting — separate from the CLI agent guide",
					},
					NextSteps: []string{
						cmdutil.AgentAuthCheckCmd(),
						SkillsSetupCmd,
					},
				})
			}

			ch.Printer.Print(clicontent.AgentGuide)
			return nil
		},
	}

	cmd.Flags().BoolVar(&asSkill, "skill", false, "Print the guide as an installable agent skill file and exit")

	return cmd
}
