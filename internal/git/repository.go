package git

import (
	"strings"
	"unicode"
)

// Repository identifies a Git repository and its requested ref.
type Repository struct {
	URL    string `json:"url"`
	Branch string `json:"branch,omitempty"`
	Name   string `json:"name"`
}

// NewRepository creates a repository with a safe display and runtime name.
func NewRepository(rawURL, branch string) Repository {
	return Repository{URL: rawURL, Branch: branch, Name: repositoryName(rawURL)}
}

func repositoryName(raw string) string {
	name := strings.TrimSpace(strings.TrimRight(raw, "/\\"))
	if scheme := strings.Index(name, "://"); scheme >= 0 {
		name = name[scheme+3:]
		if slash := strings.IndexAny(name, "/\\"); slash >= 0 {
			name = name[slash+1:]
		}
	} else if colon := strings.Index(name, ":"); colon > 0 && !(len(name) == 2 && name[1] == ':') && !strings.ContainsAny(name[:colon], "/\\") {
		name = name[colon+1:]
	}
	name = strings.TrimSuffix(name, ".git")
	if name == "." || name == "" {
		name = "repository"
	}
	return SafeName(name)
}

// SafeName replaces characters that cannot safely appear in a runtime name.
func SafeName(value string) string {
	var b strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "repository"
	}
	return b.String()
}
