package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/ardaozyildirim/hydrogo/internal/eventcatalog"
)

// eventsCommand exposes the published vocabulary without granting delivery,
// subscription, or runtime schema authority to the CLI caller.
func eventsCommand(args []string, stdout io.Writer) error {
	if len(args) != 1 || args[0] != "catalog" {
		return errors.New("events requires the catalog subcommand")
	}
	document, err := eventcatalog.ExportJSON()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, string(document))
	return err
}
