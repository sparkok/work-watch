package task

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
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
	Mode      string          `yaml:"mode,omitempty"`
	Lang      string          `yaml:"lang,omitempty"`
	Label     string          `yaml:"label,omitempty"`
}

// PilotDeckConfig holds server connection details.
type PilotDeckConfig struct {
	BaseURL          string `yaml:"base_url"`
	APIKey           string `yaml:"api_key"`
	ProjectPath      string `yaml:"project_path"`
	RetryAttempts    int    `yaml:"retry_attempts"`
	RetryIntervalSec int    `yaml:"retry_interval_sec"`
}

// taskYAML is the subset written to per-task task.yaml (no PilotDeck settings).
type taskYAML struct {
	Debug     bool   `yaml:"debug"`
	SessionID string `yaml:"session_id,omitempty"`
	Mode      string `yaml:"mode,omitempty"`
	Label     string `yaml:"label,omitempty"`
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
				cfg.PilotDeck.ProjectPath = filepath.Join(cwd, "working")
			}
		}
	} else if cwd, err := os.Getwd(); err == nil && cfg.PilotDeck.ProjectPath == cwd {
		// Correct bare cwd (old default) to <cwd>/working
		cfg.PilotDeck.ProjectPath = filepath.Join(cwd, "working")
	}
	// Ensure working directory exists
	if cfg.PilotDeck.ProjectPath != "" {
		_ = os.MkdirAll(cfg.PilotDeck.ProjectPath, 0o755)
	}
	if cfg.PilotDeck.RetryAttempts <= 0 {
		cfg.PilotDeck.RetryAttempts = 3
	}
	if cfg.PilotDeck.RetryIntervalSec <= 0 {
		cfg.PilotDeck.RetryIntervalSec = 20
	} else if cfg.PilotDeck.RetryIntervalSec < 20 {
		cfg.PilotDeck.RetryIntervalSec = 20
	}
	if cfg.Mode == "" {
		cfg.Mode = "continuous"
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
			cfg.Mode = task.Mode
			cfg.Label = task.Label
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
		Mode:      cfg.Mode,
		Label:     cfg.Label,
	}
	raw, err := yaml.Marshal(&task)
	if err != nil {
		return fmt.Errorf("marshal task config: %w", err)
	}
	if err := os.WriteFile(ConfigPath(taskDir), raw, 0o644); err != nil {
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
	if err := os.WriteFile(GlobalConfigPath(), raw, 0o644); err != nil {
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

// LoadGlobalConfig reads only the global config.yaml (PilotDeck settings + top-level fields).
func LoadGlobalConfig() (*TaskConfig, error) {
	path := GlobalConfigPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg TaskConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// pilotDeckHome returns the path to the PilotDeck configuration directory.
func pilotDeckHome() string {
	if h := os.Getenv("PILOT_HOME"); h != "" {
		return h
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".pilotdeck")
	}
	return ""
}

// readPilotDeckConfig reads PilotDeck's config files and returns a TaskConfig
// with auto-discovered base_url and api_key. Returns nil if PilotDeck is not found.
func readPilotDeckConfig() *TaskConfig {
	pdHome := pilotDeckHome()
	if pdHome == "" {
		return nil
	}

	pdYAML := filepath.Join(pdHome, "pilotdeck.yaml")
	raw, err := os.ReadFile(pdYAML)
	if err != nil {
		return nil
	}

	var pdCfg struct {
		WebUI struct {
			Runtime struct {
				ServerPort int `yaml:"serverPort"`
			} `yaml:"runtime"`
		} `yaml:"webui"`
	}
	if err := yaml.Unmarshal(raw, &pdCfg); err != nil {
		return nil
	}

	cfg := &TaskConfig{}
	// Set project_path to <cwd>/working
	if cwd, err := os.Getwd(); err == nil {
		cfg.PilotDeck.ProjectPath = filepath.Join(cwd, "working")
		_ = os.MkdirAll(cfg.PilotDeck.ProjectPath, 0o755)
	}

	// Read API key from auth.db (SQLite) — NOT from server-token (gateway WebSocket auth)
	dbPath := filepath.Join(pdHome, "auth.db")
	if db, err := sql.Open("sqlite", dbPath); err == nil {
		var apiKey string
		if err := db.QueryRow("SELECT api_key FROM api_keys WHERE is_active = 1 ORDER BY last_used DESC LIMIT 1").Scan(&apiKey); err == nil {
			cfg.PilotDeck.APIKey = apiKey
		}
		db.Close()
	}
	return cfg
}

// InitGlobalConfig discovers PilotDeck settings and populates/refreshes config.yaml.
// Always runs at startup to ensure latest API key from auth.db is used.
func InitGlobalConfig() error {
	pdCfg := readPilotDeckConfig()
	if pdCfg == nil {
		if !HasGlobalConfig() {
			fmt.Println("未检测到 PilotDeck 配置，跳过自动初始化。")
		}
		return nil
	}

	// Merge with existing config: preserve user-set project_path, but replace old bare-cwd default
	if raw, err := os.ReadFile(GlobalConfigPath()); err == nil {
		var existing TaskConfig
		if err := yaml.Unmarshal(raw, &existing); err == nil {
			if existing.PilotDeck.ProjectPath != "" {
				cwd, _ := os.Getwd()
				if cwd == "" || existing.PilotDeck.ProjectPath != cwd {
					pdCfg.PilotDeck.ProjectPath = existing.PilotDeck.ProjectPath
				}
			}
		}
	}

	fmt.Printf("检测到 PilotDeck 配置 (端口: %s)，正在自动生成 %s...\n",
		pdCfg.PilotDeck.BaseURL, GlobalConfigFileName)
	return SaveGlobalConfig(pdCfg)
}

// RefreshGlobalConfigFromPilotDeck re-reads PilotDeck config files and updates
// the global config.yaml with fresh values. Unlike InitGlobalConfig, it always runs.
func RefreshGlobalConfigFromPilotDeck() error {
	pdCfg := readPilotDeckConfig()
	if pdCfg == nil {
		return fmt.Errorf("未找到 PilotDeck 配置")
	}

	// Merge with existing config: preserve user-set project_path, but replace old bare-cwd default
	if raw, err := os.ReadFile(GlobalConfigPath()); err == nil {
		var existing TaskConfig
		if err := yaml.Unmarshal(raw, &existing); err == nil {
			if existing.PilotDeck.ProjectPath != "" {
				cwd, _ := os.Getwd()
				if cwd == "" || existing.PilotDeck.ProjectPath != cwd {
					pdCfg.PilotDeck.ProjectPath = existing.PilotDeck.ProjectPath
				}
			}
		}
	}

	fmt.Printf("正在从 PilotDeck 刷新连接配置 (base_url: %s)...\n", pdCfg.PilotDeck.BaseURL)
	return SaveGlobalConfig(pdCfg)
}
