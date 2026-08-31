package murphysec

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

// Engine invokes the MurphySec command-line scanner.
type Engine struct {
	executable string
}

// New creates a MurphySec engine for the executable path.
func New(executable string) Engine {
	return Engine{executable: executable}
}

// Scan runs MurphySec with its native JSON output.
func (e Engine) Scan(request sca.Request) (sca.EngineResult, error) {
	cmd := exec.Command(e.executable, "--token", request.Token, "scan", filepath.Base(request.Workdir), "--json")
	cmd.Dir = filepath.Dir(request.Workdir)
	cmd.Env = append(os.Environ(), "SCACLI_TOKEN=")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := process.RunWithTimeout(cmd, time.Duration(request.Timeout)*time.Second); err != nil {
		return sca.EngineResult{}, fmt.Errorf("MurphySec failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	raw := stdout.Bytes()
	var result sca.Result
	if err := decode(raw, &result); err != nil {
		return sca.EngineResult{}, err
	}
	return sca.EngineResult{Result: result, Native: sca.NativeReports{JSON: append(json.RawMessage(nil), raw...)}}, nil
}

func decode(raw []byte, result *sca.Result) error {
	if err := json.Unmarshal(raw, result); err == nil {
		return nil
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || len(envelope.Data) == 0 {
		return errors.New("invalid MurphySec JSON output")
	}
	if err := json.Unmarshal(envelope.Data, result); err != nil {
		return fmt.Errorf("decode MurphySec result: %w", err)
	}
	return nil
}
