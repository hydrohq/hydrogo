package main

import (
	"strings"
	"testing"
)

func TestValidateDemoListenAddress(t *testing.T) {
	for _, address := range []string{"127.0.0.1:8091", "[::1]:8091", "localhost:8091", "LOCALHOST:8091"} {
		t.Run(address, func(t *testing.T) {
			if err := validateDemoListenAddress(address); err != nil {
				t.Fatalf("loopback address rejected: %v", err)
			}
		})
	}
	for _, address := range []string{":8091", "0.0.0.0:8091", "192.0.2.10:8091", "example.test:8091", "invalid"} {
		t.Run(address, func(t *testing.T) {
			if err := validateDemoListenAddress(address); err == nil {
				t.Fatal("non-loopback or invalid address was accepted")
			}
		})
	}
}

func TestServeRejectsPublicBindBeforeOpeningDatabase(t *testing.T) {
	err := serve([]string{"--address", "0.0.0.0:8091"})
	if err == nil || !strings.Contains(err.Error(), "may listen only on loopback") {
		t.Fatalf("public demo server bind was not rejected: %v", err)
	}
}
