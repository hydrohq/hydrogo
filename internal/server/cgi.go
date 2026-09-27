package server

import (
	"context"
	"net"
	"net/http"
	"net/http/cgi"
	"net/http/fcgi"
	"time"
)

// ServeCGI serves one request using the standard CGI environment. It is
// intended for process-per-request shared hosting (e.g. Apache cPanel/Plesk).
func ServeCGI(handler http.Handler) error {
	return cgi.Serve(handler)
}

// ServeFCGI serves FastCGI requests on the given listener.
func ServeFCGI(listener net.Listener, handler http.Handler) error {
	return fcgi.Serve(listener, handler)
}

// LazyCronRunner represents a scheduled job that can be evaluated during
// CGI request lifecycles.
type LazyCronRunner interface {
	RunDue(ctx context.Context) error
}

// LazyCron invokes bounded, idempotent due-work checks when a CGI request
// arrives without blocking the critical client response for longer than timeout.
func LazyCron(next http.Handler, run func(context.Context)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if run != nil {
			cronCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			run(cronCtx)
		}
		next.ServeHTTP(w, r)
	})
}
