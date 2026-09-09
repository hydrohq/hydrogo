package security

import (
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"runtime/debug"
)

// RecoveryMiddleware converts unexpected handler panics into a generic 500
// response while retaining the request correlation id in the surrounding
// error middleware. http.ErrAbortHandler is deliberately re-panicked because
// net/http uses it to terminate a connection without writing a response.
func RecoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				value := recover()
				if value == nil {
					return
				}
				if isAbortHandler(value) {
					panic(value)
				}
				if logger != nil {
					logger.ErrorContext(r.Context(), "http_handler_panic",
						slog.String("request_id", RequestIDFromContext(r.Context())),
						// Panic values may contain credentials, tokens, or request
						// data. Keep the type for diagnosis without logging the value.
						slog.String("panic_type", fmt.Sprintf("%T", value)),
						slog.String("stack", string(debug.Stack())),
					)
				}
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func isAbortHandler(value any) bool {
	valueType := reflect.TypeOf(value)
	abortType := reflect.TypeOf(http.ErrAbortHandler)
	if valueType == nil || valueType != abortType || !reflect.ValueOf(value).Comparable() {
		return false
	}
	return value == http.ErrAbortHandler
}
