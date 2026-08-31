package application

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func resolveExecutable(name, explicit string) (string, error) {
	searchName := name
	if explicit != "" {
		searchName = explicit
		if isPathReference(explicit) {
			if info, err := os.Stat(explicit); err == nil && !info.IsDir() {
				return explicit, nil
			}
			return "", fmt.Errorf("%s executable not found at %s", name, explicit)
		}
	}
	if path, err := exec.LookPath(searchName); err == nil {
		return path, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve %s: locate executable failed: %w", name, err)
	}
	appDir := filepath.Dir(executable)
	for _, dir := range []string{appDir, filepath.Join(appDir, "bin")} {
		for _, candidate := range executableCandidates(dir, searchName) {
			if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("%s executable not found; searched PATH, %s, and %s", searchName, appDir, filepath.Join(appDir, "bin"))
}

func isPathReference(value string) bool {
	return filepath.IsAbs(value) || strings.ContainsAny(value, `/\`)
}

func executableCandidates(dir, name string) []string {
	paths := []string{filepath.Join(dir, name)}
	if filepath.Ext(name) == "" {
		paths = append(paths, filepath.Join(dir, name+".exe"))
	}
	return paths
}
