package main

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/licensing"
)

func licenseCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("license command requires a subcommand (issue, verify, status, keygen)")
	}

	switch args[0] {
	case "issue":
		return licenseIssueCommand(args[1:], stdout, stderr)
	case "verify":
		return licenseVerifyCommand(args[1:], stdout, stderr)
	case "status":
		return licenseStatusCommand(args[1:], stdout, stderr)
	case "keygen":
		return licenseKeygenCommand(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown license subcommand %q (supported: issue, verify, status, keygen)", args[0])
	}
}

func licenseIssueCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("license issue", flag.ContinueOnError)
	flags.SetOutput(stderr)

	customer := flags.String("customer", "", "Customer / organization name (required)")
	projectID := flags.String("project-id", "", "Project ID (required)")
	edition := flags.String("edition", "starter", "License edition (starter, growth, enterprise, custom)")
	features := flags.String("features", "", "Comma-separated list of enabled features (e.g. pages,news,commerce,members)")
	expiresIn := flags.String("expires-in", "8760h", "License duration from now (e.g. 720h, 8760h)")
	expiresAtStr := flags.String("expires-at", "", "Explicit expiry date in RFC3339 format (e.g. 2027-01-01T00:00:00Z)")
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
		return errors.New("license issue requires --customer, --project-id, and --private-key")
	}

	privKey, err := loadPrivateKey(*privateKeyPath)
	if err != nil {
		return fmt.Errorf("load private key: %w", err)
	}

	now := time.Now().UTC()
	var expiresAt time.Time
	if *expiresAtStr != "" {
		t, err := time.Parse(time.RFC3339, *expiresAtStr)
		if err != nil {
			return fmt.Errorf("parse --expires-at: %w", err)
		}
		expiresAt = t.UTC()
	} else {
		duration, err := time.ParseDuration(*expiresIn)
		if err != nil {
			return fmt.Errorf("parse --expires-in: %w", err)
		}
		expiresAt = now.Add(duration)
	}

	var featureList []string
	if *features != "" {
		for _, f := range strings.Split(*features, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				featureList = append(featureList, f)
			}
		}
	}

	claims := licensing.Claims{
		Customer:        *customer,
		ProjectID:       *projectID,
		Edition:         *edition,
		Features:        featureList,
		IssuedAt:        now,
		ExpiresAt:       expiresAt,
		GracePeriodDays: *graceDays,
	}

	signedBytes, err := licensing.Issue(claims, privKey)
	if err != nil {
		return fmt.Errorf("issue license: %w", err)
	}

	if *outPath != "" {
		if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil && filepath.Dir(*outPath) != "." {
			return fmt.Errorf("create license output dir: %w", err)
		}
		if err := os.WriteFile(*outPath, signedBytes, 0o644); err != nil {
			return fmt.Errorf("write license file: %w", err)
		}
		_, err = fmt.Fprintf(stdout, "license-issued out=%s customer=%s project_id=%s edition=%s expires=%s grace_days=%d\n",
			*outPath, claims.Customer, claims.ProjectID, claims.Edition, claims.ExpiresAt.Format("2006-01-02"), claims.GracePeriodDays)
		return err
	}

	_, err = stdout.Write(signedBytes)
	return err
}

func licenseVerifyCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("license verify", flag.ContinueOnError)
	flags.SetOutput(stderr)

	licensePath := flags.String("license", "", "Path to signed license file (required)")
	publicKeyPath := flags.String("public-key", "", "Path to trusted Ed25519 public key file or base64 string")
	projectID := flags.String("project-id", "", "Expected project ID to verify against")
	edition := flags.String("edition", "", "Required edition level (e.g. starter, growth, enterprise)")
	features := flags.String("feature", "", "Required feature(s), comma-separated")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if strings.TrimSpace(*licensePath) == "" {
		return errors.New("license verify requires --license")
	}

	licenseData, err := os.ReadFile(*licensePath)
	if err != nil {
		return fmt.Errorf("read license file: %w", err)
	}

	opts := licensing.VerificationOptions{
		ExpectedProject: *projectID,
		RequiredEdition: *edition,
	}

	if *publicKeyPath != "" {
		pubKey, err := loadPublicKey(*publicKeyPath)
		if err != nil {
			return fmt.Errorf("load public key: %w", err)
		}
		opts.TrustedPublicKey = pubKey
	}

	if *features != "" {
		for _, f := range strings.Split(*features, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				opts.RequiredFeatures = append(opts.RequiredFeatures, f)
			}
		}
	}

	result, err := licensing.Verify(licenseData, opts)
	if err != nil {
		if result != nil && result.Message != "" {
			_, _ = fmt.Fprintf(stderr, "HATA: %s\n", result.Message)
		}
		return fmt.Errorf("license verification failed: %w", err)
	}

	_, err = fmt.Fprintf(stdout, "license-verified state=%s customer=%s project_id=%s edition=%s expires=%s days_left=%d message=%q\n",
		result.State, result.Claims.Customer, result.Claims.ProjectID, result.Claims.Edition, result.Claims.ExpiresAt.Format("2006-01-02"), result.DaysLeft, result.Message)
	return err
}

func licenseStatusCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("license status", flag.ContinueOnError)
	flags.SetOutput(stderr)

	licensePath := flags.String("license", "", "Path to signed license file (required)")
	publicKeyPath := flags.String("public-key", "", "Path to trusted Ed25519 public key file or base64 string")
	projectID := flags.String("project-id", "", "Expected project ID to verify against")
	asJSON := flags.Bool("json", false, "Output verification result in JSON format")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if strings.TrimSpace(*licensePath) == "" {
		return errors.New("license status requires --license")
	}

	licenseData, err := os.ReadFile(*licensePath)
	if err != nil {
		return fmt.Errorf("read license file: %w", err)
	}

	opts := licensing.VerificationOptions{
		ExpectedProject: *projectID,
	}
	if *publicKeyPath != "" {
		pubKey, err := loadPublicKey(*publicKeyPath)
		if err != nil {
			return fmt.Errorf("load public key: %w", err)
		}
		opts.TrustedPublicKey = pubKey
	}

	result, err := licensing.Verify(licenseData, opts)
	// Even if expired or invalid, we still print human-readable status if claims were parsed
	if *asJSON {
		outData, jsonErr := json.MarshalIndent(result, "", "  ")
		if jsonErr != nil {
			return jsonErr
		}
		_, err = fmt.Fprintln(stdout, string(outData))
		return err
	}

	if result == nil {
		return fmt.Errorf("cannot read license status: %w", err)
	}

	stateLabel := strings.ToUpper(string(result.State))
	fmt.Fprintf(stdout, "==================================================\n")
	fmt.Fprintf(stdout, " HydroGo Lisans Durumu: [%s]\n", stateLabel)
	fmt.Fprintf(stdout, "==================================================\n")
	fmt.Fprintf(stdout, " Müşteri:         %s\n", result.Claims.Customer)
	fmt.Fprintf(stdout, " Proje ID:        %s\n", result.Claims.ProjectID)
	fmt.Fprintf(stdout, " Sürüm (Edition): %s\n", result.Claims.Edition)
	fmt.Fprintf(stdout, " Özellikler:      %s\n", strings.Join(result.Claims.Features, ", "))
	fmt.Fprintf(stdout, " Düzenlenme Tarihi: %s\n", result.Claims.IssuedAt.Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(stdout, " Bitiş Tarihi:    %s\n", result.Claims.ExpiresAt.Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(stdout, " Tolerans Süresi: %d gün\n", result.Claims.GracePeriodDays)
	fmt.Fprintf(stdout, " Durum Mesajı:    %s\n", result.Message)
	fmt.Fprintf(stdout, "==================================================\n")

	return err
}

func licenseKeygenCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("license keygen", flag.ContinueOnError)
	flags.SetOutput(stderr)

	pubOut := flags.String("public-out", "", "Output path for public key (base64)")
	privOut := flags.String("private-out", "", "Output path for private key (base64)")

	if err := flags.Parse(args); err != nil {
		return err
	}

	pubKey, privKey, err := licensing.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate key pair: %w", err)
	}

	pubB64 := licensing.EncodePublicKey(pubKey)
	privB64 := licensing.EncodePrivateKey(privKey)
	fp := licensing.Fingerprint(pubKey)

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
		_, err = fmt.Fprintf(stdout, "keys-generated public=%s private=%s fingerprint=%s\n", *pubOut, *privOut, fp)
		return err
	}

	fmt.Fprintf(stdout, "Ed25519 Key Pair Generated:\n")
	fmt.Fprintf(stdout, "Fingerprint: %s\n", fp)
	fmt.Fprintf(stdout, "Public Key (base64):\n%s\n\n", pubB64)
	fmt.Fprintf(stdout, "Private Key (base64 - KEEP SECRET):\n%s\n", privB64)
	return nil
}

