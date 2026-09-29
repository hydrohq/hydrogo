package site

import (
	"errors"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
)

var (
	ErrInvalidSetting           = errors.New("invalid site setting")
	ErrInvalidNavigation        = errors.New("invalid navigation item")
	ErrInvalidRedirect          = errors.New("invalid redirect")
	ErrRedirectStoreRequired    = errors.New("redirect rule store is required")
	ErrInvalidSubmission        = errors.New("invalid form submission")
	ErrInvalidInboxState        = errors.New("invalid inbox state")
	ErrStoreRequired            = errors.New("site store is required")
	ErrInvalidIdempotencyKey    = errors.New("invalid idempotency key")
	ErrIdempotencyConflict      = errors.New("idempotency key conflict")
	ErrIdempotencyStoreRequired = errors.New("idempotency store is required")
	ErrLeadStoreRequired        = errors.New("lead store is required")
	ErrMailQueueRequired        = errors.New("mail queue is required")
)

type Setting struct {
	Key       string
	Value     string
	Public    bool
	UpdatedAt time.Time
}

type ContactWidget struct {
	Enabled        bool
	Phone          string
	WhatsAppNumber string
	Message        string
	Position       string
	MobileOnly     bool
}

func ContactWidgetFromSettings(settings []Setting) ContactWidget {
	values := make(map[string]string, len(settings))
	for _, setting := range settings {
		values[setting.Key] = setting.Value
	}
	position := values["contact_widget.position"]
	if position == "" {
		position = "bottom-right"
	}
	return ContactWidget{
		Enabled: values["contact_widget.enabled"] == "true", Phone: values["contact_widget.phone"],
		WhatsAppNumber: values["contact_widget.whatsapp_number"], Message: values["contact_widget.message"],
		Position: position, MobileOnly: values["contact_widget.mobile_only"] == "true",
	}
}

func ContactWidgetConfigured(settings []Setting) bool {
	for _, setting := range settings {
		if setting.Key == "contact_widget.enabled" {
			return true
		}
	}
	return false
}

func (value ContactWidget) Settings() []Setting {
	boolString := func(value bool) string {
		if value {
			return "true"
		}
		return "false"
	}
	return []Setting{
		{Key: "contact_widget.enabled", Value: boolString(value.Enabled), Public: true},
		{Key: "contact_widget.phone", Value: strings.TrimSpace(value.Phone), Public: true},
		{Key: "contact_widget.whatsapp_number", Value: strings.TrimSpace(value.WhatsAppNumber), Public: true},
		{Key: "contact_widget.message", Value: strings.TrimSpace(value.Message), Public: true},
		{Key: "contact_widget.position", Value: strings.TrimSpace(value.Position), Public: true},
		{Key: "contact_widget.mobile_only", Value: boolString(value.MobileOnly), Public: true},
	}
}

func (value ContactWidget) Validate() error {
	validPhone := func(raw string) bool {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return true
		}
		if len(raw) < 7 || len(raw) > 40 {
			return false
		}
		for i, r := range raw {
			if r == '+' && i == 0 || r >= '0' && r <= '9' || strings.ContainsRune(" ()-.", r) {
				continue
			}
			return false
		}
		digits := 0
		for _, r := range raw {
			if r >= '0' && r <= '9' {
				digits++
			}
		}
		return digits >= 7 && digits <= 15
	}
	if value.Enabled && !validPhone(value.Phone) || value.Enabled && !validPhone(value.WhatsAppNumber) || len(value.Message) > 500 || value.Position != "bottom-right" && value.Position != "bottom-left" || value.Enabled && strings.TrimSpace(value.Phone) == "" && strings.TrimSpace(value.WhatsAppNumber) == "" || strings.ContainsAny(value.Message, "\x00") {
		return ErrInvalidSetting
	}
	return nil
}

