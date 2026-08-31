package report

import (
	"fmt"
	"strings"
)

// Format identifies a report file format.
type Format string

const (
	// FormatJSON is the machine-readable report format.
	FormatJSON Format = "json"
	// FormatHTML is the browser-readable report format.
	FormatHTML Format = "html"
	// FormatMarkdown is the plain-text report format.
	FormatMarkdown Format = "md"
)

// ResolveFormats parses requested formats and filters unsupported formats.
func ResolveFormats(values []string, engine string) ([]Format, error) {
	if len(values) == 0 {
		values = []string{"json,html,md"}
	}
	allowed := supportedFormats(engine)
	seen := map[Format]bool{}
	var formats []Format
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			format, err := parseFormat(item)
			if err != nil {
				return nil, err
			}
			if !allowed[format] || seen[format] {
				continue
			}
			seen[format] = true
			formats = append(formats, format)
		}
	}
	if len(formats) == 0 {
		return nil, fmt.Errorf("none of the requested report formats are supported by %s", engine)
	}
	return formats, nil
}

func parseFormat(value string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "json":
		return FormatJSON, nil
	case "html":
		return FormatHTML, nil
	case "md", "markdown":
		return FormatMarkdown, nil
	default:
		return "", fmt.Errorf("invalid report format %q: must be json, html, or md", value)
	}
}

func supportedFormats(engine string) map[Format]bool {
	switch engine {
	case "murphysec":
		return map[Format]bool{FormatJSON: true}
	case "opensca":
		return map[Format]bool{FormatHTML: true}
	case "osv-scanner":
		return map[Format]bool{FormatJSON: true, FormatHTML: true, FormatMarkdown: true}
	default:
		return map[Format]bool{}
	}
}

// HasFormat reports whether a format was requested.
func HasFormat(formats []Format, target Format) bool {
	for _, format := range formats {
		if format == target {
			return true
		}
	}
	return false
}
