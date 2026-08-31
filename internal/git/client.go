package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"scacli/internal/process"
)

// Config contains Git credentials and runtime workspace settings.
type Config struct {
	SSHKey       string
	User         string
	Password     string
	CloneTimeout time.Duration
	RuntimeRoot  string
}

// Client manages shallow clones and reusable runtime workspaces.
type Client struct {
	config Config
}

// NewClient creates a Git client.
func NewClient(config Config) Client {
	return Client{config: config}
}

// Prepare returns a stable workspace and whether it can be updated in place.
func (c Client) Prepare(repo Repository) (string, bool, error) {
	if err := os.MkdirAll(c.config.RuntimeRoot, 0755); err != nil {
		return "", false, fmt.Errorf("create runtime directory %s: %w", c.config.RuntimeRoot, err)
	}
	baseName := RuntimeName(repo)
	if existing, err := c.findExisting(baseName, repo.URL); err != nil {
		return "", false, err
	} else if existing != "" {
		return existing, true, nil
	}
	for index := 1; index <= 1000; index++ {
		name := baseName
		if index > 1 {
			name = fmt.Sprintf("%s-%d", baseName, index)
		}
		candidate := filepath.Join(c.config.RuntimeRoot, name)
		if err := os.Mkdir(candidate, 0755); err == nil {
			return candidate, false, nil
		} else if !errors.Is(err, os.ErrExist) {
			return "", false, fmt.Errorf("inspect runtime directory %s: %w", candidate, err)
		}
	}
	return "", false, fmt.Errorf("too many existing runtime directories for %s", baseName)
}

// Clone fetches a shallow repository into target.
func (c Client) Clone(repo Repository, target string) error {
	args := []string{"clone", "--quiet", "--depth", "1"}
	if repo.Branch != "" {
		args = append(args, "--branch", repo.Branch)
	}
	args = append(args, repo.URL, target)
	stdout, stderr, err := c.run("", args)
	if err != nil {
		return fmt.Errorf("git clone failed: %w: %s", err, process.DiagnosticOutput(stdout, stderr))
	}
	return nil
}

// Update fetches the requested ref and resets the reusable workspace.
func (c Client) Update(repo Repository, work string) error {
	args := []string{"fetch", "--quiet", "--depth", "1", "origin"}
	if repo.Branch != "" {
		args = append(args, repo.Branch)
	}
	if _, _, err := c.run(work, args); err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}
	if _, _, err := c.run(work, []string{"reset", "--hard", "FETCH_HEAD"}); err != nil {
		return fmt.Errorf("git reset failed: %w", err)
	}
	return nil
}

// RuntimeName returns the stable workspace name for a repository and ref.
func RuntimeName(repo Repository) string {
	ref := repo.Branch
	if ref == "" {
		ref = "default"
	}
	return SafeName(repo.Name) + "@" + SafeName(ref)
}

func (c Client) findExisting(baseName, remoteURL string) (string, error) {
	entries, err := os.ReadDir(c.config.RuntimeRoot)
	if err != nil {
		return "", fmt.Errorf("read runtime directory %s: %w", c.config.RuntimeRoot, err)
	}
	type candidate struct {
		path  string
		order int
	}
	var candidates []candidate
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		order, ok := runtimeSuffix(entry.Name(), baseName)
		if ok {
			candidates = append(candidates, candidate{filepath.Join(c.config.RuntimeRoot, entry.Name()), order})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].order < candidates[j].order })
	for _, item := range candidates {
		actual, _, err := c.gitCommand(item.path, []string{"remote", "get-url", "origin"})
		if err != nil {
			continue
		}
		if normalizeURL(actual) == normalizeURL(remoteURL) {
			return item.path, nil
		}
	}
	return "", nil
}

func runtimeSuffix(name, baseName string) (int, bool) {
	if name == baseName {
		return 1, true
	}
	prefix := baseName + "-"
	if !strings.HasPrefix(name, prefix) {
		return 0, false
	}
	value, err := strconv.Atoi(strings.TrimPrefix(name, prefix))
	return value, err == nil && value >= 2
}

func normalizeURL(value string) string {
	return strings.TrimSuffix(strings.TrimSpace(strings.TrimSuffix(value, "/")), ".git")
}

func (c Client) run(dir string, args []string) (string, string, error) {
	return c.gitCommand(dir, args)
}

func (c Client) gitCommand(dir string, args []string) (string, string, error) {
	gitArgs := append([]string{"-c", "http.version=HTTP/1.1"}, args...)
	cmd := exec.Command("git", gitArgs...)
	cmd.Dir = dir
	cmd.Env = gitEnvironment(c.config)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := process.RunWithTimeout(cmd, c.config.CloneTimeout)
	if err != nil {
		err = fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, process.DiagnosticOutput(stdout.String(), stderr.String()))
	}
	return stdout.String(), stderr.String(), err
}

func gitEnvironment(config Config) []string {
	env := os.Environ()
	if config.SSHKey != "" {
		env = append(env, "GIT_SSH_COMMAND=ssh -i "+shellQuote(config.SSHKey)+" -o IdentitiesOnly=yes")
	}
	if config.Password != "" || os.Getenv("GIT_PASSWORD") != "" {
		if config.Password != "" {
			env = append(env, "GIT_PASSWORD="+config.Password)
		}
		if config.User != "" {
			env = append(env, "GIT_USERNAME="+config.User)
		}
		env = append(env, "GIT_ASKPASS="+AskpassPath(), "GIT_TERMINAL_PROMPT=0")
		env = append(env, "SCACLI_ASKPASS=1")
	}
	return env
}

func shellQuote(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}
