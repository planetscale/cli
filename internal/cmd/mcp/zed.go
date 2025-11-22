package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/tidwall/jsonc"
)

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

type zedConfig struct {
	ContextServers map[string]any `json:"context_servers,omitempty"`
}

func parseZedConfig(configPath string) (*zedConfig, error) {
	var config zedConfig

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &zedConfig{ContextServers: make(map[string]any)}, nil
	}

	fileData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cleanJSON := jsonc.ToJSON(fileData)
	if err := json.Unmarshal(cleanJSON, &config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	if config.ContextServers == nil {
		config.ContextServers = make(map[string]any)
	}

	return &config, nil
}

func installZedMCPServer(configPath string) error {
	cfg, err := parseZedConfig(configPath)
	if err != nil {
		return err
	}

	cfg.ContextServers["planetscale"] = map[string]any{
		"source":  "custom",
		"command": "pscale",
		"args":    []string{"mcp", "server"},
	}

	if err := storeContextServers(configPath, cfg); err != nil {
		return fmt.Errorf("failed to store context-servers: %w", err)
	}

	return nil
}

func storeContextServers(configPath string, cfg *zedConfig) error {
	var fullSettings map[string]any

	if fileData, err := os.ReadFile(configPath); err == nil {
		backupPath := configPath + ".backup"
		if err := os.WriteFile(backupPath, fileData, 0644); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("Created backup at %s\n", backupPath)

		cleanJSON := jsonc.ToJSON(fileData)
		if err := json.Unmarshal(cleanJSON, &fullSettings); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read config file: %w", err)
	} else {
		fullSettings = make(map[string]any)
	}

	fullSettings["context_servers"] = cfg.ContextServers

	updatedData, err := json.MarshalIndent(fullSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
