package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestBuildIsDeterministicAndFingerprintsDeclaredAssets(t *testing.T) {
	first, err := build("../../themes")
	if err != nil {
		t.Fatal(err)
	}
	second, err := build("../../themes")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("asset manifest changed between identical builds")
	}
	if first["newsroom"].Assets["theme.css"].SHA256 == "" {
		t.Fatal("theme.css fingerprint is empty")
	}
	if _, err := json.Marshal(first); err != nil {
		t.Fatalf("manifest is not serializable: %v", err)
	}
}
