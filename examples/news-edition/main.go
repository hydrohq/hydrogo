package main

import (
	"fmt"

	"github.com/ardaozyildirim/hydrogo/extension"
)

const editionName = "hydrogo-news-edition"

func main() {
	if err := ValidateEditorialDraft(EditorialDraft{Headline: "example", Desk: "news", ApprovedBy: "editor"}); err != nil {
		panic(err)
	}
	fmt.Printf("%s using %s; review workflow=%s\n", editionName, extension.APIStatus, "scheduled")
}
