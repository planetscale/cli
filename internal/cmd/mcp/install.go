package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// ClaudeConfig represents the structure of the Claude Desktop config file
type ClaudeConfig map[string]interface{}

// getClaudeConfigDir returns the path to the Claude Desktop config directory based on the OS
func getClaudeConfigDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		// macOS path: ~/Library/Application Support/Claude/
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine user home directory: %w", err)
		}
		return filepath.Join(homeDir, "Library", "Application Support", "Claude"), nil
	case "windows":
		// Windows path: %APPDATA%\Claude\
		return filepath.Join(os.Getenv("APPDATA"), "Claude"), nil
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// InstallCmd returns a new cobra.Command for the mcp install command.
func InstallCmd(ch *cmdutil.Helper) *cobra.Command {
	var target string
	
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the MCP server",
		Long:  `Install the PlanetScale model context protocol (MCP) server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if target != "claude" {
				return fmt.Errorf("invalid target vendor: %s (only 'claude' is supported)", target)
			}

			configDir, err := getClaudeConfigDir()
			if err != nil {
				return fmt.Errorf("failed to determine Claude config directory: %w", err)
			}

			// Check if the directory exists
			if _, err := os.Stat(configDir); os.IsNotExist(err) {
				return fmt.Errorf("Claude Desktop is not installed: path %s not found", configDir)
			}

			configPath := filepath.Join(configDir, "claude_desktop_config.json")
			config := make(ClaudeConfig)

			// Check if the file exists
			if _, err := os.Stat(configPath); err == nil {
				// File exists, read it
				configData, err := os.ReadFile(configPath)
				if err != nil {
					return fmt.Errorf("failed to read Claude config file: %w", err)
				}

				if err := json.Unmarshal(configData, &config); err != nil {
					return fmt.Errorf("failed to parse Claude config file: %w", err)
				}
			}

			// Get or initialize the mcpServers map
			var mcpServers map[string]interface{}
			if existingServers, ok := config["mcpServers"].(map[string]interface{}); ok {
				mcpServers = existingServers
			} else {
				mcpServers = make(map[string]interface{})
			}

			// Add or update the planetscale server configuration
			mcpServers["planetscale"] = map[string]interface{}{
				"command": "pscale",
				"args":    []string{"mcp", "server"},
			}
			
			// Update the config with the new mcpServers
			config["mcpServers"] = mcpServers

			// Write the updated config back to file
			configJSON, err := json.MarshalIndent(config, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal Claude config: %w", err)
			}

			if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
				return fmt.Errorf("failed to write Claude config file: %w", err)
			}

			fmt.Printf("MCP server successfully configured for %s at %s\n", target, configPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Target vendor for MCP installation (required). Possible values: [claude]")
	cmd.MarkFlagRequired("target")
	cmd.RegisterFlagCompletionFunc("target", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"claude"}, cobra.ShellCompDirectiveDefault
	})

	return cmd
}