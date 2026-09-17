package licensing

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLicenseIssueAndVerify(t *testing.T) {
	pubKey, privKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	issuedAt := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2027, 9, 1, 0, 0, 0, 0, time.UTC)

	claims := Claims{
		Customer:        "Acme Corp",
		ProjectID:       "proj-hydro-001",
		Edition:         "growth",
		Features:        []string{"pages", "news", "commerce", "members"},
		IssuedAt:        issuedAt,
		ExpiresAt:       expiresAt,
		GracePeriodDays: 14,
	}

	licenseBytes, err := Issue(claims, privKey)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	opts := VerificationOptions{
		CurrentTime:      time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
		ExpectedProject:  "proj-hydro-001",
		RequiredEdition:  "growth",
		RequiredFeatures: []string{"pages", "commerce"},
		TrustedPublicKey: pubKey,
	}

	res, err := Verify(licenseBytes, opts)
	if err != nil {
		t.Fatalf("Verify() unexpected error = %v", err)
	}

	if res.State != StateActive {
		t.Errorf("expected StateActive, got %s", res.State)
	}
	if res.Claims.Customer != "Acme Corp" {
		t.Errorf("expected customer Acme Corp, got %s", res.Claims.Customer)
	}
	if res.DaysLeft <= 0 {
		t.Errorf("expected positive days left, got %d", res.DaysLeft)
	}
	if !strings.Contains(res.Message, "Lisans geçerli") {
		t.Errorf("expected active message, got %s", res.Message)
	}
}

func TestLicenseKeyIDRotationSlot(t *testing.T) {
	pubKey, privKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	licenseBytes, err := Issue(Claims{
		Customer: "Acme", KeyID: "vendor-2026-01", ProjectID: "proj-key-rotation",
		Edition: "growth", IssuedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour),
	}, privKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(licenseBytes, VerificationOptions{TrustedPublicKey: pubKey, ExpectedKeyID: "vendor-2026-01"}); err != nil {
		t.Fatalf("matching key ID rejected: %v", err)
	}
	_, err = Verify(licenseBytes, VerificationOptions{TrustedPublicKey: pubKey, ExpectedKeyID: "vendor-2027-01"})
	if !errors.Is(err, ErrKeyIDMismatch) {
		t.Fatalf("expected ErrKeyIDMismatch, got %v", err)
	}
}

func TestLicenseVerificationSupportsRotationOverlap(t *testing.T) {
	oldPub, oldPriv, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	newPub, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	licenseBytes, err := Issue(Claims{Customer: "Acme", KeyID: "vendor-old", ProjectID: "rotation", ExpiresAt: time.Now().UTC().Add(time.Hour)}, oldPriv)
	if err != nil {
		t.Fatal(err)
	}
	previous := TrustedPublicKeysBase64
	defer func() { TrustedPublicKeysBase64 = previous }()
	TrustedPublicKeysBase64 = EncodePublicKey(oldPub) + "," + EncodePublicKey(newPub)
	if _, err := Verify(licenseBytes, VerificationOptions{}); err != nil {
		t.Fatalf("rotation overlap rejected old key: %v", err)
	}
}

func TestLicenseInvalidSignature(t *testing.T) {
	pubKey1, privKey1, _ := GenerateKeyPair()
	pubKey2, _, _ := GenerateKeyPair()

	claims := Claims{
		Customer:  "Customer A",
		ProjectID: "proj-1",
		Edition:   "starter",
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}

	licenseBytes, err := Issue(claims, privKey1)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	// Verify using a different public key
	opts := VerificationOptions{
		TrustedPublicKey: pubKey2,
	}

	_, err = Verify(licenseBytes, opts)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}

	// Tamper with payload
	var signed SignedLicense
	_ = json.Unmarshal(licenseBytes, &signed)
	signed.Claims = []byte(`{"id":"tampered"}`)
	tamperedBytes, _ := json.Marshal(signed)

	_, err = Verify(tamperedBytes, VerificationOptions{TrustedPublicKey: pubKey1})
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature on tampered claims, got %v", err)
	}
}

