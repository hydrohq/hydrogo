package media

// AdminScriptHashes returns the CSP hashes for inline scripts on the media
// library page. There are none any more: the page loads
// /admin-assets/js/media-library.js instead, which 'self' already allows.
func AdminScriptHashes() []string {
	return nil
}
