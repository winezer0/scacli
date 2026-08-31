package git

import (
	"fmt"
	"os"
	"strings"
)

// Askpass prints the configured credential for Git's prompt.
func Askpass(args []string) {
	prompt := strings.ToLower(strings.Join(args, " "))
	if strings.Contains(prompt, "username") || strings.Contains(prompt, "user") {
		fmt.Print(os.Getenv("GIT_USERNAME"))
		return
	}
	fmt.Print(os.Getenv("GIT_PASSWORD"))
}

// AskpassPath returns the current executable path for Git's askpass callback.
func AskpassPath() string {
	path, _ := os.Executable()
	return path
}
