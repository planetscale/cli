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

// getCursorConfigPath returns the path to the Cursor MCP config file
func getCursorConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine user home directory: %w", err)
	}

	// Cursor uses ~/.cursor/mcp.json for its MCP configuration
	return filepath.Join(homeDir, ".cursor", "mcp.json"), nil
}

// InstallCmd returns a new cobra.Command for the mcp install command.
func InstallCmd(ch *cmdutil.Helper) *cobra.Command {
	var target string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the MCP server",
		Long:  `Install the PlanetScale model context protocol (MCP) server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var configPath string
			var err error

			switch target {
			case "claude":
				configDir, err := getClaudeConfigDir()
				if err != nil {
					return fmt.Errorf("failed to determine Claude config directory: %w", err)
				}

				// Check if the directory exists
				if _, err := os.Stat(configDir); os.IsNotExist(err) {
					return fmt.Errorf("no Claude Desktop installation: path %s not found", configDir)
				}

				configPath = filepath.Join(configDir, "claude_desktop_config.json")
			case "cursor":
				configPath, err = getCursorConfigPath()
				if err != nil {
					return fmt.Errorf("failed to determine Cursor config path: %w", err)
				}

				// Ensure the .cursor directory exists
				configDir := filepath.Dir(configPath)
				if _, err := os.Stat(configDir); os.IsNotExist(err) {
					if err := os.MkdirAll(configDir, 0755); err != nil {
						return fmt.Errorf("failed to create Cursor config directory: %w", err)
					}
				}
			default:
				return fmt.Errorf("invalid target vendor: %s (supported values: claude, cursor)", target)
			}

			config := make(ClaudeConfig) // Same config structure for both tools

			// Check if the file exists
			if _, err := os.Stat(configPath); err == nil {
				// File exists, read it
				configData, err := os.ReadFile(configPath)
				if err != nil {
					return fmt.Errorf("failed to read %s config file: %w", target, err)
				}

				if err := json.Unmarshal(configData, &config); err != nil {
					return fmt.Errorf("failed to parse %s config file: %w", target, err)
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
				return fmt.Errorf("failed to marshal %s config: %w", target, err)
			}

			if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
				return fmt.Errorf("failed to write %s config file: %w", target, err)
			}

			fmt.Printf("MCP server successfully configured for %s at %s\n", target, configPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Target vendor for MCP installation (required). Possible values: [claude, cursor]")
	cmd.MarkFlagRequired("target")
	cmd.RegisterFlagCompletionFunc("target", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"claude", "cursor"}, cobra.ShellCompDirectiveDefault
	})

	return cmd
}
