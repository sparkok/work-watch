package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DefaultBaseURL       = "http://localhost:3001"
	GlobalConfigFileName = "config.yaml"
	configFileName       = "task.yaml"

	envHost        = "HOST"
	envPort        = "PORT"
	envAPIKey      = "API_KEY"
	envProjectPath = "PROJECT_PATH"
)

// AppDir is the path to the work-watch application root (set at startup).
var AppDir string

// TaskConfig describes a work-watch task.
type TaskConfig struct {
	PilotDeck PilotDeckConfig `yaml:"pilotdeck"`
	Debug     bool            `yaml:"debug"`
	SessionID string          `yaml:"session_id,omitempty"`
}

// PilotDeckConfig holds server connection details.
type PilotDeckConfig struct {
	BaseURL     string `yaml:"base_url"`
	APIKey      string `yaml:"api_key"`
	ProjectPath string `yaml:"project_path"`
}

// taskYAML is the subset written to per-task task.yaml (no PilotDeck settings).
type taskYAML struct {
	Debug     bool   `yaml:"debug"`
	SessionID string `yaml:"session_id,omitempty"`
}

// GlobalConfigPath returns the path to the global config.yaml.
func GlobalConfigPath() string {
	return filepath.Join(AppDir, GlobalConfigFileName)
}

// ConfigPath returns the path to task.yaml inside taskDir.
func ConfigPath(taskDir string) string {
	return filepath.Join(taskDir, configFileName)
}

// envBaseURL builds a base URL from HOST and PORT environment variables.
func envBaseURL() string {
	host := os.Getenv(envHost)
	port := os.Getenv(envPort)
	if host != "" && port != "" {
		return fmt.Sprintf("http://%s:%s", host, port)
	}
	return ""
}

// applyDefaults fills empty PilotDeck fields with env var or hardcoded defaults.
func applyDefaults(cfg *TaskConfig) {
	if cfg.PilotDeck.BaseURL == "" {
		if u := envBaseURL(); u != "" {
			cfg.PilotDeck.BaseURL = u
		} else {
			cfg.PilotDeck.BaseURL = DefaultBaseURL
		}
	}
	if cfg.PilotDeck.APIKey == "" {
		if k := os.Getenv(envAPIKey); k != "" {
			cfg.PilotDeck.APIKey = k
		}
	}
	if cfg.PilotDeck.ProjectPath == "" {
		if p := os.Getenv(envProjectPath); p != "" {
			cfg.PilotDeck.ProjectPath = p
		} else {
			if cwd, err := os.Getwd(); err == nil {
				cfg.PilotDeck.ProjectPath = cwd
			}
		}
	}
}

// LoadConfig reads global config.yaml (PilotDeck settings) and merges with
// per-task task.yaml. Per-task values override global ones.
func LoadConfig(taskDir string) (*TaskConfig, error) {
	cfg := &TaskConfig{}

	// 1. Load global config (PilotDeck connection)
	if raw, err := os.ReadFile(GlobalConfigPath()); err == nil {
		var global struct {
			PilotDeck PilotDeckConfig `yaml:"pilotdeck"`
		}
		if err := yaml.Unmarshal(raw, &global); err == nil {
			cfg.PilotDeck = global.PilotDeck
		}
	}

	// 2. Load per-task config (debug, session_id)
	if raw, err := os.ReadFile(ConfigPath(taskDir)); err == nil {
		var task taskYAML
		if err := yaml.Unmarshal(raw, &task); err == nil {
			cfg.Debug = task.Debug
			cfg.SessionID = task.SessionID
		}
	}

	// 3. Apply env defaults for missing PilotDeck fields
	applyDefaults(cfg)

	// 4. Validate
	if cfg.PilotDeck.ProjectPath == "" {
		return nil, errors.New("pilotdeck.project_path is required (set in config.yaml or env)")
	}
	return cfg, nil
}

// SaveConfig writes only task-specific fields (debug, session_id) to task.yaml.
func SaveConfig(taskDir string, cfg *TaskConfig) error {
	task := taskYAML{
		Debug:     cfg.Debug,
		SessionID: cfg.SessionID,
	}
	raw, err := yaml.Marshal(&task)
	if err != nil {
		return fmt.Errorf("marshal task config: %w", err)
	}
	if err := os.WriteFile(ConfigPath(taskDir), raw, 0644); err != nil {
		return fmt.Errorf("write task config: %w", err)
	}
	return nil
}

// SaveGlobalConfig writes PilotDeck settings to config.yaml.
func SaveGlobalConfig(cfg *TaskConfig) error {
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal app config: %w", err)
	}
	if err := os.WriteFile(GlobalConfigPath(), raw, 0644); err != nil {
		return fmt.Errorf("write app config: %w", err)
	}
	return nil
}

// SaveSessionID updates the session_id field in the per-task config.
func SaveSessionID(taskDir, sessionID string) error {
	// Load task-only config to preserve debug setting
	cfg := &TaskConfig{}
	if raw, err := os.ReadFile(ConfigPath(taskDir)); err == nil {
		var task taskYAML
		if err := yaml.Unmarshal(raw, &task); err == nil {
			cfg.Debug = task.Debug
			cfg.SessionID = task.SessionID
		}
	}
	if cfg.SessionID == sessionID {
		return nil
	}
	cfg.SessionID = sessionID
	return SaveConfig(taskDir, cfg)
}

// HasGlobalConfig checks whether config.yaml exists.
func HasGlobalConfig() bool {
	_, err := os.Stat(GlobalConfigPath())
	return err == nil
}
