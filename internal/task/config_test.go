package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModeOrContinuous_Default(t *testing.T) {
	cfg := &TaskConfig{}
	if m := cfg.ModeOrContinuous(); m != "continuous" {
		t.Errorf("expected continuous, got %q", m)
	}
}

func TestModeOrContinuous_Discrete(t *testing.T) {
	cfg := &TaskConfig{Mode: "discrete"}
	if m := cfg.ModeOrContinuous(); m != "discrete" {
		t.Errorf("expected discrete, got %q", m)
	}
}

func TestModeOrContinuous_Continuous(t *testing.T) {
	cfg := &TaskConfig{Mode: "continuous"}
	if m := cfg.ModeOrContinuous(); m != "continuous" {
		t.Errorf("expected continuous, got %q", m)
	}
}

func TestSaveLoadConfig_Mode(t *testing.T) {
	AppDir = t.TempDir()

	dir := t.TempDir()
	cfg := &TaskConfig{Debug: true, Mode: "discrete"}

	if err := SaveConfig(dir, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.Mode != "discrete" {
		t.Errorf("expected mode discrete, got %q", loaded.Mode)
	}
	if !loaded.Debug {
		t.Error("expected debug=true")
	}
}

func TestSaveLoadConfig_NoMode(t *testing.T) {
	AppDir = t.TempDir()

	dir := t.TempDir()
	cfg := &TaskConfig{Debug: false}

	if err := SaveConfig(dir, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.Mode != "" {
		t.Errorf("expected empty mode (defaulting to continuous), got %q", loaded.Mode)
	}
}

func TestSaveConfig_NoSessionID(t *testing.T) {
	AppDir = t.TempDir()

	dir := t.TempDir()
	cfg := &TaskConfig{Mode: "discrete"}

	if err := SaveConfig(dir, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, configFileName))
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(raw), "session_id") {
		t.Error("session_id should not appear in task.yaml")
	}
}
