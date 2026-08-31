package opensca

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"scacli/internal/process"
	"scacli/internal/sca"
)

// Engine invokes the OpenSCA command-line scanner.
type Engine struct {
	executable string
}

// New creates an OpenSCA engine for the executable path.
func New(executable string) Engine {
	return Engine{executable: executable}
}

// Scan runs OpenSCA and returns its complete native HTML report.
func (e Engine) Scan(request sca.Request) (sca.EngineResult, error) {
	outputPath := filepath.Join(request.Workdir, "output.html")
	_ = os.Remove(outputPath)
	cmd := exec.Command(e.executable, "-token", request.Token, "-path", request.Workdir, "-out", "output.html")
	cmd.Dir = request.Workdir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := process.RunWithTimeout(cmd, time.Duration(request.Timeout)*time.Second); err != nil {
		return sca.EngineResult{}, fmt.Errorf("OpenSCA failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return sca.EngineResult{}, fmt.Errorf("read OpenSCA result %s: %w", outputPath, err)
	}
	return sca.EngineResult{Native: sca.NativeReports{HTML: data}}, nil
}
