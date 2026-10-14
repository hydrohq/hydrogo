package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// devCommand runs the tools/devserver-based development loop from a HydroGo
// checkout.
func devCommand(args []string, stdout, stderr io.Writer) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	if !isCoreCheckout(root) {
		return fmt.Errorf("hydro dev: %q is not a HydroGo checkout (go.mod + tools/devserver)", root)
	}
	return coreDevCommand(args, stdout, stderr)
}

// isCoreCheckout reports whether root is a HydroGo repository checkout,
// i.e. the source tree that owns tools/devserver.
func isCoreCheckout(root string) bool {
	if info, err := os.Stat(filepath.Join(root, "go.mod")); err != nil || info.IsDir() {
		return false
	}
	info, err := os.Stat(filepath.Join(root, "tools", "devserver"))
	return err == nil && info.IsDir()
}

func coreDevCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("dev", flag.ContinueOnError)
	flags.SetOutput(stderr)
	_ = flags.String("address", "0.0.0.0:8090", "development proxy listen address")
	_ = flags.String("app-address", "127.0.0.1:8091", "internal HydroGo application address")
	_ = flags.String("database-dsn", "", "SQLite/PostgreSQL DSN passed to the child server")
	_ = flags.String("redis-url", "", "Optional Redis URL passed to the child server")
	_ = flags.String("dependencies", "auto", "Local dependencies: auto, none, or required")
	_ = flags.String("compose-file", "deploy/local/docker-compose.storage.yml", "Docker Compose file for local dependencies")
	_ = flags.Duration("poll", 600*time.Millisecond, "file watch polling interval")
	_ = flags.Bool("seed-demo", true, "seed the disposable local demo data before starting")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	commandArgs := append([]string{"run", "./tools/devserver"}, args...)
	command := exec.Command("go", commandArgs...)
	command.Stdout, command.Stderr = stdout, stderr
	command.Dir, _ = os.Getwd()
	return command.Run()
}
