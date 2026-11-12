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

	// 4. Payment secrets in config
	if cfg.Payments.PayTR.Enabled() {
		if isInsecureSecret(cfg.Payments.PayTR.MerchantKey) {
			issues = append(issues, "payments.paytr.merchant_key is empty or set to CHANGE_ME")
		}
		if isInsecureSecret(cfg.Payments.PayTR.MerchantSalt) {
			issues = append(issues, "payments.paytr.merchant_salt is empty or set to CHANGE_ME")
		}
	}
	if cfg.Payments.Iyzico.Enabled() {
		if isInsecureSecret(cfg.Payments.Iyzico.APIKey) {
			issues = append(issues, "payments.iyzico.api_key is empty or set to CHANGE_ME")
		}
		if isInsecureSecret(cfg.Payments.Iyzico.SecretKey) {
			issues = append(issues, "payments.iyzico.secret_key is empty or set to CHANGE_ME")
		}
	}
	shopierKey, hasShopierKey := lookup("SHOPIER_API_KEY")
	shopierSecret, hasShopierSecret := lookup("SHOPIER_API_SECRET")
	if hasShopierKey || hasShopierSecret {
		if isInsecureSecret(shopierKey) {
			issues = append(issues, "SHOPIER_API_KEY is empty or set to CHANGE_ME")
		}
		if isInsecureSecret(shopierSecret) {
			issues = append(issues, "SHOPIER_API_SECRET is empty or set to CHANGE_ME")
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
