package media

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestS3ProviderIntegration is intentionally opt-in. It exercises the actual
// configured S3-compatible service, not a mock HTTP implementation.
func TestS3ProviderIntegration(t *testing.T) {
	endpoint := strings.TrimSpace(os.Getenv("HYDRO_TEST_S3_ENDPOINT"))
	accessKey := os.Getenv("HYDRO_TEST_S3_ACCESS_KEY")
	secretKey := os.Getenv("HYDRO_TEST_S3_SECRET_KEY")
	bucket := strings.TrimSpace(os.Getenv("HYDRO_TEST_S3_BUCKET"))
	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		t.Skip("set HYDRO_TEST_S3_ENDPOINT, HYDRO_TEST_S3_ACCESS_KEY, HYDRO_TEST_S3_SECRET_KEY, and HYDRO_TEST_S3_BUCKET for a live provider drill")
	}
	store, err := NewS3BlobStore(S3Config{
		Endpoint: endpoint, AccessKey: accessKey, SecretKey: secretKey, Bucket: bucket,
		Region: os.Getenv("HYDRO_TEST_S3_REGION"), UseSSL: os.Getenv("HYDRO_TEST_S3_USE_SSL") == "1",
		ForcePathStyle: os.Getenv("HYDRO_TEST_S3_FORCE_PATH_STYLE") == "1",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatal(err)
	}
	key := fmt.Sprintf("quarantine/integration-%x/object.txt", suffix)
	body := []byte("hydrogo-live-provider-drill")
	if err := store.Put(ctx, key, strings.NewReader(string(body)), int64(len(body)), "text/plain"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Delete(context.Background(), key) }()

	object, err := store.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	got, readErr := io.ReadAll(object)
	closeErr := object.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	if string(got) != string(body) {
		t.Fatalf("provider round-trip body = %q, want %q", got, body)
	}

	quarantineKey := fmt.Sprintf("tenants/integration/quarantine-%x/direct.txt", suffix)
	defer func() { _ = store.Delete(context.Background(), quarantineKey) }()
	putURL, err := store.PresignedQuarantineUploadURL(ctx, quarantineKey, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	putRequest, err := http.NewRequestWithContext(ctx, http.MethodPut, putURL, strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	putRequest.Header.Set("Content-Type", "text/plain")
	putResponse, err := http.DefaultClient.Do(putRequest)
	if err != nil {
		t.Fatal(err)
	}
	putResponse.Body.Close()
	if putResponse.StatusCode < 200 || putResponse.StatusCode >= 300 {
		t.Fatalf("presigned quarantine PUT status = %d", putResponse.StatusCode)
	}
	quarantineObject, err := store.Get(ctx, quarantineKey)
	if err != nil {
		t.Fatal(err)
	}
	quarantineBytes, readErr := io.ReadAll(quarantineObject)
	closeErr = quarantineObject.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	if string(quarantineBytes) != string(body) {
		t.Fatalf("presigned quarantine body = %q, want %q", quarantineBytes, body)
	}

	getURL, err := store.PresignedURL(ctx, key, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.Get(getURL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("presigned GET status = %d, want 200", response.StatusCode)
	}
}
