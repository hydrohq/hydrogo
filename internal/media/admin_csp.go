package media

// AdminScriptHashes returns the reviewed hash for the static media library
// page script. The browser acceptance test checks the rendered block against
// this source so edits cannot silently leave the CSP catalog stale.
func AdminScriptHashes() []string {
	return []string{"jZaUqlEd1qqBBHN7Zr5m6BSqZgRyxqVnclI6NYCwNp4="}
}
