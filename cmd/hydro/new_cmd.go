package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/generator"
	"github.com/ardaozyildirim/hydrogo/internal/projectgen"
)

// projectCommand is the SDK-oriented namespace for project lifecycle
// commands. Keep `hydro new` as a backwards-compatible alias while exposing
// the discoverable `hydro project new` spelling used by the public SDK.
func projectCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("project requires a subcommand (new or check)")
	}
	switch args[0] {
	case "new":
		return newProjectCommand(args[1:], stdin, stdout, stderr)
	case "check":
		return projectCheckCommand(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown project subcommand %q (supported: new, check)", args[0])
	}
}

// interactiveResourceCommand authors the same Blueprint consumed by scaffold;
// it is intentionally a thin prompt layer, not a second schema language.
func interactiveResourceCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("new resource", flag.ContinueOnError)
	flags.SetOutput(stderr)
	out := flags.String("out", ".", "Client Edition output root")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("new resource accepts only --out")
	}
	in := bufio.NewReader(stdin)
	read := func(prompt string) (string, error) {
		if _, err := fmt.Fprint(stdout, prompt); err != nil {
			return "", err
		}
		value, err := in.ReadString('\n')
		return strings.TrimSpace(value), err
	}
	name, err := read("Resource name: ")
	if err != nil {
		return err
	}
	fields, err := read("Fields (name:type[:required], comma-separated): ")
	if err != nil {
		return err
	}
	if name == "" || fields == "" {
		return errors.New("resource name and fields are required")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "name: %s\nfields:\n", name)
	for _, raw := range strings.Split(fields, ",") {
		parts := strings.Split(strings.TrimSpace(raw), ":")
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid field %q", raw)
		}
		fmt.Fprintf(&b, "  - name: %s\n    type: %s\n", parts[0], parts[1])
		if len(parts) > 2 && strings.EqualFold(parts[2], "required") {
			b.WriteString("    required: true\n")
		}
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	path := filepath.Join(*out, strings.ToLower(name)+".blueprint.yaml")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("blueprint already exists: %s", path)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "blueprint-created path=%s\n", path)
	if err != nil {
		return err
	}
	// Re-enter the canonical scaffold pipeline; interactive mode never grows a
	// second generator or bypasses the generated/custom boundary.
	return generateScaffoldCommand([]string{"--blueprint", path, "--out", *out}, stdout, stderr)
}

func projectCheckCommand(args []string, stdout, stderr io.Writer) error {
	lockPath := ""
	configPath := ""
	clean := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--lock":
			if i+1 >= len(args) {
				return errors.New("--lock requires a path")
			}
			lockPath = args[i+1]
			i++
		case strings.HasPrefix(arg, "--lock="):
			lockPath = strings.TrimPrefix(arg, "--lock=")
		case arg == "--config":
			if i+1 >= len(args) {
				return errors.New("--config requires a path")
			}
			configPath = args[i+1]
			clean = append(clean, arg, configPath)
			i++
		case strings.HasPrefix(arg, "--config="):
			configPath = strings.TrimPrefix(arg, "--config=")
			clean = append(clean, arg)
		default:
			clean = append(clean, arg)
		}
	}
	if lockPath == "" {
		lockPath = "hydrogo.lock"
		if configPath != "" {
			lockPath = filepath.Join(filepath.Dir(configPath), "hydrogo.lock")
		}
	}
	if _, err := os.Stat(lockPath); err == nil {
		if _, err := projectgen.LoadLock(lockPath); err != nil {
			return fmt.Errorf("validate project lock: %w", err)
		}
	}
	projectRoot := "."
	if configPath != "" {
		projectRoot = filepath.Dir(configPath)
	}
	generatedRoot := filepath.Join(projectRoot, "generated")
	if info, err := os.Stat(generatedRoot); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("generated path is not a directory: %s", generatedRoot)
		}
		if _, err := generator.CheckGeneratedCompatibility(projectRoot); err != nil {
			return fmt.Errorf("validate generated resources: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect generated resources: %w", err)
	}
	return validateCommand(clean, stdout, stderr)
}

func newProjectCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("new", flag.ContinueOnError)
	flags.SetOutput(stderr)

	preset := flags.String("preset", "", "Preset: news, editorial, commerce, membership, dashboard, erp, compact-site, corporate, portal or headless")
	profile := flags.String("profile", "compact", "Capacity profile: compact, standard or distributed")
	database := flags.String("database", "sqlite", "Database dialect: sqlite or postgres")
	presentation := flags.String("presentation", "integrated", "Presentation: integrated or headless")
	edition := flags.String("edition", "starter", "License edition: starter, growth, enterprise or custom")
	noDemo := flags.Bool("no-demo", false, "Do not create the seed directory")
	withGit := flags.Bool("git", false, "Run git init in the new project")
	root := flags.String("root", ".", "Parent directory for the new project")
	frontendManifest := flags.String("frontend-manifest", "", "JSON frontend build manifest containing routes and test_id selectors")
	designSystem := flags.String("design-system", "native", "Design system provider identifier (native, shadcn, material, carbon or custom)")

	projectName := ""
	parseArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		projectName = args[0]
		parseArgs = args[1:]
	}

	if err := flags.Parse(parseArgs); err != nil {
		return err
	}

	if projectName == "" && flags.NArg() == 1 {
		projectName = flags.Arg(0)
	}

	// If no project name is specified and no flags given, prompt interactively
	if projectName == "" && len(args) == 0 && stdin != nil {
		reader := bufio.NewReader(stdin)

		fmt.Fprint(stdout, "Proje adı (varsayılan: my-project): ")
		inputName, _ := reader.ReadString('\n')
		inputName = strings.TrimSpace(inputName)
		if inputName == "" {
			projectName = "my-project"
		} else {
			projectName = inputName
		}

		fmt.Fprint(stdout, "Preset [news/editorial/commerce/membership/dashboard/erp/compact-site/corporate/portal/headless] (varsayılan: news): ")
		inputPreset, _ := reader.ReadString('\n')
		inputPreset = strings.TrimSpace(inputPreset)
		if inputPreset == "" {
			*preset = "news"
		} else {
			*preset = inputPreset
		}

		fmt.Fprint(stdout, "Veritabanı [sqlite/postgres] (varsayılan: sqlite): ")
		inputDB, _ := reader.ReadString('\n')
		inputDB = strings.TrimSpace(inputDB)
		if inputDB != "" {
			*database = inputDB
		}

		fmt.Fprint(stdout, "Profil [compact/standard/distributed] (varsayılan: compact): ")
		inputProf, _ := reader.ReadString('\n')
		inputProf = strings.TrimSpace(inputProf)
		if inputProf != "" {
			*profile = inputProf
		}

		fmt.Fprint(stdout, "Sunum tipi [integrated/headless] (varsayılan: integrated): ")
		inputPres, _ := reader.ReadString('\n')
		inputPres = strings.TrimSpace(inputPres)
		if inputPres != "" {
			*presentation = inputPres
		}

		fmt.Fprint(stdout, "Demo içerik eklensin mi? [Y/n]: ")
		inputDemo, _ := reader.ReadString('\n')
		inputDemo = strings.TrimSpace(strings.ToLower(inputDemo))
		if inputDemo == "n" || inputDemo == "no" || inputDemo == "hayır" {
			*noDemo = true
		}

		fmt.Fprint(stdout, "Git başlatılsın mı? [y/N]: ")
		inputGit, _ := reader.ReadString('\n')
		inputGit = strings.TrimSpace(strings.ToLower(inputGit))
		if inputGit == "y" || inputGit == "yes" || inputGit == "evet" {
			*withGit = true
		}
	} else if *preset == "" {
		*preset = "news"
	}

	if flags.NArg() > 1 || strings.TrimSpace(projectName) == "" {
		return errors.New("new requires a project name (e.g. hydro new haber-merkezi --preset=news)")
	}

	path, err := projectgen.Generate(projectgen.Options{
		Root:             *root,
		Name:             projectName,
		Preset:           *preset,
		Profile:          *profile,
		Database:         *database,
		Presentation:     *presentation,
		Edition:          *edition,
		Demo:             !*noDemo,
		E2E:              true,
		Git:              *withGit,
		FrontendManifest: *frontendManifest,
		DesignSystem:     *designSystem,
	})
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "project-created path=%s preset=%s core=licensed-runtime\n", path, *preset)
	return err
}
