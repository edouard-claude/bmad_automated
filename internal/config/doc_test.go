package config_test

import (
	"fmt"

	"github.com/edouard-claude/bmad_automated/internal/config"
)

// This example demonstrates loading configuration using DefaultConfig
// as a fallback when no config file exists. The Loader merges file
// configuration with environment variables and built-in defaults.
func Example_loader() {
	// NewLoader creates a loader that uses Viper for configuration management.
	// Load() reads from config files and environment variables.
	loader := config.NewLoader()

	// Load attempts to read configuration from:
	// 1. BMAD_CONFIG_PATH environment variable
	// 2. ./config/workflows.yaml
	// 3. Falls back to DefaultConfig() if no file found
	cfg, err := loader.Load()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// DefaultConfig provides sensible defaults for all settings
	fmt.Println("Claude binary:", cfg.Claude.BinaryPath)
	fmt.Println("Output format:", cfg.Claude.OutputFormat)
	// Output:
	// Claude binary: claude
	// Output format: stream-json
}

// This example demonstrates expanding workflow prompt templates with story data.
// Workflow prompts use Go's text/template syntax with {{.StoryKey}} for the story.
func Example_getPrompt() {
	// Start with default configuration containing standard workflows
	cfg := config.DefaultConfig()

	// GetPrompt expands the template for a workflow with the given story key.
	// The template uses {{.StoryKey}} to reference the story identifier.
	prompt, err := cfg.GetPrompt("create-story", "7-1-define-schema")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Prompt now contains the expanded template with the story key
	fmt.Println("Prompt contains story key:", contains(prompt, "7-1-define-schema"))

	// Unknown workflows return an error
	_, err = cfg.GetPrompt("unknown-workflow", "story-key")
	fmt.Println("Unknown workflow error:", err != nil)
	// Output:
	// Prompt contains story key: true
	// Unknown workflow error: true
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
