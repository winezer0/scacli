package osvscanner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"scacli/internal/process"
	"scacli/internal/sca"
)

// Engine invokes OSV-Scanner and preserves its native output formats.
type Engine struct {
	executable string
}

// New creates an OSV-Scanner engine for the executable path.
func New(executable string) Engine {
	return Engine{executable: executable}
}

// Scan runs JSON first and then generates the requested native formats.
func (e Engine) Scan(request sca.Request) (sca.EngineResult, error) {
	jsonPath := filepath.Join(request.Workdir, "osv-output.json")
	if err := e.run(request, "json", jsonPath); err != nil {
		return sca.EngineResult{}, err
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return sca.EngineResult{}, fmt.Errorf("read OSV-Scanner result %s: %w", jsonPath, err)
	}
	var native nativeResult
	if err := json.Unmarshal(data, &native); err != nil {
		return sca.EngineResult{}, fmt.Errorf("decode OSV-Scanner result: %w", err)
	}
	result := native.normalized()
	output := sca.EngineResult{Result: result, Native: sca.NativeReports{JSON: append(json.RawMessage(nil), data...)}}
	if hasFormat(request.Formats, "html") {
		path := filepath.Join(request.Workdir, "osv-output.html")
		if err := e.run(request, "html", path); err != nil {
			return sca.EngineResult{}, err
		}
		output.Native.HTML, err = os.ReadFile(path)
		if err != nil {
			return sca.EngineResult{}, fmt.Errorf("read OSV-Scanner HTML report: %w", err)
		}
	}
	if hasFormat(request.Formats, "md") {
		path := filepath.Join(request.Workdir, "osv-output.md")
		if err := e.run(request, "markdown", path); err != nil {
			return sca.EngineResult{}, err
		}
		output.Native.Markdown, err = os.ReadFile(path)
		if err != nil {
			return sca.EngineResult{}, fmt.Errorf("read OSV-Scanner Markdown report: %w", err)
		}
	}
	return output, nil
}

func hasFormat(formats []string, target string) bool {
	for _, format := range formats {
		if format == target {
			return true
		}
	}
	return false
}

func (e Engine) run(request sca.Request, format, outputPath string) error {
	_ = os.Remove(outputPath)
	cmd := exec.Command(e.executable, "scan", "source", "-r", ".", "--format", format, "--output-file", filepath.Base(outputPath))
	cmd.Dir = request.Workdir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := process.RunWithTimeout(cmd, time.Duration(request.Timeout)*time.Second)
	if err != nil && !vulnerabilitiesFound(err) {
		return fmt.Errorf("OSV-Scanner %s failed: %w: %s", format, err, process.DiagnosticOutput(stdout.String(), stderr.String()))
	}
	return nil
}

func vulnerabilitiesFound(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitCode() == 1
}

type nativeResult struct {
	Results []resultGroup `json:"results,omitempty"`
}

type resultGroup struct {
	Source   source          `json:"source"`
	Packages []packageResult `json:"packages,omitempty"`
}

type source struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type packageResult struct {
	Package         packageInfo     `json:"package"`
	Vulnerabilities []vulnerability `json:"vulnerabilities,omitempty"`
}

type packageInfo struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Ecosystem string `json:"ecosystem"`
}

type vulnerability struct {
	ID               string   `json:"id"`
	Aliases          []string `json:"aliases,omitempty"`
	Summary          string   `json:"summary,omitempty"`
	Details          string   `json:"details,omitempty"`
	DatabaseSpecific struct {
		Severity string `json:"severity,omitempty"`
	} `json:"database_specific,omitempty"`
}

func (result nativeResult) normalized() sca.Result {
	var output sca.Result
	for _, group := range result.Results {
		for _, item := range group.Packages {
			component := sca.Component{Name: item.Package.Name, Version: item.Package.Version, Language: item.Package.Ecosystem, Path: []string{group.Source.Path}}
			for _, vuln := range item.Vulnerabilities {
				id := vuln.ID
				if len(vuln.Aliases) > 0 && vuln.Aliases[0] != "" {
					id = vuln.Aliases[0]
				}
				component.Vulns = append(component.Vulns, sca.Vulnerability{ID: id, Title: vuln.Summary, Severity: strings.ToLower(vuln.DatabaseSpecific.Severity), Detail: vuln.Details})
			}
			output.Components = append(output.Components, component)
		}
	}
	return output
}
