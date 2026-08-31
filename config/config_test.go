package config

import (
	"os"
	"path/filepath"
	"testing"

	"scacli/internal/tasks"
)

func TestLoadDefaults(t *testing.T) {
	clearTokenEnvironment(t)
	cfg, err := Load(LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Scan.Engine != "murphysec" || cfg.Git.CloneTimeout != 300 || cfg.Scan.ScanTimeout != 600 || cfg.Report.OutputDir != "reports" {
		t.Fatalf("unexpected defaults: %+v", cfg.Config)
	}
}

func TestLoadValidConfigAndCLIOverride(t *testing.T) {
	clearTokenEnvironment(t)
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte("scan:\n  engine: opensca\n  token: file-token\n  scan_timeout: 120\ngit:\n  clone_timeout: 60\nreport:\n  output: out\n")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(LoadOptions{Path: path, CLI: CLIOverrides{Token: "cli-token", OutputDir: "cli-out"}})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Scan.Engine != "opensca" || cfg.Scan.Token != "cli-token" || cfg.Report.OutputDir != "cli-out" {
		t.Fatalf("config merge failed: %+v", cfg.Config)
	}
}

func TestLoadTaskFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	taskPath := filepath.Join(t.TempDir(), "task.yaml")
	if err := os.WriteFile(configPath, []byte("scan:\n  engine: osv-scanner\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(taskPath, []byte("tasks:\n  - url: git@example.com:team/app.git\n    branch: develop\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(LoadOptions{Path: configPath, TaskPath: taskPath})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TaskPath != taskPath || len(cfg.Tasks.Tasks) != 1 || cfg.Tasks.Tasks[0].URL != "git@example.com:team/app.git" || cfg.Tasks.Tasks[0].Branch != "develop" {
		t.Fatalf("task config = %+v", cfg.Tasks)
	}
}

func TestValidateTasksRejectsMissingURL(t *testing.T) {
	if err := ValidateTasks(tasks.Config{Tasks: []tasks.Task{{Branch: "main"}}}); err == nil {
		t.Fatal("expected task validation error")
	}
}

func TestEnvironmentTokenOverridesFile(t *testing.T) {
	t.Setenv("MURPHYSEC_TOKEN", "")
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("scan:\n  token: file-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SCACLI_TOKEN", "env-token")
	cfg, err := Load(LoadOptions{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Scan.Token != "env-token" {
		t.Fatalf("token = %q", cfg.Scan.Token)
	}
}

func TestEngineSpecificTokenOverridesGenericToken(t *testing.T) {
	t.Setenv("SCACLI_TOKEN", "generic-token")
	t.Setenv("MURPHYSEC_TOKEN", "murphy-token")
	t.Setenv("OPENSCA_TOKEN", "opensca-token")

	murphy, err := Load(LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if murphy.Scan.Token != "murphy-token" {
		t.Fatalf("MurphySec token = %q", murphy.Scan.Token)
	}

	openSCA, err := Load(LoadOptions{CLI: CLIOverrides{Engine: "opensca"}})
	if err != nil {
		t.Fatal(err)
	}
	if openSCA.Scan.Token != "opensca-token" {
		t.Fatalf("OpenSCA token = %q", openSCA.Scan.Token)
	}
}

func TestCLITokenOverridesEnvironment(t *testing.T) {
	t.Setenv("SCACLI_TOKEN", "generic-token")
	t.Setenv("MURPHYSEC_TOKEN", "murphy-token")
	cfg, err := Load(LoadOptions{CLI: CLIOverrides{Token: "cli-token"}})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Scan.Token != "cli-token" {
		t.Fatalf("token = %q", cfg.Scan.Token)
	}
}

func clearTokenEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("SCACLI_TOKEN", "")
	t.Setenv("MURPHYSEC_TOKEN", "")
	t.Setenv("OPENSCA_TOKEN", "")
}

func TestValidateRejectsInvalidConfig(t *testing.T) {
	cfg, err := Load(LoadOptions{CLI: CLIOverrides{Engine: "invalid"}})
	if err == nil {
		t.Fatal("expected validation error")
	}
	_ = cfg
}

func TestLoadOSVScannerConfig(t *testing.T) {
	cfg, err := Load(LoadOptions{CLI: CLIOverrides{Engine: "osv-scanner", OSVScanner: "osv-scanner.exe"}})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Scan.Engine != "osv-scanner" || cfg.Scan.OSVScanner != "osv-scanner.exe" {
		t.Fatalf("OSV-Scanner config = %+v", cfg.Scan)
	}
}
