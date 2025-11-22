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

type ClaudeConfig map[string]any

func installMCPServer(configPath string) error {
	config := make(ClaudeConfig)

	if _, err := os.Stat(configPath); err == nil {
		configData, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}

		if err := json.Unmarshal(configData, &config); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	var mcpServers map[string]any
	if existingServers, ok := config["mcpServers"].(map[string]any); ok {
		mcpServers = existingServers
	} else {
		mcpServers = make(map[string]any)
	}

	mcpServers["planetscale"] = map[string]any{
		"command": "pscale",
		"args":    []string{"mcp", "server"},
	}

	config["mcpServers"] = mcpServers

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

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
				if err := installMCPServer(configPath); err != nil {
					return err
				}
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

				if err := installMCPServer(configPath); err != nil {
					return err
				}
			case "zed":
				configPath, err = getZedConfigPath()
				if err != nil {
					return fmt.Errorf("failed to determine Zed config path: %w", err)
				}

				configDir := filepath.Dir(configPath)
				if _, err := os.Stat(configDir); os.IsNotExist(err) {
					if err := os.MkdirAll(configDir, 0755); err != nil {
						return fmt.Errorf("failed to create Zed config directory: %w", err)
					}
				}

				if err := installZedMCPServer(configPath); err != nil {
					return err
				}
			default:
				return fmt.Errorf("invalid target vendor: %s (supported values: claude, cursor, zed)", target)
			}

			fmt.Printf("MCP server successfully configured for %s at %s\n", target, configPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Target vendor for MCP installation (required). Possible values: [claude, cursor, zed]")
	cmd.MarkFlagRequired("target")
	cmd.RegisterFlagCompletionFunc("target", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"claude", "cursor", "zed"}, cobra.ShellCompDirectiveDefault
	})

	return cmd
}
