package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	setupConfigDir string
	setupProject   bool
)

type agentFunc func(configDir string, project bool) (map[string]string, error)

func runAgentCmd(agent string, handlers map[string]agentFunc, configDir string, project bool) {
	fn, ok := handlers[agent]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: unknown agent: %s\n", agent)
		fmt.Fprintf(os.Stderr, "Supported agents: claude, cursor, windsurf, antigravity, codex, opencode, roocode\n")
		os.Exit(1)
	}

	result, err := fn(configDir, project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result["message"])
}

var setupCmd = &cobra.Command{
	Use:   "setup [agent]",
	Short: "Install Pantry hooks for an agent",
	Args:  cobra.ExactArgs(1),
	//nolint:revive
	Run: func(cmd *cobra.Command, args []string) {
		runAgentCmd(args[0], map[string]agentFunc{
			"claude":      setupClaudeCode,
			"claude-code": setupClaudeCode,
			"cursor":      setupCursor,
			"windsurf":    setupWindsurf,
			"antigravity": setupAntigravity,
			"codex":       setupCodex,
			"opencode":    func(_ string, project bool) (map[string]string, error) { return setupOpenCode(project) },
			"roo":         setupRooCode,
			"roocode":     setupRooCode,
		}, setupConfigDir, setupProject)
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [agent]",
	Short: "Remove Pantry hooks for an agent",
	Args:  cobra.ExactArgs(1),
	//nolint:revive
	Run: func(cmd *cobra.Command, args []string) {
		runAgentCmd(args[0], map[string]agentFunc{
			"claude":      uninstallClaudeCode,
			"claude-code": uninstallClaudeCode,
			"cursor":      uninstallCursor,
			"windsurf":    uninstallWindsurf,
			"antigravity": uninstallAntigravity,
			"codex":       uninstallCodex,
			"opencode":    func(_ string, project bool) (map[string]string, error) { return uninstallOpenCode(project) },
			"roo":         uninstallRooCode,
			"roocode":     uninstallRooCode,
		}, setupConfigDir, setupProject)
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

	mcpEntry := map[string]any{
		"type":    "stdio",
		"command": "pantry",
		"args":    []string{"mcp"},
		"env":     map[string]any{},
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

	msg := "Installed Pantry MCP server in " + configPath
	if installSkill(skillTarget) {
		msg += " and skill" //nolint:goconst
	}

	return map[string]string{"message": msg}, nil
}

// writeMCPJSON writes an MCP server entry into a .mcp.json file (project scope).
func writeMCPJSON(configPath string, entry map[string]any) error {
	var config map[string]any
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]any)
	}

	mcpServers, _ := config["mcpServers"].(map[string]any)
	if mcpServers == nil {
		mcpServers = make(map[string]any)
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
func writeClaudeJSONUserMCP(configPath string, entry map[string]any) error {
	var root map[string]any
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &root); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		root = make(map[string]any)
	}

	mcpServers, _ := root["mcpServers"].(map[string]any)
	if mcpServers == nil {
		mcpServers = make(map[string]any)
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
	var config map[string]any
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]any)
	}

	// Add MCP server config
	mcpServers, ok := config["mcpServers"].(map[string]any)
	if !ok {
		mcpServers = make(map[string]any)
		config["mcpServers"] = mcpServers
	}

	mcpServers["pantry"] = map[string]any{
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

	msg := "Installed Pantry MCP server in " + configPath
	if installSkill(target) {
		msg += " and skill"
	}

	return map[string]string{"message": msg}, nil
}

func setupWindsurf(configDir string, project bool) (map[string]string, error) {
	var target string
	if configDir != "" {
		target = configDir
	} else if project {
		cwd, _ := os.Getwd()
		target = filepath.Join(cwd, ".codeium", "windsurf")
	} else {
		home, _ := os.UserHomeDir()
		target = filepath.Join(home, ".codeium", "windsurf")
	}

	configPath := filepath.Join(target, "mcp_config.json")

	// Read existing config or create new
	var config map[string]any
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]any)
	}

	// Add MCP server config
	mcpServers, ok := config["mcpServers"].(map[string]any)
	if !ok {
		mcpServers = make(map[string]any)
		config["mcpServers"] = mcpServers
	}

	mcpServers["pantry"] = map[string]any{
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

	msg := "Installed Pantry MCP server in " + configPath
	if installSkill(target) {
		msg += " and skill"
	}

	return map[string]string{"message": msg}, nil
}

