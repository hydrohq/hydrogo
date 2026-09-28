package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/sdkdist"
)

func sdkCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "generate" {
		return errors.New("sdk requires the generate subcommand")
	}
	flags := flag.NewFlagSet("sdk generate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	out := flags.String("out", "sdk", "Output directory")
	openapi := flags.String("openapi", "docs/api/public-content-v1.openapi.json", "OpenAPI 3.1 JSON input")
	language := flags.String("lang", "all", "Client language: all, typescript or dart")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("sdk generate accepts only --out, --openapi and --lang")
	}
	lang := sdkdist.SDKLanguage(strings.ToLower(strings.TrimSpace(*language)))
	files, err := sdkdist.GenerateOpenAPISDK(*openapi, *out, lang)
	if err != nil {
		return err
	}
	for _, path := range files {
		if _, err := fmt.Fprintln(stdout, path); err != nil {
			return err
		}
	}
	return nil
}
