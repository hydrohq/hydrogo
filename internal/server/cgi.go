package server

import (
	"context"
	"net/http"
	"net/http/cgi"
)

// ServeCGI serves one request using the standard CGI environment. It is
// intended for process-per-request shared hosting, not FastCGI workers.
func ServeCGI(handler http.Handler) error {
	return cgi.Serve(handler)
}

// LazyCron invokes bounded, idempotent due-work checks when a CGI request
// arrives. CGI cannot keep a resident worker alive between requests.
func LazyCron(next http.Handler, run func(context.Context)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if run != nil {
			run(r.Context())
		}
		next.ServeHTTP(w, r)
	})
}
