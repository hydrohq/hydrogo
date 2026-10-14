package transport

// AdminScriptHashes returns the CSP hashes of inline scripts the news admin
// screens render. There are none: the editors load their behaviour from
// /admin-assets/js/, so nothing needs to be added to the admin script-src.
func AdminScriptHashes() []string {
	return nil
}
