package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	setupConfigDir string
	setupProject   bool
)

var setupCmd = &cobra.Command{
	Use:   "setup [agent]",
	Short: "Install Pantry hooks for an agent",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		agent := args[0]

		var result map[string]string
		var err error

		switch agent {
		case "claude", "claude-code":
			result, err = setupClaudeCode(setupConfigDir, setupProject)
		case "cursor":
			result, err = setupCursor(setupConfigDir, setupProject)
		case "codex":
			result, err = setupCodex(setupConfigDir, setupProject)
		case "opencode":
			result, err = setupOpenCode(setupProject)
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown agent: %s\n", agent)
			fmt.Fprintf(os.Stderr, "Supported agents: claude, cursor, codex, opencode\n")
			os.Exit(1)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(result["message"])
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [agent]",
	Short: "Remove Pantry hooks for an agent",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		agent := args[0]

		var result map[string]string
		var err error

		switch agent {
		case "claude", "claude-code":
			result, err = uninstallClaudeCode(setupConfigDir, setupProject)
		case "cursor":
			result, err = uninstallCursor(setupConfigDir, setupProject)
		case "codex":
			result, err = uninstallCodex(setupConfigDir, setupProject)
		case "opencode":
			result, err = uninstallOpenCode(setupProject)
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown agent: %s\n", agent)
			fmt.Fprintf(os.Stderr, "Supported agents: claude, cursor, codex, opencode\n")
			os.Exit(1)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(result["message"])
	},
}

func init() {
	setupCmd.Flags().StringVar(&setupConfigDir, "config-dir", "", "Path to agent config directory")
	setupCmd.Flags().BoolVar(&setupProject, "project", false, "Install in current project instead of globally")
	uninstallCmd.Flags().StringVar(&setupConfigDir, "config-dir", "", "Path to agent config directory")
	uninstallCmd.Flags().BoolVar(&setupProject, "project", false, "Uninstall from current project instead of globally")
}

func resolveConfigDir(agentDotDir string, configDir string, project bool) string {
	if configDir != "" {
		return configDir
	}
	if project {
		dir, _ := os.Getwd()
		return filepath.Join(dir, agentDotDir)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, agentDotDir)
}

func setupClaudeCode(configDir string, project bool) (map[string]string, error) {
	target := resolveConfigDir(".claude", configDir, project)
	configPath := filepath.Join(target, "settings.json")

	// Read existing config or create new
	var config map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]interface{})
	}

	// Add MCP server config
	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
		config["mcpServers"] = mcpServers
	}

	mcpServers["pantry"] = map[string]interface{}{
		"command": "pantry",
		"args":    []string{"mcp"},
	}

	// Write config
	if err := os.MkdirAll(target, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	msg := fmt.Sprintf("Installed Pantry MCP server in %s", configPath)
	if installSkill(target) {
		msg += " and skill"
	}

	return map[string]string{"message": msg}, nil
}

func setupCursor(configDir string, project bool) (map[string]string, error) {
	target := resolveConfigDir(".cursor", configDir, project)
	configPath := filepath.Join(target, "mcp.json")

	// Read existing config or create new
	var config map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]interface{})
	}

	// Add MCP server config
	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
		config["mcpServers"] = mcpServers
	}

	mcpServers["pantry"] = map[string]interface{}{
		"command": "pantry",
		"args":    []string{"mcp"},
	}

	// Write config
	if err := os.MkdirAll(target, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	msg := fmt.Sprintf("Installed Pantry MCP server in %s", configPath)
	if installSkill(target) {
		msg += " and skill"
	}

	return map[string]string{"message": msg}, nil
}

func setupCodex(configDir string, project bool) (map[string]string, error) {
	target := resolveConfigDir(".codex", configDir, project)
	configPath := filepath.Join(target, "config.toml")
	agentsPath := filepath.Join(target, "AGENTS.md")

	// Create config.toml entry
	// Note: TOML parsing would be needed for full implementation
	// For now, append to file

	configEntry := `
[mcpServers.pantry]
command = "pantry"
args = ["mcp"]
`

	if err := os.MkdirAll(target, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Append to config.toml
	file, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(configEntry); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	// Add to AGENTS.md
	agentsEntry := "## Pantry Memory System\n\n" +
		"You have access to a persistent memory system via Pantry. Use it to save important decisions, patterns, bugs, context, and learnings.\n\n" +
		"### Commands\n" +
		"- `pantry store` - Save a memory\n" +
		"- `pantry search` - Search memories\n" +
		"- `pantry list` - List recent memories\n"

	file2, err := os.OpenFile(agentsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		file2.WriteString(agentsEntry)
		file2.Close()
	}

	msg := fmt.Sprintf("Installed Pantry in %s", target)
	if installSkill(target) {
		msg += " (MCP + AGENTS.md + skill)"
	}

	return map[string]string{"message": msg}, nil
}

func setupOpenCode(project bool) (map[string]string, error) {
	var configPath string
	if project {
		dir, _ := os.Getwd()
		configPath = filepath.Join(dir, "opencode.json")
	} else {
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".config", "opencode", "opencode.json")
	}

	// Read existing config or create new
	var config map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]interface{})
	}

	// Add MCP server config
	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
		config["mcpServers"] = mcpServers
	}

	mcpServers["pantry"] = map[string]interface{}{
		"command": "pantry",
		"args":    []string{"mcp"},
	}

	// Write config
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	return map[string]string{
		"message": fmt.Sprintf("Installed Pantry MCP server in %s", configPath),
	}, nil
}

func uninstallClaudeCode(configDir string, project bool) (map[string]string, error) {
	target := resolveConfigDir(".claude", configDir, project)
	configPath := filepath.Join(target, "settings.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return map[string]string{"message": "Pantry not found in Claude Code config"}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if mcpServers, ok := config["mcpServers"].(map[string]interface{}); ok {
		delete(mcpServers, "pantry")
	}

	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	msg := fmt.Sprintf("Removed Pantry from %s", configPath)
	if uninstallSkill(target) {
		msg += " and skill"
	}

	return map[string]string{"message": msg}, nil
}

func uninstallCursor(configDir string, project bool) (map[string]string, error) {
	target := resolveConfigDir(".cursor", configDir, project)
	configPath := filepath.Join(target, "mcp.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return map[string]string{"message": "Pantry not found in Cursor config"}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if mcpServers, ok := config["mcpServers"].(map[string]interface{}); ok {
		delete(mcpServers, "pantry")
	}

	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	msg := fmt.Sprintf("Removed Pantry from %s", configPath)
	if uninstallSkill(target) {
		msg += " and skill"
	}

	return map[string]string{"message": msg}, nil
}

func uninstallCodex(configDir string, project bool) (map[string]string, error) {
	target := resolveConfigDir(".codex", configDir, project)
	msg := "Codex uninstall: manually remove Pantry entries from .codex/config.toml and AGENTS.md"
	if uninstallSkill(target) {
		msg += ". Removed skill."
	}
	return map[string]string{"message": msg}, nil
}

func uninstallOpenCode(project bool) (map[string]string, error) {
	var configPath string
	if project {
		dir, _ := os.Getwd()
		configPath = filepath.Join(dir, "opencode.json")
	} else {
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".config", "opencode", "opencode.json")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return map[string]string{"message": "Pantry not found in OpenCode config"}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if mcpServers, ok := config["mcpServers"].(map[string]interface{}); ok {
		delete(mcpServers, "pantry")
	}

	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	return map[string]string{
		"message": fmt.Sprintf("Removed Pantry from %s", configPath),
	}, nil
}
