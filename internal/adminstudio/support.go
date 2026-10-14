package adminstudio

import (
	"context"
	"net/http"
	"strings"
)

type SupportConfig struct {
	Enabled                  bool
	MaintenanceActive        bool
	BrokenLinksActive        bool
	BrokenLinksJobID         string
	SafeModeActive           bool
	SiteModuleEnabled        bool
	CommerceModuleEnabled    bool
	CollectionsModuleEnabled bool
	StaffNoticeActive        bool
	StaffNoticeMessage       string
	StaffNoticeSeverity      string
	StaffNoticeDismissKey    string
	AgencyName               string
	Email                    string
	Phone                    string
	WhatsAppURL              string
	SLABadge                 string
	GuideURL                 string
	// Modules is the set of enabled modules; the admin shell hides navigation
	// for inactive ones. Nil means unknown, which hides nothing.
	Modules map[string]bool
	// ContentTypes lists the YAML content types for the admin sidebar.
	ContentTypes []ContentTypeLink
}

// ContentTypeLink is one content type in the admin sidebar.
type ContentTypeLink struct {
	ID    string
	Label string
}

func SupportConfigResolverMiddleware(config SupportConfig, resolve func(*http.Request) SupportConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			current := config
			if resolve != nil && r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/admin") && strings.Contains(r.Header.Get("Accept"), "text/html") {
				resolved := resolve(r)
				current.StaffNoticeActive = resolved.StaffNoticeActive
				current.StaffNoticeMessage = resolved.StaffNoticeMessage
				current.StaffNoticeSeverity = resolved.StaffNoticeSeverity
				current.StaffNoticeDismissKey = resolved.StaffNoticeDismissKey
			}
			next.ServeHTTP(w, r.WithContext(WithSupportConfig(r.Context(), current)))
		})
	}
}

type supportContextKey struct{}

func WithSupportConfig(ctx context.Context, config SupportConfig) context.Context {
	return context.WithValue(ctx, supportContextKey{}, config)
}

func SupportConfigFromContext(ctx context.Context) SupportConfig {
	if ctx == nil {
		return SupportConfig{}
	}
	config, _ := ctx.Value(supportContextKey{}).(SupportConfig)
	return config
}

func SupportConfigMiddleware(config SupportConfig) func(http.Handler) http.Handler {
	return SupportConfigResolverMiddleware(config, nil)
}