func TestVerifyNeverTrustsPublicKeyAdvertisedByLicense(t *testing.T) {
	_, privateKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	licenseBytes, err := Issue(Claims{Customer: "A", ProjectID: "project", Edition: "starter", ExpiresAt: time.Now().Add(time.Hour)}, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(licenseBytes, VerificationOptions{ExpectedProject: "project"}); err == nil || !strings.Contains(err.Error(), "trusted") {
		t.Fatalf("self-advertised public key was accepted: %v", err)
	}
}

func TestLicenseExpiryAndGracePeriod(t *testing.T) {
	pubKey, privKey, _ := GenerateKeyPair()

	issuedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	graceDays := 7

	claims := Claims{
		Customer:        "Customer Expiring",
		ProjectID:       "proj-exp",
		Edition:         "starter",
		IssuedAt:        issuedAt,
		ExpiresAt:       expiresAt,
		GracePeriodDays: graceDays,
	}

	licenseBytes, err := Issue(claims, privKey)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	// Case 1: Active
	resActive, err := Verify(licenseBytes, VerificationOptions{
		CurrentTime:      time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		TrustedPublicKey: pubKey,
	})
	if err != nil || resActive.State != StateActive {
		t.Errorf("expected active license, got %v, state=%s", err, resActive.State)
	}

	// Case 2: In Grace Period
	resGrace, err := Verify(licenseBytes, VerificationOptions{
		CurrentTime:      time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC),
		TrustedPublicKey: pubKey,
	})
	if err != nil {
		t.Errorf("grace period should not return hard error, got %v", err)
	}
	if resGrace.State != StateGracePeriod {
		t.Errorf("expected StateGracePeriod, got %s", resGrace.State)
	}
	if !strings.Contains(resGrace.Message, "Tolerans") {
		t.Errorf("expected grace message, got %s", resGrace.Message)
	}

	// Case 3: Fully Expired (after grace period)
	resExpired, err := Verify(licenseBytes, VerificationOptions{
		CurrentTime:      time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
		TrustedPublicKey: pubKey,
	})
	if !errors.Is(err, ErrLicenseExpired) {
		t.Errorf("expected ErrLicenseExpired, got %v", err)
	}
	if resExpired == nil || resExpired.State != StateExpired {
		t.Errorf("expected StateExpired, got %v", resExpired)
	}
	if !strings.Contains(resExpired.Message, "Lisans süresi doldu") {
		t.Errorf("expected expired message, got %s", resExpired.Message)
	}
}

func TestLicenseProjectIDMismatch(t *testing.T) {
	pubKey, privKey, _ := GenerateKeyPair()

	claims := Claims{
		Customer:  "Customer Proj",
		ProjectID: "expected-proj-123",
		Edition:   "starter",
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}

	licenseBytes, err := Issue(claims, privKey)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	opts := VerificationOptions{
		ExpectedProject:  "wrong-proj-456",
		TrustedPublicKey: pubKey,
	}

	res, err := Verify(licenseBytes, opts)
	if !errors.Is(err, ErrProjectMismatch) {
		t.Errorf("expected ErrProjectMismatch, got %v", err)
	}
	if res.State != StateInvalid {
		t.Errorf("expected StateInvalid, got %s", res.State)
	}
	if !strings.Contains(res.Message, "Proje kimliği uyuşmazlığı") {
		t.Errorf("expected project mismatch message, got %s", res.Message)
	}
}

func TestLicenseFeatureMissing(t *testing.T) {
	pubKey, privKey, _ := GenerateKeyPair()

	claims := Claims{
		Customer:  "Customer Feat",
		ProjectID: "proj-feat",
		Edition:   "starter",
		Features:  []string{"pages", "news"},
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}

	licenseBytes, err := Issue(claims, privKey)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	opts := VerificationOptions{
		RequiredFeatures: []string{"pages", "commerce"},
		TrustedPublicKey: pubKey,
	}

	res, err := Verify(licenseBytes, opts)
	if !errors.Is(err, ErrFeatureMissing) {
		t.Errorf("expected ErrFeatureMissing, got %v", err)
	}
	if res.State != StateInvalid {
		t.Errorf("expected StateInvalid, got %s", res.State)
	}
	if !strings.Contains(res.Message, "Gereken özellik lisans kapsamında değil") {
		t.Errorf("expected feature missing message, got %s", res.Message)
	}
}

