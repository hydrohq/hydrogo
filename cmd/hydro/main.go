package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/ardaozyildirim/hydrogo/internal/server"
)

const version = "0.0.0-dev"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "hydro:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stderr)
		return errors.New("a command is required")
	}

	switch args[0] {
	case "version":
		if len(args) != 1 {
			return errors.New("version does not accept arguments")
		}
		_, err := fmt.Fprintln(stdout, version)
		return err
	case "serve":
		return serve(args[1:], stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func serve(args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	address := flags.String("address", "127.0.0.1:8080", "HTTP listen address")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return server.Serve(ctx, *address)
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `HydroGo development CLI

Usage:
  hydro version
  hydro serve [--address HOST:PORT]
  hydro help

`)
}
