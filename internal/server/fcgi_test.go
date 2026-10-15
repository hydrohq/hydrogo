package server

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestServeFCGIResponder(t *testing.T) {
	listener := newMemoryListener()
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- ServeFCGI(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/healthz" {
				http.NotFound(w, r)
				return
			}
			_, _ = io.WriteString(w, "fastcgi-ok")
		}))
	}()
	defer func() {
		_ = listener.Close()
		select {
		case <-serveDone:
		case <-time.After(time.Second):
			t.Error("FastCGI listener did not stop")
		}
	}()

	conn, err := listener.dial()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	params := fastCGIParams(map[string]string{
		"REQUEST_METHOD":    "GET",
		"REQUEST_URI":       "/healthz",
		"SERVER_PROTOCOL":   "HTTP/1.1",
		"SERVER_NAME":       "localhost",
		"SERVER_PORT":       "80",
		"REMOTE_ADDR":       "127.0.0.1",
		"GATEWAY_INTERFACE": "CGI/1.1",
	})
	request := append(fastCGIRecord(1, 1, []byte{0, 1, 0, 0, 0, 0, 0, 0}), fastCGIRecord(4, 1, params)...)
	request = append(request, fastCGIRecord(4, 1, nil)...)
	request = append(request, fastCGIRecord(5, 1, nil)...)
	if _, err := conn.Write(request); err != nil {
		t.Fatal(err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var response strings.Builder
	for {
		header := make([]byte, 8)
		if _, err := io.ReadFull(conn, header); err != nil {
			t.Fatal(err)
		}
		length := int(binary.BigEndian.Uint16(header[4:6]))
		padding := int(header[6])
		payload := make([]byte, length+padding)
		if _, err := io.ReadFull(conn, payload); err != nil {
			t.Fatal(err)
		}
		if header[1] == 6 {
			response.Write(payload[:length])
		}
		if header[1] == 3 {
			break
		}
	}
	if got := response.String(); !strings.Contains(got, "Content-Type:") || !strings.Contains(got, "fastcgi-ok") {
		t.Fatalf("FastCGI response = %q", got)
	}
}

func TestServeFastCGIListener(t *testing.T) {
	listener := newMemoryListener()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- ServeFastCGIListener(ctx, listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "fastcgi-listener-ok")
		}), 100*time.Millisecond)
	}()

	conn, err := listener.dial()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	params := fastCGIParams(map[string]string{
		"REQUEST_METHOD": "GET",
		"REQUEST_URI":    "/test",
	})
	request := append(fastCGIRecord(1, 1, []byte{0, 1, 0, 0, 0, 0, 0, 0}), fastCGIRecord(4, 1, params)...)
	request = append(request, fastCGIRecord(4, 1, nil)...)
	request = append(request, fastCGIRecord(5, 1, nil)...)
	if _, err := conn.Write(request); err != nil {
		t.Fatal(err)
	}

	cancel() // trigger graceful shutdown
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("unexpected ServeFastCGIListener error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ServeFastCGIListener did not stop within timeout")
	}
}

func fastCGIRecord(kind byte, requestID uint16, payload []byte) []byte {
	padding := byte((8 - len(payload)%8) % 8)
	result := make([]byte, 8+len(payload)+int(padding))
	result[0], result[1] = 1, kind
	binary.BigEndian.PutUint16(result[2:4], requestID)
	binary.BigEndian.PutUint16(result[4:6], uint16(len(payload)))
	result[6] = padding
	copy(result[8:], payload)
	return result
}

func fastCGIParams(params map[string]string) []byte {
	var result bytes.Buffer
	for name, value := range params {
		if len(name) >= 128 || len(value) >= 128 {
			panic(fmt.Sprintf("test FastCGI param too long: %s", name))
		}
		result.WriteByte(byte(len(name)))
		result.WriteByte(byte(len(value)))
		result.WriteString(name)
		result.WriteString(value)
	}
	return result.Bytes()
}
