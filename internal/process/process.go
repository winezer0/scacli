package process

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RunWithTimeout starts a command and returns its exit or timeout error.
func RunWithTimeout(cmd *exec.Cmd, timeout time.Duration) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		<-done
		return fmt.Errorf("timed out after %s", timeout)
	}
}

// DiagnosticOutput combines command standard output and error output.
func DiagnosticOutput(stdout, stderr string) string {
	parts := make([]string, 0, 2)
	if value := strings.TrimSpace(stderr); value != "" {
		parts = append(parts, "stderr: "+value)
	}
	if value := strings.TrimSpace(stdout); value != "" {
		parts = append(parts, "stdout: "+value)
	}
	if len(parts) == 0 {
		return "no diagnostic output"
	}
	return strings.Join(parts, " | ")
}