func setupAntigravity(configDir string, project bool) (map[string]string, error) {
	var target string
	if configDir != "" {
		target = configDir
	} else if project {
		cwd, _ := os.Getwd()
		target = filepath.Join(cwd, ".gemini", "antigravity")
	} else {
		home, _ := os.UserHomeDir()
		target = filepath.Join(home, ".gemini", "antigravity")
	}

	configPath := filepath.Join(target, "mcp_config.json")

	// Read existing config or create new
	var config map[string]any
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]any)
	}

	// Add MCP server config
	mcpServers, ok := config["mcpServers"].(map[string]any)
	if !ok {
		mcpServers = make(map[string]any)
		config["mcpServers"] = mcpServers
	}

	mcpServers["pantry"] = map[string]any{
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

	msg := "Installed Pantry MCP server in " + configPath
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
		closeErr := f.Close()

		if writeErr != nil {
			return nil, fmt.Errorf("failed to write config: %w", writeErr)
		}

		if closeErr != nil {
			return nil, fmt.Errorf("failed to close config file: %w", closeErr)
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
			_, _ = f2.WriteString(pantryAgentsSection)
			_ = f2.Close()
		}
	}

	msg := "Installed Pantry in " + target
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
	var config map[string]any
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]any)
	}

	// OpenCode uses a "mcp" key (not "mcpServers"), and command must be an array.
	mcp, _ := config["mcp"].(map[string]any)
	if mcp == nil {
		mcp = make(map[string]any)
		config["mcp"] = mcp
	}

	mcp["pantry"] = map[string]any{
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
		"message": "Installed Pantry MCP server in " + configPath,
	}, nil
}

// removePantryFromMCPJSON reads a JSON config file, removes the "pantry" key from
// "mcpServers", and writes the result back.
func removePantryFromMCPJSON(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if mcpServers, ok := config["mcpServers"].(map[string]any); ok {
		delete(mcpServers, "pantry")
	}

	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
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
	} else {
		home, _ := os.UserHomeDir()

		configPath = filepath.Join(home, ".claude.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return map[string]string{"message": "Pantry not found in Claude Code config"}, nil
		}
	}

	if err := removePantryFromMCPJSON(configPath); err != nil {
		return nil, err
	}

	msg := "Removed Pantry from " + configPath
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

	if err := removePantryFromMCPJSON(configPath); err != nil {
		return nil, err
	}

	msg := "Removed Pantry from " + configPath
	if uninstallSkill(target) {
		msg += " and skill"
	}

	return map[string]string{"message": msg}, nil
}

func uninstallWindsurf(configDir string, project bool) (map[string]string, error) {
	var target string
	if configDir != "" {
		target = configDir
	} else if project {
		cwd, _ := os.Getwd()
		target = filepath.Join(cwd, ".codeium", "windsurf")
	} else {
		home, _ := os.UserHomeDir()
		target = filepath.Join(home, ".codeium", "windsurf")
	}

	configPath := filepath.Join(target, "mcp_config.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return map[string]string{"message": "Pantry not found in Windsurf config"}, nil
	}

	if err := removePantryFromMCPJSON(configPath); err != nil {
		return nil, err
	}

	msg := "Removed Pantry from " + configPath
	if uninstallSkill(target) {
		msg += " and skill"
	}

	return map[string]string{"message": msg}, nil
}

func uninstallAntigravity(configDir string, project bool) (map[string]string, error) {
	var target string
	if configDir != "" {
		target = configDir
	} else if project {
		cwd, _ := os.Getwd()
		target = filepath.Join(cwd, ".gemini", "antigravity")
	} else {
		home, _ := os.UserHomeDir()
		target = filepath.Join(home, ".gemini", "antigravity")
	}

	configPath := filepath.Join(target, "mcp_config.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return map[string]string{"message": "Pantry not found in Antigravity config"}, nil
	}

	if err := removePantryFromMCPJSON(configPath); err != nil {
		return nil, err
	}

	msg := "Removed Pantry from " + configPath
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

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if mcp, ok := config["mcp"].(map[string]any); ok {
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
		"message": "Removed Pantry from " + configPath,
	}, nil
}

func setupRooCode(configDir string, project bool) (map[string]string, error) {
	var target string
	//nolint:gocritic
	if configDir != "" {
		target = configDir
	} else if project {
		cwd, _ := os.Getwd()
		target = filepath.Join(cwd, ".roo")
	} else {
		return nil, errors.New("RooCode global MCP config is managed via VS Code settings.\nUse --project (-p) to install in the current project's .roo/mcp.json instead")
	}

	configPath := filepath.Join(target, "mcp.json")

	var config map[string]any
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]any)
	}

	mcpServers, _ := config["mcpServers"].(map[string]any)
	if mcpServers == nil {
		mcpServers = make(map[string]any)
		config["mcpServers"] = mcpServers
	}

	mcpServers["pantry"] = map[string]any{
		"command": "pantry",
		"args":    []string{"mcp"},
	}

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

	return map[string]string{
		"message": "Installed Pantry MCP server in " + configPath,
	}, nil
}

func uninstallRooCode(configDir string, project bool) (map[string]string, error) {
	var target string
	//nolint:gocritic
	if configDir != "" {
		target = configDir
	} else if project {
		cwd, _ := os.Getwd()
		target = filepath.Join(cwd, ".roo")
	} else {
		return nil, errors.New("RooCode global MCP config is managed via VS Code settings.\nUse --project (-p) to uninstall from the current project's .roo/mcp.json instead")
	}

	configPath := filepath.Join(target, "mcp.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return map[string]string{"message": "Pantry not found in RooCode config"}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if mcpServers, ok := config["mcpServers"].(map[string]any); ok {
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
		"message": "Removed Pantry from " + configPath,
	}, nil
}
