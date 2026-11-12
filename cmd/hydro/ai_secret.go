package main

import (
	"os"
	"strings"
)

// aiMasterSecret is the key that protects stored AI provider keys.
func (a *serveApp) aiMasterSecret() string {
	if secret := strings.TrimSpace(os.Getenv("HYDRO_AI_KEY_SECRET")); secret != "" {
		return secret
	}
	if secret := strings.TrimSpace(a.cfg.Security.MFAMasterKey); secret != "" {
		return secret
	}
	if secret := strings.TrimSpace(a.cfg.Security.PreviewSigningKey); secret != "" {
		return secret
	}
	// No implicit default: encrypting with a well-known fallback would put
	// every deployment on the same key.
	return ""
}
