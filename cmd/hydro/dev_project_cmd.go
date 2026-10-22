package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// devService is a side process started next to the server: NAME=COMMAND.
type devService struct {
	name string
	argv []string
}

// splitDevArgs takes the --service flags out of args and returns the rest,
// which goes to `hydro serve --dev` unchanged.
func splitDevArgs(args []string) (services []devService, rest []string, err error) {
	for i := 0; i < len(args); i++ {
		value, ok := "", false
		switch {
		case args[i] == "--service" || args[i] == "-service":
			if i+1 >= len(args) {
				return nil, nil, errors.New("--service needs NAME=COMMAND")
			}
			i++
			value, ok = args[i], true
		case strings.HasPrefix(args[i], "--service="):
			value, ok = strings.TrimPrefix(args[i], "--service="), true
		}
		if !ok {
			rest = append(rest, args[i])
			continue
		}
		name, command, found := strings.Cut(value, "=")
		argv := strings.Fields(command)
		if !found || name == "" || len(argv) == 0 {
			return nil, nil, fmt.Errorf("--service %q: expected NAME=COMMAND, for example ledger=go run ./automation", value)
		}
		services = append(services, devService{name: name, argv: argv})
	}
	return services, rest, nil
}

func hasConfigFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--config" || arg == "-config" || strings.HasPrefix(arg, "--config=") || strings.HasPrefix(arg, "-config=") {
			return true
		}
	}
	return false
}

// prefixWriter writes complete lines to out, each marked with the process name.
type prefixWriter struct {
	mu     *sync.Mutex
	out    io.Writer
	prefix string
	part   []byte
}

func (w *prefixWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.part = append(w.part, p...)
	for {
		end := bytes.IndexByte(w.part, '\n')
		if end < 0 {
			break
		}
		fmt.Fprintf(w.out, "%s %s\n", w.prefix, w.part[:end])
		w.part = w.part[end+1:]
	}
	return len(p), nil
}

// projectDevCommand runs the server (`hydro serve --dev`) and the side
// services of a project made with hydro new in one terminal, with prefixed
// output. Ctrl-C stops all of them; if one exits, so do the others.
//
//	hydro dev --service ledger="go run ./automation" [--service name=cmd ...] [serve flags]
//
// A service command is split on spaces (no shell quoting); put anything more
// complex in a script. Without --config, config.yaml is used.
func projectDevCommand(args []string, stdout, stderr io.Writer) error {
	services, rest, err := splitDevArgs(args)
	if err != nil {
		return err
	}
	if !hasConfigFlag(rest) {
		rest = append([]string{"--config", "config.yaml"}, rest...)
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var mu sync.Mutex
	processes := []devService{{name: "hydro", argv: append([]string{self, "serve", "--dev"}, rest...)}}
	processes = append(processes, services...)
	results := make(chan error, len(processes))
	for _, process := range processes {
		command := exec.CommandContext(ctx, process.argv[0], process.argv[1:]...) // #nosec G204 -- the processes are the ones the developer named with hydro dev --service, no shell
		command.Stdout = &prefixWriter{mu: &mu, out: stdout, prefix: "[" + process.name + "]"}
		command.Stderr = &prefixWriter{mu: &mu, out: stderr, prefix: "[" + process.name + "]"}
		command.Cancel = func() error {
			if err := command.Process.Signal(os.Interrupt); err != nil {
				return command.Process.Kill()
			}
			return nil
		}
		command.WaitDelay = 5 * time.Second
		if err := command.Start(); err != nil {
			cancel()
			return fmt.Errorf("start %s: %w", process.name, err)
		}
		go func(name string) {
			err := command.Wait()
			// Ctrl-C reaches every process at once: let our own context notice
			// before an exit by signal is reported as a crash.
			select {
			case <-ctx.Done():
			case <-time.After(300 * time.Millisecond):
			}
			if err != nil && ctx.Err() == nil {
				err = fmt.Errorf("%s stopped: %w", name, err)
			} else if ctx.Err() != nil {
				err = nil
			} else {
				err = fmt.Errorf("%s stopped", name)
			}
			results <- err
		}(process.name)
	}
	first := <-results
	cancel()
	for i := 1; i < len(processes); i++ {
		<-results
	}
	return first
}
