package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type Config struct {
	Profile  capacity.Profile `yaml:"profile"`
	Modules  []string         `yaml:"modules"`
	HTTP     HTTP             `yaml:"http"`
	Database Database         `yaml:"database"`
	Media    Media            `yaml:"media"`
	ThemeDir string           `yaml:"theme_dir"`
	Theme    string           `yaml:"theme"`
	Security Security         `yaml:"security"`
	Mail     Mail             `yaml:"mail"`
	Webhooks []Webhook        `yaml:"webhooks"`
	Payments Payments         `yaml:"-"`
}

type Webhook struct {
	ID     string   `yaml:"id"`
	URL    string   `yaml:"url"`
	Secret string   `yaml:"-"`
	Topics []string `yaml:"topics"`
}

type HTTP struct {
	Address      string `yaml:"address"`
	PublicOrigin string `yaml:"public_origin"`
}

type Database struct {
	Dialect database.Dialect `yaml:"dialect"`
	DSN     string           `yaml:"dsn"`
}

type Media struct {
	Dir string `yaml:"dir"`
}

type Security struct {
	MFAMasterKey  string `yaml:"mfa_master_key"`
	SecureCookies bool   `yaml:"secure_cookies"`
	EnableHSTS    bool   `yaml:"enable_hsts"`
}

