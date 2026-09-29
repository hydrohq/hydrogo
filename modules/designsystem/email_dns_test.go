package designsystem

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestEmailDNSChecksEscapesUntrustedGuidance(t *testing.T) {
	var output bytes.Buffer
	err := EmailDNSChecks([]EmailDNSCheck{{Name: "SPF", Guidance: `<script>alert("x")</script>`}}).Render(context.Background(), &output)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), `<script>`) || !strings.Contains(output.String(), `&lt;script&gt;`) {
		t.Fatalf("guidance not escaped: %s", output.String())
	}
}
