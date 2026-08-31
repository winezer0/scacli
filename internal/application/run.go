package application

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"scacli/internal/git"
	"scacli/internal/report"
	"scacli/internal/sca"
	"scacli/internal/sca/murphysec"
	"scacli/internal/sca/opensca"
	"scacli/internal/sca/osvscanner"
	"scacli/internal/settings"
	"scacli/internal/tasks"
)

// Run executes the configured engine for every selected repository.
func Run(opts Options, cfg settings.Config, taskConfig tasks.Config) error {
	repositories, err := loadRepositories(opts, taskConfig.Tasks)
	if err != nil {
		return err
	}
	if len(repositories) == 0 {
		return errors.New("provide at least one --git or --git-file")
	}
	if cfg.Git.CloneTimeout <= 0 || cfg.Scan.ScanTimeout <= 0 {
		return errors.New("timeouts must be positive")
	}
	if err := os.MkdirAll(cfg.Report.OutputDir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if cfg.Scan.Token == "" && cfg.Scan.Engine != "osv-scanner" {
		return errors.New("SCA token is required: use --token or SCACLI_TOKEN")
	}
	formats, err := report.ResolveFormats(opts.Formats, cfg.Scan.Engine)
	if err != nil {
		return err
	}
	engine, err := createEngine(cfg.Scan.Engine, cfg.Scan.Murphysec, cfg.Scan.OpenSCA, cfg.Scan.OSVScanner)
	if err != nil {
		return err
	}
	client := git.NewClient(git.Config{SSHKey: cfg.Git.SSHKey, User: cfg.Git.GitUser, Password: cfg.Git.GitPassword, CloneTimeout: time.Duration(cfg.Git.CloneTimeout) * time.Second, RuntimeRoot: runtimeDirectory()})
	failed := 0
	for index, repository := range repositories {
		fmt.Printf("[%d/%d] %s\n", index+1, len(repositories), repository.URL)
		value := scanOne(client, engine, cfg.Scan.Engine, cfg.Scan.Token, repository, formats, cfg.Scan.ScanTimeout)
		if err := report.WriteReports(cfg.Report.OutputDir, value, formats); err != nil {
			return err
		}
		fmt.Printf("  status=%s components=%d vulnerabilities=%d critical=%d high=%d\n", value.Status, value.Summary.Components, value.Summary.Vulnerabilities, value.Summary.Critical, value.Summary.High)
		if value.Error != "" {
			fmt.Println("  error:", value.Error)
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d repositories failed; reports were generated", failed, len(repositories))
	}
	return nil
}

func createEngine(name, murphyPath, openSCAPath, osvPath string) (sca.Engine, error) {
	switch name {
	case "murphysec":
		path, err := resolveExecutable("murphysec", murphyPath)
		return murphysec.New(path), err
	case "opensca":
		path, err := resolveExecutable("opensca-cli", openSCAPath)
		return opensca.New(path), err
	case "osv-scanner":
		path, err := resolveExecutable("osv-scanner", osvPath)
		return osvscanner.New(path), err
	default:
		return nil, fmt.Errorf("unsupported SCA engine %q", name)
	}
}

func scanOne(client git.Client, engine sca.Engine, engineName, token string, repository git.Repository, formats []report.Format, timeout int) report.Report {
	started := time.Now()
	value := report.Report{Repository: repository, Engine: engineName, StartedAt: started, Status: "failed"}
	work, reused, err := client.Prepare(repository)
	value.Workspace = work
	if err == nil {
		if reused {
			err = client.Update(repository, work)
		} else {
			err = client.Clone(repository, work)
		}
	}
	if err == nil {
		request := sca.Request{Token: token, Workdir: work, Timeout: timeout, Formats: formatNames(formats)}
		result, scanErr := engine.Scan(request)
		err = scanErr
		if err == nil {
			value.Result = result.Result
			value.Native = result.Native
			value.Summary = report.Summarize(result.Result)
		}
	}
	if err != nil {
		value.Error = err.Error()
	} else {
		value.Status = "passed"
	}
	value.FinishedAt = time.Now()
	value.Duration = value.FinishedAt.Sub(started).Round(time.Millisecond).String()
	return value
}

func formatNames(formats []report.Format) []string {
	names := make([]string, 0, len(formats))
	for _, format := range formats {
		names = append(names, string(format))
	}
	return names
}

func loadRepositories(opts Options, configured []tasks.Task) ([]git.Repository, error) {
	var lines []string
	lines = append(lines, opts.GitURLs...)
	if opts.GitFile != "" {
		file, err := os.Open(opts.GitFile)
		if err != nil {
			return nil, fmt.Errorf("open git file: %w", err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("read git file: %w", err)
		}
	}
	if len(lines) == 0 {
		for _, item := range configured {
			lines = append(lines, strings.TrimSpace(item.URL)+" "+strings.TrimSpace(item.Branch))
		}
	}
	seen := map[string]bool{}
	var repositories []git.Repository
	for _, raw := range lines {
		fields := strings.Fields(strings.TrimSpace(raw))
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		if len(fields) > 2 {
			return nil, fmt.Errorf("invalid repository line: %q", raw)
		}
		branch := opts.Branch
		if len(fields) == 2 && fields[1] != "" {
			branch = fields[1]
		}
		key := fields[0] + "\x00" + branch
		if !seen[key] {
			seen[key] = true
			repositories = append(repositories, git.NewRepository(fields[0], branch))
		}
	}
	return repositories, nil
}

func runtimeDirectory() string {
	executable, err := os.Executable()
	if err != nil {
		return filepath.Join(".", "runtimes")
	}
	return filepath.Join(filepath.Dir(executable), "runtimes")
}
