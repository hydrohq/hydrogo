package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/openapi"
	"github.com/ardaozyildirim/hydrogo/internal/sdkdist"
)

func sdkCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "generate" {
		return errors.New("sdk requires the generate subcommand")
	}
	flags := flag.NewFlagSet("sdk generate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	out := flags.String("out", "sdk", "Output directory")
	openapiPath := flags.String("openapi", "docs/api/public-content-v1.openapi.json", "OpenAPI 3.1 JSON input")
	contentTypesDir := flags.String("content-types-dir", "content-types", "Directory with content-types YAML definitions")
	language := flags.String("lang", "all", "Client language: all, typescript or dart")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("sdk generate accepts only --out, --openapi, --content-types-dir and --lang")
	}
	lang := sdkdist.SDKLanguage(strings.ToLower(strings.TrimSpace(*language)))

	var specData []byte
	if *openapiPath != "" {
		if data, err := os.ReadFile(*openapiPath); err == nil {
			specData = data
		}
	}
	if *contentTypesDir != "" {
		if defs, err := contenttypes.LoadDir(*contentTypesDir); err == nil && len(defs) > 0 {
			if merged, err := openapi.BuildDocument(specData, defs); err == nil {
				specData = merged
			}
		}
	}

	var files []string
	var err error
	if len(specData) > 0 {
		files, err = sdkdist.GenerateOpenAPISDKFromBytes(specData, *out, lang)
	} else {
		files, err = sdkdist.GenerateOpenAPISDK(*openapiPath, *out, lang)
	}
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
