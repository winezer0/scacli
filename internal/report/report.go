package report

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"scacli/internal/git"
	"scacli/internal/sca"
)

// Report is the independent report model written for one repository.
type Report struct {
	Repository git.Repository    `json:"repository"`
	Workspace  string            `json:"workspace,omitempty"`
	Engine     string            `json:"engine"`
	StartedAt  time.Time         `json:"started_at"`
	FinishedAt time.Time         `json:"finished_at"`
	Duration   string            `json:"duration"`
	Status     string            `json:"status"`
	Error      string            `json:"error,omitempty"`
	Summary    Summary           `json:"summary"`
	Result     sca.Result        `json:"result,omitempty"`
	Native     sca.NativeReports `json:"native,omitempty"`
}

// Summary contains the aggregate SCA counts shown in the CLI and reports.
type Summary struct {
	Components      int `json:"components"`
	Vulnerabilities int `json:"vulnerabilities"`
	Critical        int `json:"critical"`
	High            int `json:"high"`
	Medium          int `json:"medium"`
	Low             int `json:"low"`
}

// Summarize calculates aggregate counts for normalized scan data.
func Summarize(result sca.Result) Summary {
	summary := Summary{Components: len(result.Components)}
	for _, component := range result.Components {
		summary.Vulnerabilities += len(component.Vulns)
		for _, vuln := range component.Vulns {
			switch strings.ToLower(vuln.Severity) {
			case "critical":
				summary.Critical++
			case "high":
				summary.High++
			case "medium":
				summary.Medium++
			case "low":
				summary.Low++
			}
		}
	}
	return summary
}

// WriteReports writes every requested format as a complete independent file.
func WriteReports(dir string, value Report, formats []Format) error {
	base := reportBase(dir, value)
	for _, format := range formats {
		data, err := reportData(value, format)
		if err != nil {
			return err
		}
		if err := os.WriteFile(base+"."+string(format), data, 0644); err != nil {
			return fmt.Errorf("write %s report: %w", format, err)
		}
	}
	return nil
}

func reportData(value Report, format Format) ([]byte, error) {
	switch format {
	case FormatJSON:
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return nil, err
		}
		return append(data, '\n'), nil
	case FormatHTML:
		if len(value.Native.HTML) > 0 {
			return value.Native.HTML, nil
		}
		return fallbackHTML(value)
	case FormatMarkdown:
		if len(value.Native.Markdown) > 0 {
			return value.Native.Markdown, nil
		}
		return []byte(fallbackMarkdown(value)), nil
	default:
		return nil, fmt.Errorf("unsupported report format %q", format)
	}
}

func reportBase(dir string, value Report) string {
	stem := git.RuntimeName(value.Repository) + "-" + time.Now().Format("20060102_150405")
	base := filepath.Join(dir, stem)
	for index := 2; index <= 1000; index++ {
		if !reportExists(base) {
			return base
		}
		base = filepath.Join(dir, fmt.Sprintf("%s-%d", stem, index))
	}
	return base
}

func reportExists(base string) bool {
	for _, suffix := range []string{".html", ".json", ".md"} {
		if _, err := os.Stat(base + suffix); err == nil {
			return true
		}
	}
	return false
}

func fallbackMarkdown(value Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# SCA Report: %s\n\n- Repository: `%s`\n- Branch or tag: `%s`\n- Workspace: `%s`\n- Engine: `%s`\n- Status: **%s**\n- Duration: `%s`\n\n", value.Repository.Name, value.Repository.URL, value.Repository.Branch, value.Workspace, value.Engine, value.Status, value.Duration)
	if value.Error != "" {
		fmt.Fprintf(&b, "## Error\n\n%s\n\n", value.Error)
	}
	fmt.Fprintf(&b, "## Summary\n\n| Metric | Count |\n|---|---:|\n| Components | %d |\n| Vulnerabilities | %d |\n| Critical | %d |\n| High | %d |\n| Medium | %d |\n| Low | %d |\n\n", value.Summary.Components, value.Summary.Vulnerabilities, value.Summary.Critical, value.Summary.High, value.Summary.Medium, value.Summary.Low)
	if len(value.Result.Components) == 0 {
		return b.String()
	}
	b.WriteString("## Components and vulnerabilities\n\n| Component | Version | Language | Vulnerability | Severity | Fixed version | Details |\n|---|---|---|---|---|---|---|\n")
	for _, component := range value.Result.Components {
		if len(component.Vulns) == 0 {
			fmt.Fprintf(&b, "| %s | %s | %s | - | - | - | - |\n", cell(component.Name), cell(component.Version), cell(component.Language))
			continue
		}
		for _, vuln := range component.Vulns {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n", cell(component.Name), cell(component.Version), cell(component.Language), cell(first(vuln.ID, vuln.Title)), cell(vuln.Severity), cell(vuln.Fixed), cell(vuln.Detail))
		}
	}
	return b.String()
}

func fallbackHTML(value Report) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><title>SCA Report</title><style>body{font-family:Arial,sans-serif;margin:2rem}table{border-collapse:collapse}th,td{border:1px solid #ccc;padding:.4rem;text-align:left}pre{white-space:pre-wrap;background:#f6f8fa;padding:1rem}</style></head><body>")
	fmt.Fprintf(&b, "<h1>SCA Report: %s</h1><table><tr><th>Repository</th><td>%s</td></tr><tr><th>Branch or tag</th><td>%s</td></tr><tr><th>Workspace</th><td>%s</td></tr><tr><th>Engine</th><td>%s</td></tr><tr><th>Status</th><td>%s</td></tr><tr><th>Duration</th><td>%s</td></tr></table>", html.EscapeString(value.Repository.Name), html.EscapeString(value.Repository.URL), html.EscapeString(value.Repository.Branch), html.EscapeString(value.Workspace), html.EscapeString(value.Engine), html.EscapeString(value.Status), html.EscapeString(value.Duration))
	if value.Error != "" {
		fmt.Fprintf(&b, "<h2>Error</h2><pre>%s</pre>", html.EscapeString(value.Error))
	}
	fmt.Fprintf(&b, "<h2>Summary</h2><p>Components: %d; Vulnerabilities: %d; Critical: %d; High: %d; Medium: %d; Low: %d</p><h2>Complete report data</h2><pre>%s</pre></body></html>\n", value.Summary.Components, value.Summary.Vulnerabilities, value.Summary.Critical, value.Summary.High, value.Summary.Medium, value.Summary.Low, html.EscapeString(string(data)))
	return []byte(b.String()), nil
}

func first(left, right string) string {
	if left != "" {
		return left
	}
	return right
}

func cell(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "|", "\\|"), "\n", " ")
}
