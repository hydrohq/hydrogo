package licensing

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Default errors for licensing.
var (
	ErrInvalidSignature = errors.New("license signature verification failed")
	ErrLicenseExpired   = errors.New("license has expired and exceeded grace period")
	ErrProjectMismatch  = errors.New("license project ID does not match")
	ErrEditionMismatch  = errors.New("license edition does not meet requirement")
	ErrFeatureMissing   = errors.New("required feature is not permitted by license")
	ErrKeyIDMismatch    = errors.New("license key ID does not match trusted key slot")
	ErrMalformedLicense = errors.New("malformed license document")
	ErrInvalidKey       = errors.New("invalid Ed25519 cryptographic key")
)

// TrustedPublicKeyBase64 is injected into official runtime builds with
// -ldflags. It is intentionally empty in the development source build.
var TrustedPublicKeyBase64 string

// TrustedPublicKeysBase64 optionally contains a comma-separated list of
// vendor public keys for a short key-rotation overlap window. The legacy
// TrustedPublicKeyBase64 value remains supported for single-key builds.
var TrustedPublicKeysBase64 string

// LicenseState represents the runtime operational status of a license.
type LicenseState string

const (
	StateActive      LicenseState = "active"
	StateGracePeriod LicenseState = "grace_period"
	StateExpired     LicenseState = "expired"
	StateInvalid     LicenseState = "invalid"
)