type Mail struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	From     string `yaml:"from"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	StartTLS bool   `yaml:"starttls"`
}

type Payments struct {
	StripeSecretKey     string
	StripeWebhookSecret string
	IyzicoSecretKey     string
	PayTRMerchantKey    string
	PayTRMerchantSalt   string
	PayTRCurrency       string
}

type Overrides struct {
	CapacityProfile *string
	HTTPAddress     *string
	DatabaseDialect *string
	DatabaseDSN     *string
}

type LookupEnv func(string) (string, bool)

func Load(path string, lookup LookupEnv, overrides Overrides) (Config, error) {
	if lookup == nil {
		lookup = func(string) (string, bool) { return "", false }
	}
	result := Config{Profile: capacity.Compact, Modules: []string{"core", "auth", "pages", "media", "site", "news"}, HTTP: HTTP{Address: "127.0.0.1:8080"}, Database: Database{Dialect: database.SQLite}, Media: Media{Dir: "data/media"}, ThemeDir: "themes", Mail: Mail{Port: 587, StartTLS: true}}
	if path != "" {
		file, err := os.Open(path)
		if err != nil {
			return Config{}, err
		}
		defer file.Close()
		decoder := yaml.NewDecoder(file)
		decoder.KnownFields(true)
		if err := decoder.Decode(&result); err != nil {
			return Config{}, fmt.Errorf("decode config: %w", err)
		}
	}

	if value, ok := lookup("HYDRO_HTTP_ADDRESS"); ok {
		result.HTTP.Address = value
	}
	if value, ok := lookup("HYDRO_CAPACITY_PROFILE"); ok && value != "" {
		result.Profile = capacity.Profile(value)
	}
	if value, ok := lookup("HYDRO_PUBLIC_ORIGIN"); ok && value != "" {
		result.HTTP.PublicOrigin = value
	}
	if value, ok := lookup("HYDRO_DATABASE_DIALECT"); ok {
		result.Database.Dialect = database.Dialect(value)
	}
	if value, ok := lookup("HYDRO_MEDIA_DIR"); ok && value != "" {
		result.Media.Dir = value
	}
	if value, ok := lookup("HYDRO_THEME_DIR"); ok && value != "" {
		result.ThemeDir = value
	}
	if value, ok := lookup("HYDRO_THEME"); ok && value != "" {
		result.Theme = value
	}
	if value, ok := lookup("HYDRO_WEBHOOK_URL"); ok && value != "" {
		if len(result.Webhooks) == 0 {
			result.Webhooks = []Webhook{{ID: "default", URL: value}}
		} else {
			result.Webhooks[0].URL = value
		}
	}
	if value, ok := lookup("HYDRO_WEBHOOK_SECRET"); ok && value != "" {
		if len(result.Webhooks) == 0 {
			result.Webhooks = []Webhook{{ID: "default"}}
		}
		result.Webhooks[0].Secret = value
	}
	if value, ok := lookup("HYDRO_WEBHOOK_TOPICS"); ok && value != "" {
		if len(result.Webhooks) == 0 {
			result.Webhooks = []Webhook{{ID: "default"}}
		}
		result.Webhooks[0].Topics = strings.Split(value, ",")
	}
	if value, ok := lookup("HYDRO_MFA_MASTER_KEY"); ok && value != "" {
		result.Security.MFAMasterKey = value
	}
	if value, ok := lookup("HYDRO_SECURE_COOKIES"); ok && value != "" {
		secureCookies, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_SECURE_COOKIES: %w", err)
		}
		result.Security.SecureCookies = secureCookies
	}
	if value, ok := lookup("HYDRO_ENABLE_HSTS"); ok && value != "" {
		enableHSTS, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_ENABLE_HSTS: %w", err)
		}
		result.Security.EnableHSTS = enableHSTS
	}
	if value, ok := lookup("HYDRO_SMTP_HOST"); ok && value != "" {
		result.Mail.Host = value
	}
	if value, ok := lookup("HYDRO_SMTP_FROM"); ok && value != "" {
		result.Mail.From = value
	}
	if value, ok := lookup("HYDRO_SMTP_USERNAME"); ok && value != "" {
		result.Mail.Username = value
	}
	if value, ok := lookup("HYDRO_SMTP_PASSWORD"); ok && value != "" {
		result.Mail.Password = value
	}
	if value, ok := lookup("HYDRO_SMTP_PORT"); ok && value != "" {
		port, err := strconv.Atoi(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_SMTP_PORT: %w", err)
		}
		result.Mail.Port = port
	}
	if value, ok := lookup("HYDRO_SMTP_STARTTLS"); ok && value != "" {
		startTLS, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_SMTP_STARTTLS: %w", err)
		}
		result.Mail.StartTLS = startTLS
	}
	result.Payments.StripeSecretKey, _ = lookup("HYDRO_STRIPE_SECRET_KEY")
	result.Payments.StripeWebhookSecret, _ = lookup("HYDRO_STRIPE_WEBHOOK_SECRET")
	result.Payments.IyzicoSecretKey, _ = lookup("HYDRO_IYZICO_SECRET_KEY")
	result.Payments.PayTRMerchantKey, _ = lookup("HYDRO_PAYTR_MERCHANT_KEY")
	result.Payments.PayTRMerchantSalt, _ = lookup("HYDRO_PAYTR_MERCHANT_SALT")
	result.Payments.PayTRCurrency, _ = lookup("HYDRO_PAYTR_CURRENCY")
	if result.Payments.PayTRCurrency == "" {
		result.Payments.PayTRCurrency = "TRY"
	}

	dsnSources := 0
	if result.Database.DSN != "" {
		dsnSources++
	}
	if value, ok := lookup("HYDRO_DATABASE_DSN"); ok && value != "" {
		dsnSources++
		result.Database.DSN = value
	}
	if overrides.DatabaseDSN != nil && *overrides.DatabaseDSN != "" {
		dsnSources++
		result.Database.DSN = *overrides.DatabaseDSN
	}
	if dsnSources > 1 {
		return Config{}, errors.New("database DSN is defined by multiple secret sources")
	}

	if overrides.HTTPAddress != nil {
		result.HTTP.Address = *overrides.HTTPAddress
	}
	if overrides.CapacityProfile != nil {
		result.Profile = capacity.Profile(*overrides.CapacityProfile)
	}
	if overrides.DatabaseDialect != nil {
		result.Database.Dialect = database.Dialect(*overrides.DatabaseDialect)
	}
	if err := result.Validate(); err != nil {
		return Config{}, err
	}
	return result, nil
}

func (c Config) Validate() error {
	if err := c.Profile.Validate(); err != nil {
		return err
	}
	seenModules := make(map[string]struct{}, len(c.Modules))
	for _, module := range c.Modules {
		module = strings.TrimSpace(module)
		if module == "" {
			return errors.New("module name cannot be empty")
		}
		if _, exists := seenModules[module]; exists {
			return fmt.Errorf("module %q is configured more than once", module)
		}
		seenModules[module] = struct{}{}
	}
	if (c.Payments.StripeSecretKey == "") != (c.Payments.StripeWebhookSecret == "") {
		return errors.New("Stripe secret and webhook secret must be configured together")
	}
	if (c.Payments.PayTRMerchantKey == "") != (c.Payments.PayTRMerchantSalt == "") {
		return errors.New("PayTR merchant key and salt must be configured together")
	}
	for _, webhook := range c.Webhooks {
		if strings.TrimSpace(webhook.ID) == "" || strings.TrimSpace(webhook.URL) == "" || strings.TrimSpace(webhook.Secret) == "" {
			return errors.New("webhook id, url and environment secret are required")
		}
		parsed, err := url.Parse(webhook.URL)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return fmt.Errorf("invalid webhook URL %q", webhook.URL)
		}
		if len(webhook.Topics) > 64 {
			return errors.New("webhook topics exceed limit")
		}
		for _, topic := range webhook.Topics {
			if strings.TrimSpace(topic) == "" {
				return errors.New("webhook topic cannot be empty")
			}
		}
	}
	if _, _, err := net.SplitHostPort(c.HTTP.Address); err != nil {
		return fmt.Errorf("invalid HTTP address %q: %w", c.HTTP.Address, err)
	}
	var publicOrigin *url.URL
	if strings.TrimSpace(c.HTTP.PublicOrigin) != "" {
		parsed, err := url.Parse(c.HTTP.PublicOrigin)
		if err != nil || !isHTTPOrigin(parsed) {
			return fmt.Errorf("invalid public origin %q", c.HTTP.PublicOrigin)
		}
		publicOrigin = parsed
	}
	if c.Security.EnableHSTS && (publicOrigin == nil || publicOrigin.Scheme != "https") {
		return errors.New("HSTS requires an HTTPS public origin")
	}
	if c.Security.EnableHSTS && !c.Security.SecureCookies {
		return errors.New("HSTS requires secure cookies")
	}
	if c.Security.SecureCookies && (publicOrigin == nil || publicOrigin.Scheme != "https") {
		return errors.New("secure cookies require an HTTPS public origin")
	}
	if publicOrigin != nil && publicOrigin.Scheme == "https" && !c.Security.SecureCookies {
		return errors.New("HTTPS public origin requires secure cookies")
	}
	switch c.Database.Dialect {
	case database.SQLite, database.Postgres:
	default:
		return fmt.Errorf("unsupported database dialect %q", c.Database.Dialect)
	}
	if strings.TrimSpace(c.Database.DSN) == "" {
		return errors.New("database DSN is required")
	}
	if strings.TrimSpace(c.Media.Dir) == "" {
		return errors.New("media directory is required")
	}
	if c.Mail.Host == "" {
		if c.Mail.From != "" || c.Mail.Username != "" || c.Mail.Password != "" {
			return errors.New("mail host is required when SMTP is configured")
		}
	} else {
		if strings.TrimSpace(c.Mail.From) == "" {
			return errors.New("mail from is required when SMTP is configured")
		}
		if c.Mail.Port <= 0 || c.Mail.Port > 65535 {
			return errors.New("mail port is invalid")
		}
		if c.Mail.Username == "" && c.Mail.Password != "" {
			return errors.New("mail password requires username")
		}
		if c.Mail.Username != "" && !c.Mail.StartTLS {
			return errors.New("mail SMTP AUTH requires STARTTLS")
		}
	}
	return nil
}

func isHTTPOrigin(parsed *url.URL) bool {
	if parsed == nil || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return (parsed.Path == "" || parsed.Path == "/") && parsed.RawQuery == "" && parsed.Fragment == ""
}
