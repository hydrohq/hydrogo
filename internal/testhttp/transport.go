// Package testhttp contains in-memory HTTP transports for tests that do not
// need to exercise socket behavior.
package testhttp

import (
	"net/http"
	"net/http/httptest"
)

// Handlers routes requests to a handler by URL host. The empty host is a
// fallback for tests with one upstream.
type Handlers map[string]http.Handler

func (handlers Handlers) RoundTrip(request *http.Request) (*http.Response, error) {
	handler := handlers[request.URL.Host]
	if handler == nil {
		handler = handlers[""]
	}
	if handler == nil {
		return nil, http.ErrNotSupported
	}
	copy := request.Clone(request.Context())
	copy.RequestURI = ""
	if copy.URL.Path == "" {
		copy.URL.Path = "/"
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, copy)
	response := recorder.Result()
	response.Request = request
	return response, nil
}

// Client returns an HTTP client that serves requests directly through the
// supplied handlers without opening a listener or making a network call.
func Client(handlers Handlers) *http.Client {
	return &http.Client{Transport: handlers}
}