// Claims contains the payload data of an offline license.
type Claims struct {
	ID              string            `json:"id"`
	KeyID           string            `json:"key_id,omitempty"`
	Customer        string            `json:"customer"`
	ProjectID       string            `json:"project_id"`
	Edition         string            `json:"edition"`
	Features        []string          `json:"features"`
	IssuedAt        time.Time         `json:"issued_at"`
	ExpiresAt       time.Time         `json:"expires_at"`
	GracePeriodDays int               `json:"grace_period_days"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// SignedLicense is the serializable offline license structure.
type SignedLicense struct {
	Claims    json.RawMessage `json:"claims"`
	PublicKey string          `json:"public_key"`
	Signature string          `json:"signature"`
}

// VerificationOptions defines requirements when validating a license at runtime.
type VerificationOptions struct {
	CurrentTime      time.Time
	ExpectedProject  string
	RequiredEdition  string
	RequiredFeatures []string
	TrustedPublicKey ed25519.PublicKey
	ExpectedKeyID    string
}

// VerificationResult holds the status, claims, and human-readable summary.
type VerificationResult struct {
	State         LicenseState `json:"state"`
	Claims        Claims       `json:"claims"`
	DaysLeft      int          `json:"days_left"`
	GraceDaysLeft int          `json:"grace_days_left,omitempty"`
	Message       string       `json:"message"`
}

// GenerateKeyPair produces a new Ed25519 key pair for license issuing and verification.
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key pair: %w", err)
	}
	return pub, priv, nil
}

// Issue creates and signs a new offline license with the provided private key.
func Issue(claims Claims, privateKey ed25519.PrivateKey) ([]byte, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, ErrInvalidKey
	}
	if strings.TrimSpace(claims.ID) == "" {
		idBytes := make([]byte, 16)
		if _, err := rand.Read(idBytes); err != nil {
			return nil, fmt.Errorf("generate license ID: %w", err)
		}
		claims.ID = "lic-" + hex.EncodeToString(idBytes)
	}
	if strings.TrimSpace(claims.Customer) == "" {
		return nil, errors.New("customer name is required")
	}
	if strings.TrimSpace(claims.ProjectID) == "" {
		return nil, errors.New("project_id is required")
	}
	if strings.TrimSpace(claims.Edition) == "" {
		claims.Edition = "starter"
	}
	if claims.IssuedAt.IsZero() {
		claims.IssuedAt = time.Now().UTC()
	}
	if claims.ExpiresAt.IsZero() {
		return nil, errors.New("expires_at date is required")
	}
	if claims.ExpiresAt.Before(claims.IssuedAt) {
		return nil, errors.New("expires_at cannot be before issued_at")
	}
	if claims.GracePeriodDays < 0 {
		claims.GracePeriodDays = 0
	}

	// Canonicalize features
	featureSet := make(map[string]struct{})
	for _, f := range claims.Features {
		f = strings.TrimSpace(f)
		if f != "" {
			featureSet[f] = struct{}{}
		}
	}
	sortedFeatures := make([]string, 0, len(featureSet))
	for f := range featureSet {
		sortedFeatures = append(sortedFeatures, f)
	}
	sort.Strings(sortedFeatures)
	claims.Features = sortedFeatures

	canonicalClaims, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical license claims: %w", err)
	}

	pubKey := privateKey.Public().(ed25519.PublicKey)
	signature := ed25519.Sign(privateKey, canonicalClaims)

	signed := SignedLicense{
		Claims:    canonicalClaims,
		PublicKey: base64.RawStdEncoding.EncodeToString(pubKey),
		Signature: base64.RawStdEncoding.EncodeToString(signature),
	}

	output, err := json.MarshalIndent(signed, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal signed license: %w", err)
	}
	return append(output, '\n'), nil
}

// Verify decodes, verifies the cryptographic signature, and checks business constraints.
func Verify(rawLicense []byte, opts VerificationOptions) (*VerificationResult, error) {
	if len(rawLicense) == 0 {
		return nil, fmt.Errorf("%w: empty license data", ErrMalformedLicense)
	}

	var signed SignedLicense
	if err := json.Unmarshal(rawLicense, &signed); err != nil {
		return nil, fmt.Errorf("%w: json parse failure: %v", ErrMalformedLicense, err)
	}

	if len(signed.Claims) == 0 || signed.Signature == "" {
		return nil, fmt.Errorf("%w: missing claims or signature", ErrMalformedLicense)
	}

	sigBytes, err := base64.RawStdEncoding.DecodeString(signed.Signature)
	if err != nil {
		return nil, fmt.Errorf("%w: decode signature: %v", ErrMalformedLicense, err)
	}

	var trustedKeys []ed25519.PublicKey
	if len(opts.TrustedPublicKey) == ed25519.PublicKeySize {
		trustedKeys = append(trustedKeys, opts.TrustedPublicKey)
	}
	if len(trustedKeys) == 0 && TrustedPublicKeysBase64 != "" {
		for _, encoded := range strings.Split(TrustedPublicKeysBase64, ",") {
			decoded, decodeErr := DecodePublicKey(encoded)
			if decodeErr != nil {
				return nil, fmt.Errorf("trusted vendor public key error: %w", decodeErr)
			}
			trustedKeys = append(trustedKeys, decoded)
		}
	}
	if len(trustedKeys) == 0 && TrustedPublicKeyBase64 != "" {
		decoded, err := DecodePublicKey(TrustedPublicKeyBase64)
		if err != nil {
			return nil, fmt.Errorf("trusted vendor public key error: %w", err)
		}
		trustedKeys = append(trustedKeys, decoded)
	}
	if len(trustedKeys) == 0 {
		return nil, errors.New("trusted vendor public key is required; self-signed public keys in license file are rejected (güvenilir satıcı açık anahtarı gereklidir)")
	}

	// Reject self-signed licenses where advertised public key does not match trusted key
	var pubKey ed25519.PublicKey
	if signed.PublicKey != "" {
		advertisedKey, err := base64.RawStdEncoding.DecodeString(signed.PublicKey)
		if err == nil && len(advertisedKey) == ed25519.PublicKeySize {
			for _, trusted := range trustedKeys {
				if trusted.Equal(ed25519.PublicKey(advertisedKey)) {
					pubKey = trusted
					break
				}
			}
			if len(pubKey) == 0 {
				return &VerificationResult{
					State:   StateInvalid,
					Message: "Yetkisiz veya sahte lisans anahtarı: lisanstaki açık anahtar resmi satıcı anahtarıyla eşleşmiyor (unauthorized vendor key).",
				}, ErrInvalidSignature
			}
		}
	}

	var claims Claims
	if err := json.Unmarshal(signed.Claims, &claims); err != nil {
		return nil, fmt.Errorf("%w: parse claims: %v", ErrMalformedLicense, err)
	}

	canonicalClaims, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("%w: canonicalize claims: %v", ErrMalformedLicense, err)
	}

	if len(pubKey) > 0 && !ed25519.Verify(pubKey, canonicalClaims, sigBytes) {
		pubKey = nil
	}
	if len(pubKey) == 0 {
		for _, trusted := range trustedKeys {
			if ed25519.Verify(trusted, canonicalClaims, sigBytes) {
				pubKey = trusted
				break
			}
		}
	}
	if len(pubKey) == 0 {
		return &VerificationResult{
			State:   StateInvalid,
			Message: "Geçersiz lisans imzası: dosya içeriği değiştirilmiş veya imza anahtarı uyuşmuyor.",
		}, ErrInvalidSignature
	}

	now := opts.CurrentTime
	if now.IsZero() {
		now = time.Now().UTC()
	}

	result := &VerificationResult{
		Claims: claims,
	}

	// Project ID check
	if opts.ExpectedProject != "" && !strings.EqualFold(claims.ProjectID, opts.ExpectedProject) {
		result.State = StateInvalid
		result.Message = fmt.Sprintf("Proje kimliği uyuşmazlığı: beklenen %q, lisanstaki %q.", opts.ExpectedProject, claims.ProjectID)
		return result, fmt.Errorf("%w (expected: %s, license: %s)", ErrProjectMismatch, opts.ExpectedProject, claims.ProjectID)
	}
	if opts.ExpectedKeyID != "" && !strings.EqualFold(strings.TrimSpace(claims.KeyID), strings.TrimSpace(opts.ExpectedKeyID)) {
		result.State = StateInvalid
		result.Message = fmt.Sprintf("Lisans anahtar yuvası uyuşmazlığı: beklenen %q, lisanstaki %q.", opts.ExpectedKeyID, claims.KeyID)
		return result, fmt.Errorf("%w (expected: %s, license: %s)", ErrKeyIDMismatch, opts.ExpectedKeyID, claims.KeyID)
	}

	// Edition check
	if opts.RequiredEdition != "" && !isEditionSatisfied(claims.Edition, opts.RequiredEdition) {
		result.State = StateInvalid
		result.Message = fmt.Sprintf("Lisans sürüm uyuşmazlığı: gereken %q, lisanstaki %q.", opts.RequiredEdition, claims.Edition)
		return result, fmt.Errorf("%w (required: %s, license: %s)", ErrEditionMismatch, opts.RequiredEdition, claims.Edition)
	}

	// Feature check
	licenseFeatures := make(map[string]bool)
	for _, f := range claims.Features {
		licenseFeatures[strings.ToLower(f)] = true
	}
	// "all" or "*" grants everything
	hasAllFeatures := licenseFeatures["*"] || licenseFeatures["all"]
	for _, reqFeature := range opts.RequiredFeatures {
		if !hasAllFeatures && !licenseFeatures[strings.ToLower(reqFeature)] {
			result.State = StateInvalid
			result.Message = fmt.Sprintf("Gereken özellik lisans kapsamında değil: %q.", reqFeature)
			return result, fmt.Errorf("%w: %s", ErrFeatureMissing, reqFeature)
		}
	}

	// Expiry & Grace period check
	graceEnd := claims.ExpiresAt.AddDate(0, 0, claims.GracePeriodDays)

	if now.Before(claims.ExpiresAt) {
		result.State = StateActive
		daysLeft := int(claims.ExpiresAt.Sub(now).Hours() / 24)
		if daysLeft < 0 {
			daysLeft = 0
		}
		result.DaysLeft = daysLeft
		result.Message = fmt.Sprintf("Lisans geçerli ve aktif. Kalan süre: %d gün (Bitiş: %s).", daysLeft, claims.ExpiresAt.Format("2006-01-02"))
		return result, nil
	}

	if now.Before(graceEnd) || now.Equal(graceEnd) {
		result.State = StateGracePeriod
		graceDaysLeft := int(graceEnd.Sub(now).Hours() / 24)
		if graceDaysLeft < 0 {
			graceDaysLeft = 0
		}
		result.GraceDaysLeft = graceDaysLeft
		result.Message = fmt.Sprintf("DİKKAT: Lisans süresi %s tarihinde doldu! Tolerans (grace period) modunda çalışıyor. Kalan ek süre: %d gün. Lütfen lisansınızı yenileyin.", claims.ExpiresAt.Format("2006-01-02"), graceDaysLeft)
		return result, nil
	}

	result.State = StateExpired
	result.Message = fmt.Sprintf("Lisans süresi doldu. %s tarihinde sona erdi ve %d günlük tolerans süresi aşıldı.", claims.ExpiresAt.Format("2006-01-02"), claims.GracePeriodDays)
	return result, ErrLicenseExpired
}

// EncodePublicKey converts an Ed25519 public key to base64 format.
func EncodePublicKey(pubKey ed25519.PublicKey) string {
	return base64.RawStdEncoding.EncodeToString(pubKey)
}

// DecodePublicKey parses an Ed25519 public key from base64 format.
func DecodePublicKey(encoded string) (ed25519.PublicKey, error) {
	keyBytes, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, ErrInvalidKey
	}
	return ed25519.PublicKey(keyBytes), nil
}

// EncodePrivateKey converts an Ed25519 private key to base64 format.
func EncodePrivateKey(privKey ed25519.PrivateKey) string {
	return base64.RawStdEncoding.EncodeToString(privKey)
}

// DecodePrivateKey parses an Ed25519 private key from base64 format.
func DecodePrivateKey(encoded string) (ed25519.PrivateKey, error) {
	keyBytes, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}
	if len(keyBytes) != ed25519.PrivateKeySize {
		return nil, ErrInvalidKey
	}
	return ed25519.PrivateKey(keyBytes), nil
}

// Fingerprint returns a truncated SHA-256 fingerprint for a public key.
func Fingerprint(pubKey ed25519.PublicKey) string {
	hash := sha256.Sum256(pubKey)
	return hex.EncodeToString(hash[:8])
}

// editionHierarchy defines the tier weights.
var editionHierarchy = map[string]int{
	"starter":    10,
	"growth":     20,
	"enterprise": 30,
	"custom":     30,
}

func isEditionSatisfied(licensedEdition, requiredEdition string) bool {
	licensedTier, ok1 := editionHierarchy[strings.ToLower(licensedEdition)]
	requiredTier, ok2 := editionHierarchy[strings.ToLower(requiredEdition)]
	if ok1 && ok2 {
		return licensedTier >= requiredTier
	}
	return strings.EqualFold(licensedEdition, requiredEdition)
}
