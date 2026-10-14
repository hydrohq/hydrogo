package ui

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui/shell"
)

// AdminScriptHashes lists the CSP hashes of the inline scripts the admin may
// render. Page behaviour is loaded from /admin-assets/js/ and needs no hash;
// the only inline script left is the shell's pre-paint theme bootstrap, which
// has to run before the stylesheets load.
func AdminScriptHashes() ([]string, error) {
	var document bytes.Buffer
	if err := shell.Head(shell.View{}).Render(context.Background(), &document); err != nil {
		return nil, fmt.Errorf("render Admin script catalog: %w", err)
	}
	scripts, _ := security.InlineCSPHashes(document.Bytes())
	return uniqueScriptHashes(withoutEmptyBlocks(scripts)), nil
}

// emptyBlockHash is the hash of an empty <script> body. security.InlineCSPHashes
// also hashes <script src> tags, whose body is empty; allowing that hash only
// permits empty inline scripts, so it is left out of the policies built here.
var emptyBlockHash = func() string {
	digest := sha256.Sum256(nil)
	return base64.StdEncoding.EncodeToString(digest[:])
}()

func withoutEmptyBlocks(hashes []string) []string {
	kept := make([]string, 0, len(hashes))
	for _, source := range hashes {
		if source != emptyBlockHash {
			kept = append(kept, source)
		}
	}
	return kept
}

// inlineCSPDirectives returns script-src and style-src hash sources for the
// non-empty inline blocks in a rendered document, or "" when there are none.
func inlineCSPDirectives(document []byte) string {
	scripts, styles := security.InlineCSPHashes(document)
	var directives []string
	for _, directive := range []struct {
		name   string
		hashes []string
	}{{"script-src", withoutEmptyBlocks(scripts)}, {"style-src", withoutEmptyBlocks(styles)}} {
		if len(directive.hashes) == 0 {
			continue
		}
		quoted := make([]string, 0, len(directive.hashes))
		for _, source := range uniqueScriptHashes(directive.hashes) {
			quoted = append(quoted, "'sha256-"+source+"'")
		}
		directives = append(directives, directive.name+" "+strings.Join(quoted, " "))
	}
	return strings.Join(directives, "; ")
}

func uniqueScriptHashes(hashes []string) []string {
	seen := make(map[string]struct{}, len(hashes))
	unique := make([]string, 0, len(hashes))
	for _, source := range hashes {
		if source == "" {
			continue
		}
		if _, exists := seen[source]; exists {
			continue
		}
		seen[source] = struct{}{}
		unique = append(unique, source)
	}
	return unique
}
