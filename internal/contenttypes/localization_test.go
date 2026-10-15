package contenttypes

import (
	"os"
	"path/filepath"
	"testing"
)

const localizedGuide = `id: guide
label: Guide
title_field: title
public: true
drafts: true
revisions: true
fields:
  - {id: title, label: Title, kind: text, required: true, list: true, localized: true}
  - {id: body, label: Body, kind: richtext, localized: true}
  - {id: rank, label: Rank, kind: number, list: true}
`

func TestLocalizedFieldsAreValidated(t *testing.T) {
	for name, content := range map[string]string{
		"boolean":     "id: x\nlabel: X\nrevisions: true\nfields: [{id: a, label: A, kind: boolean, localized: true}]\n",
		"relation":    "id: x\nlabel: X\nrevisions: true\nfields: [{id: a, label: A, kind: relation, to: x, localized: true}]\n",
		"searchable":  "id: x\nlabel: X\nrevisions: true\nfields: [{id: a, label: A, kind: text, searchable: true, localized: true}]\n",
		"unique":      "id: x\nlabel: X\nrevisions: true\nfields: [{id: a, label: A, kind: text, unique: true, localized: true}]\n",
		"no history":  "id: x\nlabel: X\nfields: [{id: a, label: A, kind: text, localized: true}]\n",
		"locale name": "id: x\nlabel: X\nfields: [{id: locale, label: A, kind: text}]\n",
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "x.yaml"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadDir(dir); err == nil {
			t.Errorf("%s: invalid localized definition accepted", name)
		}
	}
}
