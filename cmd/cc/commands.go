package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/ai"
	"github.com/myersguo/cc-mono/pkg/ai/providers/openai"
	"github.com/myersguo/cc-mono/pkg/codingagent"
	"github.com/myersguo/cc-mono/pkg/codingagent/extensions"
	"github.com/myersguo/cc-mono/pkg/codingagent/tools"
	"github.com/myersguo/cc-mono/pkg/shared"
	"github.com/spf13/cobra"

	// Import example extensions to register them
	_ "github.com/myersguo/cc-mono/extensions/example"
)

var (
	// Global flags
	configPath     string
	modelsPath     string
	providersPath  string
	themeName      string
	verbose        bool
	workingDir     string
	modelID        string
	providerName   string
	extensionNames []string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "cc",
	Short: "CC-Mono - AI coding agent",
	Long: `CC-Mono is an interactive AI coding agent that helps with software development tasks.
It supports multiple LLM providers, has a rich TUI, and an extensible plugin system.`,
	RunE: runChat,
}

// chatCmd starts an interactive chat session
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session",
	Long:  "Start an interactive chat session with the AI agent using the TUI interface.",
	RunE:  runChat,
}

// modelCmd manages models
var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "Manage AI models",
	Long:  "List available models, show current model, or switch to a different model.",
}

// modelListCmd lists available models
var modelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available models",
	Long:  "List all available AI models from the registry.",
	RunE: func(cmd *cobra.Command, args []string) error {
		resolvedModelsPath := resolveConfigPath(modelsPath)

		registry := codingagent.NewModelRegistry()
		if err := registry.LoadFromFile(resolvedModelsPath); err != nil {
			return fmt.Errorf("failed to load models: %w", err)
		}

		models := registry.List()
		if len(models) == 0 {
			fmt.Println("No models available.")
			return nil
		}

		fmt.Printf("Available models (%d):\n", len(models))
		for _, model := range models {
			fmt.Printf("  - %s\n", model.ID)
			if verbose {
				fmt.Printf("    Provider: %s\n", model.Provider)
				fmt.Printf("    Name: %s\n", model.Name)
				fmt.Printf("    Context Window: %d tokens\n", model.ContextWindow)
				fmt.Printf("    Max Output: %d tokens\n", model.MaxOutput)
				fmt.Printf("    Cost: $%.4f/1M input, $%.4f/1M output\n",
					model.InputCostPer1M, model.OutputCostPer1M)
			}
		}

		return nil
	},
}

// sessionCmd manages sessions
var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage chat sessions",
	Long:  "Create, list, delete, or fork chat sessions.",
}

// sessionListCmd lists all sessions
var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions",
	Long:  "List all saved chat sessions with their metadata.",
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionsDir, err := getSessionsDir()
		if err != nil {
			return err
		}

		sessionMgr, err := codingagent.NewSessionManager(sessionsDir)
		if err != nil {
			return fmt.Errorf("failed to create session manager: %w", err)
		}

		sessions, err := sessionMgr.List()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}

		if len(sessions) == 0 {
			fmt.Println("No sessions found.")
			return nil
		}

		fmt.Printf("Found %d session(s):\n", len(sessions))
		for _, meta := range sessions {
			fmt.Printf("  ID: %s\n", meta.ID)
			fmt.Printf("    Title: %s\n", meta.Title)
			fmt.Printf("    Created: %s\n", meta.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("    Updated: %s\n", meta.UpdatedAt.Format("2006-01-02 15:04:05"))
			if meta.ParentID != "" {
				fmt.Printf("    Parent: %s (branch at message %d)\n", meta.ParentID, meta.BranchPoint)
			}
			if len(meta.Tags) > 0 {
				fmt.Printf("    Tags: %v\n", meta.Tags)
			}
			fmt.Println()
		}

		return nil
	},
}

// sessionDeleteCmd deletes a session
var sessionDeleteCmd = &cobra.Command{
	Use:   "delete [session-id]",
	Short: "Delete a session",
	Long:  "Delete a saved chat session by ID.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]

		sessionsDir, err := getSessionsDir()
		if err != nil {
			return err
		}

		sessionMgr, err := codingagent.NewSessionManager(sessionsDir)
		if err != nil {
			return fmt.Errorf("failed to create session manager: %w", err)
		}

		if err := sessionMgr.Delete(sessionID); err != nil {
			return fmt.Errorf("failed to delete session: %w", err)
		}

		fmt.Printf("Session %s deleted successfully.\n", sessionID)
		return nil
	},
}

// extensionCmd manages extensions
var extensionCmd = &cobra.Command{
	Use:   "extension",
	Short: "Manage extensions",
	Long:  "List available extensions.",
}

// extensionListCmd lists available extensions
var extensionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available extensions",
	Long:  "List all registered extensions in the global registry.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Import example extensions to register them
		// This happens in their init() functions
		exts := shared.GlobalRegistry.List()
		if len(exts) == 0 {
			fmt.Println("No extensions available.")
			return nil
		}

		fmt.Printf("Available extensions (%d):\n", len(exts))
		for _, ext := range exts {
			fmt.Printf("  - %s (v%s)\n", ext.Name(), ext.Version())
			if verbose {
				fmt.Printf("    Description: %s\n", ext.Description())
			}
		}

		return nil
	},
}

