package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ensureGlobalConfig checks if config.yaml exists; if not, interactively creates it.
func ensureGlobalConfig() error {
	if HasGlobalConfig() {
		return nil
	}

	fmt.Println("No global config.yaml found. Let's set up PilotDeck connection.")

	defaultProjectPath := os.Getenv(envProjectPath)
	if defaultProjectPath == "" {
		if cwd, err := os.Getwd(); err == nil {
			defaultProjectPath = filepath.Join(cwd, "working")
		}
	}
	fmt.Printf("Project path [%s]: ", defaultProjectPath)
	projectPath := ReadLine()
	if projectPath == "" {
		projectPath = defaultProjectPath
	}

	fmt.Print("API key (optional): ")
	apiKey := ReadLine()
	if apiKey == "" {
		fmt.Println("  (no API key will be sent)")
	}

	host := os.Getenv(envHost)
	port := os.Getenv(envPort)
	defaultBase := ""
	if host != "" && port != "" {
		defaultBase = fmt.Sprintf("http://%s:%s", host, port)
	}
	if defaultBase == "" {
		defaultBase = "http://localhost:3001"
	}
	fmt.Printf("Base URL [%s]: ", defaultBase)
	baseURL := ReadLine()
	if baseURL == "" {
		baseURL = defaultBase
	}

	cfg := &TaskConfig{
		PilotDeck: PilotDeckConfig{
			BaseURL:          baseURL,
			APIKey:           apiKey,
			ProjectPath:      projectPath,
			RetryAttempts:    3,
			RetryIntervalSec: 20,
		},
	}
	return SaveGlobalConfig(cfg)
}

// CreateTaskWizard interactively creates a new task directory and task-level config.
func CreateTaskWizard(taskName string) error {
	if err := ensureGlobalConfig(); err != nil {
		return fmt.Errorf("global config: %w", err)
	}

	taskDir := TaskDir(taskName)
	for _, dir := range []string{taskDir, filepath.Join(taskDir, jobsDirName), filepath.Join(taskDir, logsDirName)} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	fmt.Print("Debug mode? (y/N): ")
	debugStr := ReadLine()
	debug := strings.EqualFold(debugStr, "y") || strings.EqualFold(debugStr, "yes")

	fmt.Print("Task mode [continuous] (continuous/discrete): ")
	modeStr := ReadLine()
	mode := strings.TrimSpace(strings.ToLower(modeStr))
	if mode == "" {
		mode = "continuous"
	}
	if mode != "continuous" && mode != "discrete" {
		fmt.Println("Invalid mode, defaulting to continuous.")
		mode = "continuous"
	}

	cfg := &TaskConfig{Debug: debug, Mode: mode}
	if err := SaveConfig(taskDir, cfg); err != nil {
		return err
	}

	fmt.Println("\nTask created successfully!")
	fmt.Printf("  Config:     %s\n", ConfigPath(taskDir))
	fmt.Printf("  Jobs dir:   %s/\n", filepath.Join(taskDir, jobsDirName))
	fmt.Printf("  Logs dir:   %s/\n", filepath.Join(taskDir, logsDirName))
	fmt.Println("\nPlace your job files in the jobs/ directory with format: 001_xxx.txt")
	fmt.Println("Then run: work-watch " + taskName)
	return nil
}

// ReconfigureWizard interactively updates task-level config only.
func ReconfigureWizard(taskName string) error {
	taskDir := TaskDir(taskName)
	cfg, err := LoadConfig(taskDir)
	if err != nil {
		return fmt.Errorf("load existing config: %w", err)
	}

	fmt.Printf("Debug mode [%v] (y/N): ", cfg.Debug)
	debugStr := ReadLine()
	if debugStr != "" {
		cfg.Debug = strings.EqualFold(debugStr, "y") || strings.EqualFold(debugStr, "yes")
	}

	if err := SaveConfig(taskDir, cfg); err != nil {
		return err
	}

	fmt.Println("Configuration updated.")
	return nil
}
