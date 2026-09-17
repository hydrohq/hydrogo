// Command hydro-issuer contains vendor-only commercial license operations.
// It must not be distributed with customer runtime builds. The main hydro
// binary intentionally exposes verification and status only.
package main

import (
	"crypto/ed25519"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/licensing"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "hydro-issuer:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("command requires a subcommand (issue, keygen)")
	}
	switch args[0] {
	case "issue":
		return issueCommand(args[1:], stdout, stderr)
	case "keygen":
		return keygenCommand(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown subcommand %q (supported: issue, keygen)", args[0])
	}
}

func issueCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("issue", flag.ContinueOnError)
	flags.SetOutput(stderr)
	customer := flags.String("customer", "", "Customer / organization name (required)")
	projectID := flags.String("project-id", "", "Project ID (required)")
	edition := flags.String("edition", "starter", "License edition (starter, growth, enterprise, custom)")
	keyID := flags.String("key-id", "", "Vendor signing key slot identifier")
	features := flags.String("features", "", "Comma-separated list of enabled features")
	expiresIn := flags.String("expires-in", "8760h", "License duration from now")
	expiresAtStr := flags.String("expires-at", "", "Explicit expiry date in RFC3339 format")
	graceDays := flags.Int("grace-days", 14, "Grace period in days after expiration")
	privateKeyPath := flags.String("private-key", "", "Path to Ed25519 private key file or base64 string (required)")
	outPath := flags.String("out", "", "Output path for the signed license file (default: stdout)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if strings.TrimSpace(*customer) == "" || strings.TrimSpace(*projectID) == "" || strings.TrimSpace(*privateKeyPath) == "" {
		return errors.New("issue requires --customer, --project-id, and --private-key")
	}
	privKey, err := loadPrivateKey(*privateKeyPath)
	if err != nil {
		return fmt.Errorf("load private key: %w", err)
	}
	now := time.Now().UTC()
	var expiresAt time.Time
	if *expiresAtStr != "" {
		expiresAt, err = time.Parse(time.RFC3339, *expiresAtStr)
		if err != nil {
			return fmt.Errorf("parse --expires-at: %w", err)
		}
		expiresAt = expiresAt.UTC()
	} else {
		duration, parseErr := time.ParseDuration(*expiresIn)
		if parseErr != nil {
			return fmt.Errorf("parse --expires-in: %w", parseErr)
		}
		expiresAt = now.Add(duration)
	}
	var featureList []string
	for _, feature := range strings.Split(*features, ",") {
		if feature = strings.TrimSpace(feature); feature != "" {
			featureList = append(featureList, feature)
		}
	}
	claims := licensing.Claims{Customer: *customer, KeyID: strings.TrimSpace(*keyID), ProjectID: *projectID, Edition: *edition, Features: featureList, IssuedAt: now, ExpiresAt: expiresAt, GracePeriodDays: *graceDays}
	signedBytes, err := licensing.Issue(claims, privKey)
	if err != nil {
		return fmt.Errorf("issue license: %w", err)
	}
	if *outPath == "" {
		_, err = stdout.Write(signedBytes)
		return err
	}
	if dir := filepath.Dir(*outPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create license output dir: %w", err)
		}
	}
	if err := os.WriteFile(*outPath, signedBytes, 0o644); err != nil {
		return fmt.Errorf("write license file: %w", err)
	}
	_, err = fmt.Fprintf(stdout, "license-issued out=%s customer=%s project_id=%s edition=%s expires=%s grace_days=%d\n", *outPath, claims.Customer, claims.ProjectID, claims.Edition, claims.ExpiresAt.Format("2006-01-02"), claims.GracePeriodDays)
	return err
}

func keygenCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("keygen", flag.ContinueOnError)
	flags.SetOutput(stderr)
	pubOut := flags.String("public-out", "", "Output path for public key (base64)")
	privOut := flags.String("private-out", "", "Output path for private key (base64)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	pubKey, privKey, err := licensing.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate key pair: %w", err)
	}
	pubB64, privB64 := licensing.EncodePublicKey(pubKey), licensing.EncodePrivateKey(privKey)
	if *pubOut != "" {
		if err := os.WriteFile(*pubOut, []byte(pubB64+"\n"), 0o644); err != nil {
			return fmt.Errorf("write public key file: %w", err)
		}
	}
	if *privOut != "" {
		if err := os.WriteFile(*privOut, []byte(privB64+"\n"), 0o600); err != nil {
			return fmt.Errorf("write private key file: %w", err)
		}
	}
	if *pubOut != "" && *privOut != "" {
		_, err = fmt.Fprintf(stdout, "keys-generated public=%s private=%s fingerprint=%s\n", *pubOut, *privOut, licensing.Fingerprint(pubKey))
		return err
	}
	_, err = fmt.Fprintf(stdout, "Ed25519 Key Pair Generated:\nFingerprint: %s\nPublic Key (base64):\n%s\n\nPrivate Key (base64 - KEEP SECRET):\n%s\n", licensing.Fingerprint(pubKey), pubB64, privB64)
	return err
}

func loadPrivateKey(pathOrBase64 string) (ed25519.PrivateKey, error) {
	content := strings.TrimSpace(pathOrBase64)
	if info, err := os.Stat(pathOrBase64); err == nil && !info.IsDir() {
		data, err := os.ReadFile(pathOrBase64)
		if err != nil {
			return nil, err
		}
		content = strings.TrimSpace(string(data))
	}
	return licensing.DecodePrivateKey(content)
}
