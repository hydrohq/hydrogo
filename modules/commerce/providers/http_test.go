package providers

import (
	"bytes"
	"net/http"
	"testing"
)

func TestReadBoundedRejectsOversizedResponse(t *testing.T) {
	if _, err := ReadBounded(bytes.NewReader(make([]byte, MaxResponseBytes+1))); err != ErrResponseTooLarge {
		t.Fatalf("error = %v", err)
	}
}

func TestHTTPClientRejectsRedirectsByDefault(t *testing.T) {
	client := HTTPClient(nil)
	if client.CheckRedirect == nil || client.CheckRedirect(nil, nil) != http.ErrUseLastResponse {
		t.Fatal("redirect policy is not fail-closed")
	}
}