func loadPublicKey(pathOrBase64 string) (ed25519.PublicKey, error) {
	content := strings.TrimSpace(pathOrBase64)
	if info, err := os.Stat(pathOrBase64); err == nil && !info.IsDir() {
		data, err := os.ReadFile(pathOrBase64)
		if err != nil {
			return nil, err
		}
		content = strings.TrimSpace(string(data))
	}
	return licensing.DecodePublicKey(content)
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

func validateRuntimeLicense(cfg config.Config, stderr io.Writer) (*licensing.VerificationResult, error) {
	if !cfg.License.Required {
		return nil, nil
	}

	licenseFile := cfg.License.File
	if licenseFile == "" {
		licenseFile = "hydro.lic"
	}

	licenseData, err := os.ReadFile(licenseFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("lisans dosyası bulunamadı (%s): hydro license verify veya geçerli bir ticari lisans gereklidir (license file missing)", licenseFile)
		}
		return nil, fmt.Errorf("lisans dosyası okunamadı (%s): %w", licenseFile, err)
	}

	// The lock file is part of the generated Client Edition identity. When
	// licensing is required, accepting a missing or partially matching lock
	// file would allow the runtime configuration to drift away from the
	// project that was licensed.
	lockPath := filepath.Join(filepath.Dir(licenseFile), "hydrogo.lock")
	lockData, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("hydrogo.lock bulunamadı (%s): lisanslı proje kimliği doğrulanamıyor", lockPath)
		}
		return nil, fmt.Errorf("hydrogo.lock okunamadı (%s): %w", lockPath, err)
	}
	var lock struct {
		ProjectID string   `json:"project_id"`
		Edition   string   `json:"edition"`
		Features  []string `json:"features"`
		Licensing struct {
			Required bool `json:"required"`
		} `json:"licensing"`
	}
	if err := json.Unmarshal(lockData, &lock); err != nil {
		return nil, fmt.Errorf("hydrogo.lock geçersiz JSON: %w", err)
	}
	if strings.TrimSpace(lock.ProjectID) == "" {
		return nil, errors.New("hydrogo.lock proje kimliği boş")
	}
	if cfg.License.ProjectID != "" && !strings.EqualFold(lock.ProjectID, cfg.License.ProjectID) {
		return nil, fmt.Errorf("hydrogo.lock proje kimliği (%s) ile config.yaml lisans proje kimliği (%s) uyuşmuyor", lock.ProjectID, cfg.License.ProjectID)
	}
	if cfg.License.Edition != "" && !strings.EqualFold(lock.Edition, cfg.License.Edition) {
		return nil, fmt.Errorf("hydrogo.lock edition (%s) ile config.yaml lisans edition (%s) uyuşmuyor", lock.Edition, cfg.License.Edition)
	}
	if !lock.Licensing.Required {
		return nil, errors.New("hydrogo.lock lisans zorunluluğunu belirtmiyor")
	}
	lockFeatures := make(map[string]struct{}, len(lock.Features))
	for _, feature := range lock.Features {
		lockFeatures[strings.ToLower(strings.TrimSpace(feature))] = struct{}{}
	}
	for _, requiredFeature := range cfg.License.Features {
		feature := strings.ToLower(strings.TrimSpace(requiredFeature))
		if feature == "" {
			continue
		}
		if _, ok := lockFeatures[feature]; !ok {
			return nil, fmt.Errorf("hydrogo.lock feature %q değerini içermiyor", requiredFeature)
		}
	}

	opts := licensing.VerificationOptions{
		ExpectedProject:  cfg.License.ProjectID,
		RequiredEdition:  cfg.License.Edition,
		RequiredFeatures: cfg.License.Features,
	}

	result, err := licensing.Verify(licenseData, opts)
	if err != nil {
		if result != nil && result.Message != "" {
			fmt.Fprintf(stderr, "LİSANS HATASI: %s\n", result.Message)
		}
		return nil, fmt.Errorf("lisans doğrulaması başarısız oldu (license verification failed): %w", err)
	}

	if result.State == licensing.StateGracePeriod {
		fmt.Fprintf(stderr, "LİSANS UYARISI: %s\n", result.Message)
	}

	return result, nil
}