func TestLicenseEditionHierarchy(t *testing.T) {
	pubKey, privKey, _ := GenerateKeyPair()

	// Enterprise license satisfies growth and starter requirement
	claims := Claims{
		Customer:  "Enterprise Customer",
		ProjectID: "proj-ent",
		Edition:   "enterprise",
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	licenseBytes, _ := Issue(claims, privKey)

	res, err := Verify(licenseBytes, VerificationOptions{
		RequiredEdition:  "growth",
		TrustedPublicKey: pubKey,
	})
	if err != nil || res.State != StateActive {
		t.Errorf("enterprise should satisfy growth requirement, got %v", err)
	}

	// Starter license fails enterprise requirement
	starterClaims := Claims{
		Customer:  "Starter Customer",
		ProjectID: "proj-start",
		Edition:   "starter",
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	starterBytes, _ := Issue(starterClaims, privKey)

	_, err = Verify(starterBytes, VerificationOptions{
		RequiredEdition:  "enterprise",
		TrustedPublicKey: pubKey,
	})
	if !errors.Is(err, ErrEditionMismatch) {
		t.Errorf("starter should not satisfy enterprise, got %v", err)
	}
}

func TestKeyEncodingAndFingerprint(t *testing.T) {
	pubKey, privKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair error = %v", err)
	}

	encodedPub := EncodePublicKey(pubKey)
	decodedPub, err := DecodePublicKey(encodedPub)
	if err != nil {
		t.Fatalf("DecodePublicKey error = %v", err)
	}
	if !decodedPub.Equal(pubKey) {
		t.Error("decoded public key does not match original")
	}

	encodedPriv := EncodePrivateKey(privKey)
	decodedPriv, err := DecodePrivateKey(encodedPriv)
	if err != nil {
		t.Fatalf("DecodePrivateKey error = %v", err)
	}
	if !decodedPriv.Equal(privKey) {
		t.Error("decoded private key does not match original")
	}

	fp := Fingerprint(pubKey)
	if len(fp) != 16 {
		t.Errorf("expected 16 hex characters for fingerprint, got %d (%s)", len(fp), fp)
	}
}

func TestIssueValidationErrors(t *testing.T) {
	_, privKey, _ := GenerateKeyPair()

	// Missing customer
	_, err := Issue(Claims{ProjectID: "p1", ExpiresAt: time.Now().Add(time.Hour)}, privKey)
	if err == nil {
		t.Error("expected error for missing customer")
	}

	// Missing project ID
	_, err = Issue(Claims{Customer: "c1", ExpiresAt: time.Now().Add(time.Hour)}, privKey)
	if err == nil {
		t.Error("expected error for missing project_id")
	}

	// Missing expires_at
	_, err = Issue(Claims{Customer: "c1", ProjectID: "p1"}, privKey)
	if err == nil {
		t.Error("expected error for missing expires_at")
	}

	// Expires before issued
	_, err = Issue(Claims{
		Customer:  "c1",
		ProjectID: "p1",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(-time.Hour),
	}, privKey)
	if err == nil {
		t.Error("expected error for expires_at before issued_at")
	}

	// Invalid private key size
	_, err = Issue(Claims{
		Customer:  "c1",
		ProjectID: "p1",
		ExpiresAt: time.Now().Add(time.Hour),
	}, ed25519.PrivateKey(make([]byte, 10)))
	if !errors.Is(err, ErrInvalidKey) {
		t.Errorf("expected ErrInvalidKey, got %v", err)
	}
}

func TestVerifyMalformedInput(t *testing.T) {
	_, err := Verify(nil, VerificationOptions{})
	if !errors.Is(err, ErrMalformedLicense) {
		t.Errorf("expected ErrMalformedLicense on nil, got %v", err)
	}

	_, err = Verify([]byte(`{not json`), VerificationOptions{})
	if !errors.Is(err, ErrMalformedLicense) {
		t.Errorf("expected ErrMalformedLicense on invalid json, got %v", err)
	}

	_, err = Verify([]byte(`{"claims":null}`), VerificationOptions{})
	if !errors.Is(err, ErrMalformedLicense) {
		t.Errorf("expected ErrMalformedLicense on missing signature, got %v", err)
	}
}
