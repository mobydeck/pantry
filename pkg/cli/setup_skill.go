package cli

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed skills/pantry/SKILL.md
var skillContent []byte

// installSkill installs the Pantry SKILL.md into an agent's skills directory.
// agentHome: path to the agent's config directory (e.g. ~/.claude, ~/.cursor, ~/.codex).
// Returns true if skill was installed, false if already present.
func installSkill(agentHome string) bool {
	skillDir := filepath.Join(agentHome, "skills", "pantry")
	skillPath := filepath.Join(skillDir, "SKILL.md")

	if _, err := os.Stat(skillPath); err == nil {
		return false
	}

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return false
	}

	if err := os.WriteFile(skillPath, skillContent, 0644); err != nil {
		return false
	}

	return true
}

// uninstallSkill removes the Pantry skill from an agent's skills directory.
// Returns true if skill was removed, false if not found.
func uninstallSkill(agentHome string) bool {
	skillDir := filepath.Join(agentHome, "skills", "pantry")

	info, err := os.Stat(skillDir)
	if err != nil {
		return false
	}

	if info.IsDir() {
		if err := os.RemoveAll(skillDir); err != nil {
			return false
		}
	} else {
		// Symlink
		if err := os.Remove(skillDir); err != nil {
			return false
		}
	}

	// Remove the parent skills/ dir if now empty.
	skillsDir := filepath.Join(agentHome, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err == nil && len(entries) == 0 {
		os.Remove(skillsDir)
	}

	return true
}
