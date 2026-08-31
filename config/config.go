package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"scacli/internal/settings"
	"scacli/internal/tasks"
)

// CLIOverrides contains only values explicitly supplied by the CLI.
type CLIOverrides struct {
	Engine       string
	Token        string
	Murphysec    string
	OpenSCA      string
	OSVScanner   string
	OutputDir    string
	SSHKey       string
	GitUser      string
	GitPassword  string
	CloneTimeout int
	ScanTimeout  int
}

type LoadOptions struct {
	Path     string
	TaskPath string
	CLI      CLIOverrides
}

// AppConfig is the public configuration facade returned to external entrypoints.
type AppConfig struct {
	settings.Config
	Tasks    tasks.Config
	Path     string
	TaskPath string
}

func Load(opts LoadOptions) (AppConfig, error) {
	cfg := settings.Config{
		Scan:   settings.ScanConfig{Engine: "murphysec", ScanTimeout: 600},
		Git:    settings.GitConfig{CloneTimeout: 300},
		Report: settings.ReportConfig{OutputDir: "reports"},
	}

	path, err := findConfigPath(opts.Path)
	if err != nil {
		return AppConfig{}, err
	}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return AppConfig{}, fmt.Errorf("read config file %s: %w", path, err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return AppConfig{}, fmt.Errorf("parse config file %s: %w", path, err)
		}
	}
	taskPath, err := findTaskPath(opts.TaskPath)
	if err != nil {
		return AppConfig{}, err
	}
	var taskConfig tasks.Config
	if taskPath != "" {
		data, err := os.ReadFile(taskPath)
		if err != nil {
			return AppConfig{}, fmt.Errorf("read task file %s: %w", taskPath, err)
		}
		if err := yaml.Unmarshal(data, &taskConfig); err != nil {
			return AppConfig{}, fmt.Errorf("parse task file %s: %w", taskPath, err)
		}
	}

	applyCLI(&cfg, opts.CLI)
	if opts.CLI.Token == "" {
		if token := environmentToken(cfg.Scan.Engine); token != "" {
			cfg.Scan.Token = token
		}
	}
	if err := Validate(cfg); err != nil {
		return AppConfig{}, err
	}
	if err := ValidateTasks(taskConfig); err != nil {
		return AppConfig{}, err
	}
	return AppConfig{Config: cfg, Tasks: taskConfig, Path: path, TaskPath: taskPath}, nil
}

func environmentToken(engine string) string {
	if name := engineTokenVariable(engine); name != "" {
		if token := os.Getenv(name); token != "" {
			return token
		}
	}
	return os.Getenv("SCACLI_TOKEN")
}

func engineTokenVariable(engine string) string {
	switch engine {
	case "murphysec":
		return "MURPHYSEC_TOKEN"
	case "opensca":
		return "OPENSCA_TOKEN"
	default:
		return ""
	}
}

func Validate(cfg settings.Config) error {
	if cfg.Scan.Engine != "murphysec" && cfg.Scan.Engine != "opensca" && cfg.Scan.Engine != "osv-scanner" {
		return fmt.Errorf("invalid scan.engine %q: must be murphysec, opensca, or osv-scanner", cfg.Scan.Engine)
	}
	if cfg.Git.CloneTimeout <= 0 {
		return errors.New("git.clone_timeout must be positive")
	}
	if cfg.Scan.ScanTimeout <= 0 {
		return errors.New("scan.scan_timeout must be positive")
	}
	if cfg.Report.OutputDir == "" {
		return errors.New("report.output cannot be empty")
	}
	return nil
}

// ValidateTasks validates task entries loaded from task.yaml.
func ValidateTasks(cfg tasks.Config) error {
	for index, task := range cfg.Tasks {
		if task.URL == "" {
			return fmt.Errorf("tasks[%d].url cannot be empty", index)
		}
	}
	return nil
}

func applyCLI(cfg *settings.Config, cli CLIOverrides) {
	if cli.Engine != "" {
		cfg.Scan.Engine = cli.Engine
	}
	if cli.Token != "" {
		cfg.Scan.Token = cli.Token
	}
	if cli.Murphysec != "" {
		cfg.Scan.Murphysec = cli.Murphysec
	}
	if cli.OpenSCA != "" {
		cfg.Scan.OpenSCA = cli.OpenSCA
	}
	if cli.OSVScanner != "" {
		cfg.Scan.OSVScanner = cli.OSVScanner
	}
	if cli.OutputDir != "" {
		cfg.Report.OutputDir = cli.OutputDir
	}
	if cli.SSHKey != "" {
		cfg.Git.SSHKey = cli.SSHKey
	}
	if cli.GitUser != "" {
		cfg.Git.GitUser = cli.GitUser
	}
	if cli.GitPassword != "" {
		cfg.Git.GitPassword = cli.GitPassword
	}
	if cli.CloneTimeout != 0 {
		cfg.Git.CloneTimeout = cli.CloneTimeout
	}
	if cli.ScanTimeout != 0 {
		cfg.Scan.ScanTimeout = cli.ScanTimeout
	}
}

func findConfigPath(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("config file %s: %w", explicit, err)
		}
		return explicit, nil
	}
	candidates := []string{"config.yaml"}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "config.yaml"))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", nil
}

func findTaskPath(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("task file %s: %w", explicit, err)
		}
		return explicit, nil
	}
	candidates := []string{"task.yaml"}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "task.yaml"))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", nil
}
