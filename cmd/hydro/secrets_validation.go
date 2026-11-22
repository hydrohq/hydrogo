package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/hydrohq/hydrogo/internal/config"
)

func isInsecureSecret(val string) bool {
	v := strings.TrimSpace(val)
	return v == "" || strings.EqualFold(v, "CHANGE_ME") || strings.EqualFold(v, "CHANGEME")
}

func validateSecrets(cfg config.Config, lookup config.LookupEnv, isProd bool, logger *slog.Logger) error {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	var issues []string

	// 1. HYDRO_JWT_SECRET
	jwtSecret, _ := lookup("HYDRO_JWT_SECRET")
	if isInsecureSecret(jwtSecret) {
		issues = append(issues, "HYDRO_JWT_SECRET is empty or set to CHANGE_ME")
	}

	// 2. Webhook secrets
	for i, wh := range cfg.Webhooks {
		if isInsecureSecret(wh.Secret) {
			issues = append(issues, fmt.Sprintf("webhooks[%d] (%s) secret is empty or set to CHANGE_ME", i, wh.URL))
		}
	}
	if whSecret, ok := lookup("HYDRO_WEBHOOK_SECRET"); ok && isInsecureSecret(whSecret) {
		issues = append(issues, "HYDRO_WEBHOOK_SECRET is empty or set to CHANGE_ME")
	}

	// 3. RECOVER_SECRET
	recoverSecret, ok := lookup("RECOVER_SECRET")
	if !ok {
		recoverSecret, ok = lookup("HYDRO_RECOVER_SECRET")
	}
	if !ok || isInsecureSecret(recoverSecret) {
		issues = append(issues, "RECOVER_SECRET is empty or set to CHANGE_ME")
	}

	// 4. Secrets and settings of optional add-on modules (registered config extensions)
	issues = append(issues, config.ExtensionProductionIssues(cfg, lookup)...)

	// 8. Analytics hash key must not remain on a local/dev key
	if cfg.Analytics.NoCookieEnabled || strings.TrimSpace(cfg.Analytics.HashKey) != "" {
		if config.IsTestAnalyticsHashKey(cfg.Analytics.HashKey) {
			issues = append(issues, "analytics.hash_key is set to a development or placeholder key")
		}
	}

	if len(issues) == 0 {
		return nil
	}

	if isProd {
		return fmt.Errorf("production startup failed due to insecure secret configuration: %s", strings.Join(issues, "; "))
	}

	if logger != nil {
		for _, issue := range issues {
			logger.Warn("insecure secret configuration in development", "issue", issue)
		}
	}
	return nil
}
