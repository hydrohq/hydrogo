package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/capacity"
)

func profileCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("profile requires explain or check")
	}
	flags := flag.NewFlagSet("profile "+args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	profileName := flags.String("profile", string(capacity.Compact), "compact, standard or distributed")
	jsonOutput := flags.Bool("json", false, "emit JSON")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("profile accepts only --profile and --json")
	}
	profile := capacity.Profile(strings.ToLower(strings.TrimSpace(*profileName)))
	budget, err := capacity.Defaults(profile)
	if err != nil {
		return err
	}
	manifest, err := capacity.Manifest(profile)
	if err != nil {
		return err
	}
	if args[0] == "check" {
		if err := budget.Validate(); err != nil {
			return err
		}
		if *jsonOutput {
			return json.NewEncoder(stdout).Encode(map[string]any{"valid": true, "profile": profile, "capabilities": manifest})
		}
		_, err := fmt.Fprintf(stdout, "valid profile=%s capabilities=%d db_max_open=%d workers=%d\n", profile, len(manifest), budget.DBMaxOpenConns, budget.WorkerConcurrency)
		return err
	}
	if args[0] != "explain" {
		return fmt.Errorf("unknown profile command %q", args[0])
	}
	if *jsonOutput {
		return json.NewEncoder(stdout).Encode(map[string]any{"profile": profile, "budget": budget, "capabilities": manifest})
	}
	if _, err := fmt.Fprintf(stdout, "profile=%s db_max_open=%d db_max_idle=%d cache_entries=%d workers=%d queue_entries=%d\n", profile, budget.DBMaxOpenConns, budget.DBMaxIdleConns, budget.CacheEntries, budget.WorkerConcurrency, budget.QueueEntries); err != nil {
		return err
	}
	for _, item := range manifest {
		if _, err := fmt.Fprintf(stdout, "capability=%s kind=%s activation=%s resources=%s\n", item.Name, item.Kind, item.Activation, strings.Join(item.Resources, ",")); err != nil {
			return err
		}
	}
	return nil
}
