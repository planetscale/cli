package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
	"github.com/tidwall/jsonc"
)

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
				configPath, err = getClaudeConfigPath()
				if err != nil {
					return fmt.Errorf("failed to determine Claude config path: %w", err)
				}
				if err := installMCPServer(configPath, target, modifyClaudeConfig); err != nil {
					return err
				}
			case "cursor":
				configPath, err = getCursorConfigPath()
				if err != nil {
					return fmt.Errorf("failed to determine Cursor config path: %w", err)
				}
				// Cursor uses the same config structure as Claude
				if err := installMCPServer(configPath, target, modifyClaudeConfig); err != nil {
					return err
				}
			case "zed":
				configPath, err = getZedConfigPath()
				if err != nil {
					return fmt.Errorf("failed to determine Zed config path: %w", err)
				}
				if err := installMCPServer(configPath, target, modifyZedConfig); err != nil {
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

// installMCPServer handles common file I/O for all editors
func installMCPServer(configPath string, target string, modifyConfig func(map[string]any) error) error {
	// Check if config directory exists
	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return fmt.Errorf("no %s installation: path %s not found", target, configDir)
	}

	// Read existing config or create empty settings
	var fullSettings map[string]any
	if fileData, err := os.ReadFile(configPath); err == nil {
		cleanJSON := jsonc.ToJSON(fileData)
		if err := json.Unmarshal(cleanJSON, &fullSettings); err != nil {
			return fmt.Errorf("failed to parse %s config file: %w", target, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s config file: %w", target, err)
	} else {
		fullSettings = make(map[string]any)
	}

	// Let the editor-specific function modify the config
	if err := modifyConfig(fullSettings); err != nil {
		return err
	}

	// Marshal updated config
	updatedData, err := json.MarshalIndent(fullSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal %s config: %w", target, err)
	}

	// Backup existing file before writing
	if _, err := os.Stat(configPath); err == nil {
		backupPath := configPath + "~"
		if err := os.Rename(configPath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("Created backup at %s\n", backupPath)
	}

	// Write updated config
	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write %s config file: %w", target, err)
	}

	return nil
}

// modifyClaudeConfig adds planetscale to mcpServers (Claude and Cursor both use this)
func modifyClaudeConfig(settings map[string]any) error {
	var mcpServers map[string]any
	if existingServers, ok := settings["mcpServers"].(map[string]any); ok {
		mcpServers = existingServers
	} else {
		mcpServers = make(map[string]any)
	}

	mcpServers["planetscale"] = map[string]any{
		"command": "pscale",
		"args":    []string{"mcp", "server"},
	}

	settings["mcpServers"] = mcpServers
	return nil
}

// modifyZedConfig adds planetscale to context_servers (Zed-specific)
func modifyZedConfig(settings map[string]any) error {
	var contextServers map[string]any
	if existingServers, ok := settings["context_servers"].(map[string]any); ok {
		contextServers = existingServers
	} else {
		contextServers = make(map[string]any)
	}

	contextServers["planetscale"] = map[string]any{
		"source":  "custom",
		"command": "pscale",
		"args":    []string{"mcp", "server"},
	}

	settings["context_servers"] = contextServers
	return nil
}

// getClaudeConfigPath returns the path to the Claude Desktop config file based on the OS
func getClaudeConfigPath() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		// macOS path: ~/Library/Application Support/Claude/claude_desktop_config.json
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine user home directory: %w", err)
		}
		return filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json"), nil
	case "windows":
		// Windows path: %APPDATA%\Claude\claude_desktop_config.json
		return filepath.Join(os.Getenv("APPDATA"), "Claude", "claude_desktop_config.json"), nil
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

// getZedConfigPath returns the path to the Zed config file based on the OS
func getZedConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine user home directory: %w", err)
	}

	switch runtime.GOOS {
	case "darwin", "linux":
		return filepath.Join(homeDir, ".config", "zed", "settings.json"), nil
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "Zed", "settings.json"), nil
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
