package main

import (
	"fmt"

	"github.com/ardaozyildirim/hydrogo/extension"
)

const editionName = "hydrogo-news-edition"

func main() {
	fmt.Printf("%s using %s; review workflow=%s\n", editionName, extension.APIStatus, "scheduled")
}