// versionCmd shows version information
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display the version and build information for CC-Mono.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("CC-Mono v1.0.0")
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Config directory path")
	rootCmd.PersistentFlags().StringVar(&modelsPath, "models", "configs/models.json", "Models configuration file")
	rootCmd.PersistentFlags().StringVar(&providersPath, "providers", "configs/providers.json", "Providers configuration file")
	rootCmd.PersistentFlags().StringVar(&themeName, "theme", "dark", "TUI theme (dark/light)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().StringVar(&workingDir, "dir", ".", "Working directory")
	rootCmd.PersistentFlags().StringVar(&modelID, "model", "", "Model ID to use")
	rootCmd.PersistentFlags().StringVar(&providerName, "provider", "", "Provider to use")
	rootCmd.PersistentFlags().StringSliceVar(&extensionNames, "extensions", nil, "Extension names to load")

	// Add subcommands
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(modelCmd)
	rootCmd.AddCommand(sessionCmd)
	rootCmd.AddCommand(extensionCmd)
	rootCmd.AddCommand(versionCmd)

	// Model subcommands
	modelCmd.AddCommand(modelListCmd)

	// Session subcommands
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionDeleteCmd)

	// Extension subcommands
	extensionCmd.AddCommand(extensionListCmd)
}

// runChat starts the interactive chat TUI
func runChat(cmd *cobra.Command, args []string) error {
	// Resolve paths
	resolvedModelsPath := modelsPath
	resolvedProvidersPath := providersPath

	// If config directory is specified, use it for relative paths
	if configPath != "" {
		if !filepath.IsAbs(modelsPath) {
			resolvedModelsPath = filepath.Join(configPath, filepath.Base(modelsPath))
		}
		if !filepath.IsAbs(providersPath) {
			resolvedProvidersPath = filepath.Join(configPath, filepath.Base(providersPath))
		}
	}

	// Load model registry
	modelRegistry := codingagent.NewModelRegistry()
	if err := modelRegistry.LoadFromFile(resolvedModelsPath); err != nil {
		return fmt.Errorf("failed to load models: %w", err)
	}

	// Load providers config
	providersConfig, err := codingagent.LoadProvidersConfig(resolvedProvidersPath)
	if err != nil {
		return fmt.Errorf("failed to load providers config: %w", err)
	}

	// Get provider name (use flag, or default from config, or "openai")
	if providerName == "" {
		providerName = providersConfig.DefaultProvider
		if providerName == "" {
			providerName = "openai"
		}
	}

	// Get provider config
	providerConfig, ok := providersConfig.Providers[providerName]
	if !ok {
		return fmt.Errorf("provider %s not found in config", providerName)
	}

	// Get model ID (use flag, or default from provider config)
	if modelID == "" {
		modelID = providerConfig.DefaultModel
		if modelID == "" {
			return fmt.Errorf("no model specified and no default model in provider config")
		}
	}

	// Get AI model from registry
	aiModel, err := modelRegistry.ToAIModel(modelID)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	// Create provider
	provider, err := createProvider(providerName, providerConfig)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Get working directory
	if workingDir == "." {
		workingDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Create tools
	agentTools := []agent.AgentTool{
		tools.CreateReadTool(workingDir),
		tools.CreateWriteTool(workingDir),
		tools.CreateEditTool(workingDir),
		tools.CreateBashTool(workingDir),
	}

	// Load extensions
	extensionLoader := extensions.NewLoader()
	if len(extensionNames) > 0 {
		if err := extensionLoader.LoadFromRegistry(extensionNames, nil); err != nil {
			return fmt.Errorf("failed to load extensions: %w", err)
		}
	}

	// Create extension runner
	extensionRunner := extensions.NewRunner(extensionLoader)

	// Get additional tools from extensions
	extensionTools := extensionRunner.GetRegisteredTools()
	agentTools = append(agentTools, extensionTools...)

	// Wrap tools with extension hooks
	agentTools = extensionRunner.WrapAllTools(agentTools)

	// System prompt
	systemPrompt := `You are a helpful AI coding assistant. You can:
- Read files from the filesystem
- Write new files or update existing files
- Edit files with precise text replacements
- Run bash commands

When working with code:
1. Always read files before editing them
2. Make precise, targeted changes
3. Test your changes when possible
4. Explain what you're doing

Be concise but thorough in your responses.`

	// Start TUI
	return runTUI(provider, aiModel, systemPrompt, agentTools, themeName, extensionRunner)
}

// createProvider creates a provider based on name and config
// All providers use the OpenAI-compatible API (openai, deepseek, qwen, etc.)
func createProvider(name string, config codingagent.ProviderConfig) (ai.Provider, error) {
	// All providers use OpenAI-compatible API
	// This includes: openai, deepseek, qwen, moonshot, zhipu, ollama, etc.
	return openai.NewProvider(openai.Config{
		APIKey:  config.APIKey,
		BaseURL: config.BaseURL,
		Model:   config.DefaultModel,
	})
}

// resolveConfigPath resolves a config file path based on --config flag
func resolveConfigPath(path string) string {
	// If path is absolute, use it as-is
	if filepath.IsAbs(path) {
		return path
	}

	// If config directory is specified, join with base name
	if configPath != "" {
		return filepath.Join(configPath, filepath.Base(path))
	}

	// Otherwise, use path as-is (relative to current directory)
	return path
}

// getConfigDir returns the config directory path
func getConfigDir() (string, error) {
	if configPath != "" {
		return configPath, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, ".cc-mono"), nil
}

// getSessionsDir returns the sessions directory path
func getSessionsDir() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "sessions"), nil
}

// Execute runs the root command
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}
