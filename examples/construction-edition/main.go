package main

import (
	"fmt"

	"github.com/ardaozyildirim/hydrogo/extension"
)

const editionName = "hydrogo-construction-edition"

func main() {
	fmt.Printf("%s using %s; workflows=%s\n", editionName, extension.APIStatus, "documents,daily-reports,issues,approvals")
}