// ResolvePublicHomePath returns a configured internal home path when the
// setting is public and safe. Invalid or missing settings deliberately fall
// back to the caller-provided path so a bad editorial value cannot create an
// open redirect or break the root route.
func ResolvePublicHomePath(settings []Setting, fallback string) string {
	for _, setting := range settings {
		if setting.Key != "site.home_path" || !setting.Public {
			continue
		}
		path := strings.TrimSpace(setting.Value)
		if len(path) > 2048 {
			continue
		}
		parsed, err := url.Parse(path)
		decoded, decodeErr := url.PathUnescape(path)
		if err != nil || decodeErr != nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || !strings.HasPrefix(decoded, "/") || strings.HasPrefix(decoded, "//") || strings.ContainsAny(path+decoded, "\r\n\\") {
			continue
		}
		return path
	}
	return fallback
}

type NavigationItem struct {
	ID        string
	Locale    string
	Label     string
	Href      string
	ParentID  string `json:"parent_id,omitempty"`
	Target    string `json:"target,omitempty"`
	Position  int
	Visible   bool
	UpdatedAt time.Time
}

type NavigationSuggestion struct {
	Label string
	Href  string
}

type Redirect struct {
	SourcePath string
	TargetPath string
	Status     int
	CreatedAt  time.Time
}

type RedirectRule struct {
	ID         string
	TenantID   string
	Pattern    string
	Kind       string
	TargetPath string
	Status     int
	Hits       uint64
	CreatedAt  time.Time
}

type Submission struct {
	ID                  string
	FormKey             string
	Email               string
	Payload             string
	SourceIPHash        string
	Status              string
	CreatedAt           time.Time
	IdempotencyKey      string
	RequestHash         string
	LeadStage           string
	AssignedTo          string
	LocationCountryCode string              `json:"location_country_code,omitempty"`
	LocationCity        string              `json:"location_city,omitempty"`
	LocationLocal       bool                `json:"location_local,omitempty"`
	Notifications       []EmailNotification `json:"-"`
}

type EmailNotification struct {
	EventKey string
	To       string
	Subject  string
	Body     string
	HTMLBody string
}

type LeadNote struct {
	ID           string
	TenantID     string
	SubmissionID string
	AuthorID     string
	Note         string
	CreatedAt    time.Time
}

func ValidateIdempotencyKey(key string) error {
	if len(key) < 16 || len(key) > 128 || strings.TrimSpace(key) != key {
		return ErrInvalidIdempotencyKey
	}
	for i := 0; i < len(key); i++ {
		if key[i] < 0x21 || key[i] > 0x7e {
			return ErrInvalidIdempotencyKey
		}
	}
	return nil
}

func ValidateSetting(value Setting) error {
	value.Key = strings.TrimSpace(value.Key)
	if value.Key == "" || len(value.Key) > 128 || strings.ContainsAny(value.Key, " \t\r\n") || len(value.Value) > 64<<10 {
		return ErrInvalidSetting
	}
	if value.Key == "site.home_path" && value.Public && ResolvePublicHomePath([]Setting{value}, "") == "" {
		return ErrInvalidSetting
	}
	return nil
}

func ValidateNavigation(value NavigationItem) error {
	href := strings.TrimSpace(value.Href)
	label := strings.TrimSpace(value.Label)
	if value.ID == "" || strings.TrimSpace(value.ID) != value.ID || strings.ContainsAny(value.ID, "\x00\r\n\t") || len(value.ID) > 128 || len(value.ParentID) > 128 || strings.ContainsAny(value.ParentID, "\x00\r\n\t") || value.ID == value.ParentID || value.Locale == "" || len(value.Locale) > 16 || label == "" || len(label) > 256 || value.Position < 0 || href == "" || len(href) > 2048 || (value.Target != "" && value.Target != "_self" && value.Target != "_blank") {
		return ErrInvalidNavigation
	}
	parsed, err := url.Parse(href)
	// Navigation links are internal references. Reject every absolute or
	// network-path URI, including odd forms such as "https:foo" that have no
	// host but still carry a scheme.
	if err != nil || parsed.Host != "" || parsed.IsAbs() || strings.HasPrefix(strings.ToLower(href), "javascript:") {
		return ErrInvalidNavigation
	}
	return nil
}

