package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/projectgen"
)

func newProjectCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("new", flag.ContinueOnError)
	flags.SetOutput(stderr)

	preset := flags.String("preset", "", "Preset: news, editorial, commerce, compact-site, corporate, portal or headless")
	profile := flags.String("profile", "compact", "Capacity profile: compact, standard or distributed")
	database := flags.String("database", "sqlite", "Database dialect: sqlite or postgres")
	presentation := flags.String("presentation", "integrated", "Presentation: integrated or headless")
	edition := flags.String("edition", "starter", "License edition: starter, growth, enterprise or custom")
	noDemo := flags.Bool("no-demo", false, "Do not create the seed directory")
	withGit := flags.Bool("git", false, "Run git init in the new project")
	root := flags.String("root", ".", "Parent directory for the new project")
	frontendManifest := flags.String("frontend-manifest", "", "JSON frontend build manifest containing routes and test_id selectors")

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

		fmt.Fprint(stdout, "Preset [news/editorial/commerce/compact-site/corporate/portal/headless] (varsayılan: news): ")
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
	})
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "project-created path=%s preset=%s core=licensed-runtime\n", path, *preset)
	return err
}
