package task

import (
	"bufio"
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
	scanner := bufio.NewScanner(os.Stdin)

	reader := func(prompt string) string {
		fmt.Print(prompt)
		scanner.Scan()
		return strings.TrimSpace(scanner.Text())
	}

	defaultProjectPath := os.Getenv(envProjectPath)
	if defaultProjectPath == "" {
		if cwd, err := os.Getwd(); err == nil {
			defaultProjectPath = cwd
		}
	}
	projectPath := reader(fmt.Sprintf("Project path [%s]: ", defaultProjectPath))
	if projectPath == "" {
		projectPath = defaultProjectPath
	}

	apiKey := reader("API key (optional): ")
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
	baseURL := reader(fmt.Sprintf("Base URL [%s]: ", defaultBase))
	if baseURL == "" {
		baseURL = defaultBase
	}

	cfg := &TaskConfig{
		PilotDeck: PilotDeckConfig{
			BaseURL:     baseURL,
			APIKey:      apiKey,
			ProjectPath: projectPath,
		},
	}
	return SaveGlobalConfig(cfg)
}

// CreateTaskWizard interactively creates a new task directory and task-level config.
func CreateTaskWizard(taskName string) error {
	// First ensure global config exists
	if err := ensureGlobalConfig(); err != nil {
		return fmt.Errorf("global config: %w", err)
	}

	taskDir := TaskDir(taskName)

	// Create directory structure
	for _, dir := range []string{taskDir, filepath.Join(taskDir, jobsDirName), filepath.Join(taskDir, logsDirName)} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	scanner := bufio.NewScanner(os.Stdin)

	reader := func(prompt string) string {
		fmt.Print(prompt)
		scanner.Scan()
		return strings.TrimSpace(scanner.Text())
	}

	debugStr := reader("Debug mode? (y/N): ")
	debug := strings.EqualFold(debugStr, "y") || strings.EqualFold(debugStr, "yes")

	cfg := &TaskConfig{Debug: debug}

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

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Printf("Debug mode [%v] (y/N): ", cfg.Debug)
	scanner.Scan()
	debugStr := strings.TrimSpace(scanner.Text())
	if debugStr != "" {
		cfg.Debug = strings.EqualFold(debugStr, "y") || strings.EqualFold(debugStr, "yes")
	}

	if err := SaveConfig(taskDir, cfg); err != nil {
		return err
	}

	fmt.Println("Configuration updated.")
	return nil
}
