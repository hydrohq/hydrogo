package adminstudio

import (
	"context"
	"net/http"
)

type SupportConfig struct {
	Enabled     bool
	AgencyName  string
	Email       string
	Phone       string
	WhatsAppURL string
	SLABadge    string
	GuideURL    string
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
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(WithSupportConfig(r.Context(), config)))
		})
	}
}