func ValidateRedirect(value Redirect) error {
	source := normalizePath(value.SourcePath)
	target := normalizePath(value.TargetPath)
	if source == "" || target == "" || source == target || (value.Status != 301 && value.Status != 308) {
		return ErrInvalidRedirect
	}
	return nil
}

func ValidateRedirectRule(value RedirectRule) error {
	pattern := strings.TrimSpace(value.Pattern)
	target := strings.TrimSpace(value.TargetPath)
	if strings.TrimSpace(value.ID) == "" || len(value.ID) > 128 || len(pattern) == 0 || len(pattern) > 512 || len(target) == 0 || len(target) > 2048 || (value.Status != 301 && value.Status != 302 && value.Status != 308) {
		return ErrInvalidRedirect
	}
	if strings.ContainsAny(pattern+target, "\\\r\n\x00?#") || !strings.HasPrefix(pattern, "/") || !strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") {
		return ErrInvalidRedirect
	}
	parsed, err := url.Parse(target)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.User != nil {
		return ErrInvalidRedirect
	}
	switch value.Kind {
	case "exact":
		if strings.ContainsAny(pattern, "*[](){}") || normalizePath(pattern) != pattern {
			return ErrInvalidRedirect
		}
	case "wildcard":
		if strings.Count(pattern, "*") != 1 || strings.ContainsAny(pattern, "[](){}") {
			return ErrInvalidRedirect
		}
		if strings.Count(target, "*") > 1 {
			return ErrInvalidRedirect
		}
	case "regex":
		if len(pattern) > 256 {
			return ErrInvalidRedirect
		}
		if _, err := regexp.Compile(pattern); err != nil {
			return ErrInvalidRedirect
		}
	default:
		return ErrInvalidRedirect
	}
	return nil
}

func ValidateSubmission(value Submission) error {
	if value.ID == "" || strings.TrimSpace(value.FormKey) == "" || len(value.Payload) > 64<<10 || strings.TrimSpace(value.SourceIPHash) == "" {
		return ErrInvalidSubmission
	}
	code := strings.TrimSpace(value.LocationCountryCode)
	if code != "" && (len(code) != 2 || code[0] < 'A' || code[0] > 'Z' || code[1] < 'A' || code[1] > 'Z') || len(value.LocationCity) > 128 || value.LocationLocal && (code != "" || strings.TrimSpace(value.LocationCity) != "") {
		return ErrInvalidSubmission
	}
	if _, err := mail.ParseAddress(value.Email); err != nil || len(value.Email) > 320 {
		return ErrInvalidSubmission
	}
	return nil
}

func ValidateInboxStatus(status string) error {
	switch strings.TrimSpace(status) {
	case "new", "triaged", "resolved", "deleted":
		return nil
	default:
		return ErrInvalidInboxState
	}
}

func ValidateLeadStage(stage string) error {
	switch strings.TrimSpace(stage) {
	case "new", "contacted", "qualified", "won", "lost":
		return nil
	default:
		return ErrInvalidInboxState
	}
}

func ValidateLeadAssignee(id string) error {
	id = strings.TrimSpace(id)
	if len(id) > 128 || strings.ContainsAny(id, "\r\n\x00") {
		return ErrInvalidInboxState
	}
	return nil
}

func ValidateLeadNote(note LeadNote) error {
	if strings.TrimSpace(note.ID) == "" || len(note.ID) > 128 || strings.TrimSpace(note.SubmissionID) == "" || len(note.SubmissionID) > 128 || strings.TrimSpace(note.AuthorID) == "" || len(note.AuthorID) > 128 || strings.TrimSpace(note.Note) == "" || len(note.Note) > 8192 {
		return ErrInvalidInboxState
	}
	return nil
}

func normalizePath(value string) string {
	path, err := canonical.Path(value)
	if err != nil {
		return ""
	}
	if path == "" {
		return "/"
	}
	return "/" + path
}
