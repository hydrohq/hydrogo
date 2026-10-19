package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/scaffold"
)

// newCommand implements `hydro new <dir>` and `hydro init` (new in the current
// directory): scaffold a project, then optionally migrate and create an admin.
func newCommand(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	preset := flags.String("preset", "minimal", "project preset: "+presetNames())
	adminEmail := flags.String("admin-email", "", "create an admin with this e-mail after scaffolding")
	passwordStdin := flags.Bool("admin-password-stdin", false, "read the admin password from stdin (default: generate one and print it)")
	list := flags.Bool("list-presets", false, "list available presets and exit")
	// Allow the directory before or after the flags: `hydro new site --preset blog`.
	var positional []string
	rest := args
	for len(rest) > 0 {
		if err := flags.Parse(rest); err != nil {
			return err
		}
		if flags.NArg() == 0 {
			break
		}
		positional = append(positional, flags.Arg(0))
		rest = flags.Args()[1:]
	}
	if *list {
		for _, p := range scaffold.Presets() {
			fmt.Fprintf(stdout, "%-10s %s\n", p.Name, p.Description)
		}
		return nil
	}
	dir := "."
	switch {
	case name == "new" && len(positional) != 1:
		return errors.New("usage: hydro new <directory> [--preset NAME] [--admin-email EMAIL]")
	case name == "init" && len(positional) != 0:
		return errors.New("usage: hydro init [--preset NAME] [--admin-email EMAIL] (works in the current directory)")
	case name == "new":
		dir = positional[0]
	}

	created, err := scaffold.Create(scaffold.Options{Dir: dir, Preset: *preset})
	if err != nil {
		return err
	}
	abs, _ := filepath.Abs(dir)
	fmt.Fprintf(stdout, "Created HydroGo project %q (preset %s) in %s\n", filepath.Base(abs), created.Name, abs)

	if *adminEmail == "" {
		fmt.Fprintln(stdout, "\nNext steps:")
		if name == "new" {
			fmt.Fprintf(stdout, "  cd %s\n", dir)
		}
		fmt.Fprintln(stdout, "  hydro migrate --config config.yaml")
		fmt.Fprintln(stdout, "  hydro create-admin --email you@example.com --password-stdin --config config.yaml")
		fmt.Fprintln(stdout, "  hydro serve --config config.yaml")
		fmt.Fprintln(stdout, "\nOr run again with --admin-email to do the first two automatically.")
		return nil
	}

	password, generated := "", false
	if *passwordStdin {
		if password, err = readPassword(stdin); err != nil {
			return fmt.Errorf("read admin password: %w", err)
		}
	} else if env := os.Getenv("HYDRO_ADMIN_PASSWORD"); env != "" {
		password = env
	} else {
		if password, err = randomPassword(); err != nil {
			return err
		}
		generated = true
	}
	// Relative paths in config.yaml (data/, content-types/) resolve against the
	// project directory, so bootstrap from inside it.
	previous, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(dir); err != nil {
		return err
	}
	defer os.Chdir(previous)
	if err := migrate([]string{"--config", "config.yaml"}, stderr); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := createAdmin([]string{"--email", *adminEmail, "--password-stdin", "--config", "config.yaml"}, strings.NewReader(password+"\n"), stdout, stderr); err != nil {
		return err
	}
	if generated {
		fmt.Fprintf(stdout, "Generated admin password (shown once): %s\n", password)
	}
	fmt.Fprintln(stdout, "\nStart it with:")
	if name == "new" {
		fmt.Fprintf(stdout, "  cd %s && hydro serve --config config.yaml\n", dir)
	} else {
		fmt.Fprintln(stdout, "  hydro serve --config config.yaml")
	}
	fmt.Fprintln(stdout, "Then open http://127.0.0.1:8080/admin")
	return nil
}

func presetNames() string {
	names := []string{}
	for _, p := range scaffold.Presets() {
		names = append(names, p.Name)
	}
	return strings.Join(names, "|")
}

// randomPassword returns a random password that passes the admin password
// policy (a random draw can, rarely, contain a rejected common fragment).
func randomPassword() (string, error) {
	for attempt := 0; attempt < 20; attempt++ {
		buf := make([]byte, 18)
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		password := base64.RawURLEncoding.EncodeToString(buf)
		if auth.ValidatePasswordStrength(password) == nil {
			return password, nil
		}
	}
	return "", errors.New("could not generate a password that satisfies the password policy")
}
