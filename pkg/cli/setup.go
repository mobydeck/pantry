package cli

import (
	"bytes"
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
	setupCmd.Flags().BoolVarP(&setupProject, "project", "p", false, "Install in current project instead of globally")
	uninstallCmd.Flags().StringVar(&setupConfigDir, "config-dir", "", "Path to agent config directory")
	uninstallCmd.Flags().BoolVarP(&setupProject, "project", "p", false, "Uninstall from current project instead of globally")
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
	skillTarget := resolveConfigDir(".claude", configDir, project)

	mcpEntry := map[string]interface{}{
		"type":    "stdio",
		"command": "pantry",
		"args":    []string{"mcp"},
		"env":     map[string]interface{}{},
	}

	var configPath string
	if project {
		// Project scope: write to .mcp.json in the current directory.
		// This is checked into source control and shared with the team.
		cwd, _ := os.Getwd()
		configPath = filepath.Join(cwd, ".mcp.json")
		if err := writeMCPJSON(configPath, mcpEntry); err != nil {
			return nil, err
		}
	} else {
		// User scope: write to ~/.claude.json top-level mcpServers.
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".claude.json")
		if err := writeClaudeJSONUserMCP(configPath, mcpEntry); err != nil {
			return nil, err
		}
	}

	msg := fmt.Sprintf("Installed Pantry MCP server in %s", configPath)
	if installSkill(skillTarget) {
		msg += " and skill"
	}

	return map[string]string{"message": msg}, nil
}

// writeMCPJSON writes an MCP server entry into a .mcp.json file (project scope).
func writeMCPJSON(configPath string, entry map[string]interface{}) error {
	var config map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]interface{})
	}

	mcpServers, _ := config["mcpServers"].(map[string]interface{})
	if mcpServers == nil {
		mcpServers = make(map[string]interface{})
		config["mcpServers"] = mcpServers
	}
	mcpServers["pantry"] = entry

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

// writeClaudeJSONUserMCP writes an MCP server entry into ~/.claude.json top-level mcpServers (user scope).
func writeClaudeJSONUserMCP(configPath string, entry map[string]interface{}) error {
	var root map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &root); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		root = make(map[string]interface{})
	}

	mcpServers, _ := root["mcpServers"].(map[string]interface{})
	if mcpServers == nil {
		mcpServers = make(map[string]interface{})
		root["mcpServers"] = mcpServers
	}
	mcpServers["pantry"] = entry

	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
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

	if err := os.MkdirAll(target, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Codex uses [mcp_servers.<name>] in config.toml.
	// Only append the block if it's not already present (idempotent).
	const pantryTOML = "\n[mcp_servers.pantry]\ncommand = \"pantry\"\nargs = [\"mcp\"]\n"
	existing, _ := os.ReadFile(configPath)
	if !bytes.Contains(existing, []byte("[mcp_servers.pantry]")) {
		f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open config file: %w", err)
		}
		_, writeErr := f.WriteString(pantryTOML)
		f.Close()
		if writeErr != nil {
			return nil, fmt.Errorf("failed to write config: %w", writeErr)
		}
	}

	// Add to AGENTS.md (idempotent).
	const pantryAgentsSection = "## Pantry\n\nYou have access to a persistent note storage system via Pantry. " +
		"Use it to store important decisions, patterns, bugs, context, and learnings.\n\n" +
		"### Commands\n" +
		"- `pantry store` - Store a note\n" +
		"- `pantry search` - Search notes\n" +
		"- `pantry list` - List recent notes\n"
	existingAgents, _ := os.ReadFile(agentsPath)
	if !bytes.Contains(existingAgents, []byte("## Pantry")) {
		f2, err := os.OpenFile(agentsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			f2.WriteString(pantryAgentsSection)
			f2.Close()
		}
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

	// OpenCode uses a "mcp" key (not "mcpServers"), and command must be an array.
	mcp, _ := config["mcp"].(map[string]interface{})
	if mcp == nil {
		mcp = make(map[string]interface{})
		config["mcp"] = mcp
	}

	mcp["pantry"] = map[string]interface{}{
		"type":    "local",
		"command": []string{"pantry", "mcp"},
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
	skillTarget := resolveConfigDir(".claude", configDir, project)

	var configPath string
	if project {
		cwd, _ := os.Getwd()
		configPath = filepath.Join(cwd, ".mcp.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return map[string]string{"message": "Pantry not found in project .mcp.json"}, nil
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
	} else {
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".claude.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return map[string]string{"message": "Pantry not found in Claude Code config"}, nil
		}
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		var root map[string]interface{}
		if err := json.Unmarshal(data, &root); err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}
		if mcpServers, ok := root["mcpServers"].(map[string]interface{}); ok {
			delete(mcpServers, "pantry")
		}
		newData, err := json.MarshalIndent(root, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		if err := os.WriteFile(configPath, newData, 0644); err != nil {
			return nil, fmt.Errorf("failed to write config: %w", err)
		}
	}

	msg := fmt.Sprintf("Removed Pantry from %s", configPath)
	if uninstallSkill(skillTarget) {
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

	if mcp, ok := config["mcp"].(map[string]interface{}); ok {
		delete(mcp, "pantry")
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
