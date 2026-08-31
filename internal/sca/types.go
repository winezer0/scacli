package sca

import "encoding/json"

// Component describes a dependency detected by an SCA engine.
type Component struct {
	Name     string          `json:"name"`
	Version  string          `json:"version"`
	Language string          `json:"language,omitempty"`
	Path     []string        `json:"dependency_path,omitempty"`
	Vulns    []Vulnerability `json:"vulnerabilities,omitempty"`
}

// Vulnerability describes a vulnerability associated with a component.
type Vulnerability struct {
	ID       string `json:"cve_id,omitempty"`
	Title    string `json:"title"`
	Severity string `json:"severity,omitempty"`
	Fixed    string `json:"fixed_version,omitempty"`
	Detail   string `json:"description,omitempty"`
}

// Result is the normalized result shared by report writers and the CLI.
type Result struct {
	Components []Component `json:"components,omitempty"`
}

// NativeReports stores complete native engine outputs independently.
type NativeReports struct {
	JSON     json.RawMessage `json:"native_json,omitempty"`
	HTML     []byte          `json:"-"`
	Markdown []byte          `json:"-"`
}

// EngineResult contains normalized data and native engine outputs.
type EngineResult struct {
	Result Result        `json:"result"`
	Native NativeReports `json:"native"`
}

// Request contains the common input for an engine scan.
type Request struct {
	Token   string
	Workdir string
	Timeout int
	Formats []string
}

// Engine scans one checked-out project.
type Engine interface {
	Scan(Request) (EngineResult, error)
}
