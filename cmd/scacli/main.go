package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"scacli/config"
	"scacli/internal/application"
	"scacli/internal/git"
)

const version = "0.1.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--internal-askpass" {
		git.Askpass(os.Args[2:])
		return
	}
	if os.Getenv("SCACLI_ASKPASS") == "1" {
		git.Askpass(os.Args[1:])
		return
	}
	var opts application.Options
	parser := flags.NewParser(&opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		os.Exit(2)
	}
	if opts.Version {
		fmt.Println(version)
		return
	}
	appConfig, err := config.Load(config.LoadOptions{Path: opts.ConfigFile, TaskPath: opts.TaskFile, CLI: config.CLIOverrides{
		Engine: opts.Engine, Token: opts.Token, Murphysec: opts.Murphysec, OpenSCA: opts.OpenSCA, OSVScanner: opts.OSVScanner,
		OutputDir: opts.OutputDir, SSHKey: opts.SSHKey, GitUser: opts.GitUser, GitPassword: opts.GitPassword,
		CloneTimeout: opts.CloneTimeout, ScanTimeout: opts.ScanTimeout,
	}})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if err := application.Run(opts, appConfig.Config, appConfig.Tasks); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
