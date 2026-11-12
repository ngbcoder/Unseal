package audible_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/ngbcoder/Unseal/internal/audible"
)

// testKey is a small RSA key used only by these tests.
var testKey *rsa.PrivateKey

func init() {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	testKey = key
}

func TestSignProducesTheExpectedShape(t *testing.T) {
	sig, err := audible.Sign("GET", "/1.0/library", "", "adp-token", testKey, time.Date(2026, 11, 11, 22, 54, 19, 0, time.UTC))
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// The signature is base64(sig):<time>.
	parts := splitColon(sig)
	if len(parts) != 2 {
		t.Fatalf("the signature is %q, want base64:time", sig)
	}

	// The time is rounded down to a ten-minute boundary.
	if parts[1] != "2026-11-11T22:50:00Z" {
		t.Errorf("the time is %q, want %q", parts[1], "2026-11-11T22:50:00Z")
	}

	// The base64 part must decode.
	if _, err := base64Decode(parts[0]); err != nil {
		t.Errorf("the signature is not valid base64: %v", err)
	}
}

func TestSignRoundsTheTimeDown(t *testing.T) {
	sig, err := audible.Sign("GET", "/1.0/library", "", "adp-token", testKey, time.Date(2026, 11, 11, 22, 59, 59, 0, time.UTC))
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if !stringsContains(sig, "2026-11-11T22:50:00Z") {
		t.Errorf("the time was not rounded down: %q", sig)
	}
}

func TestSignIncludesTheBodyInDataString(t *testing.T) {
	// Two requests that differ only in body must produce different signatures.
	a, err := audible.Sign("POST", "/1.0/content/B0/licenserequest", `{"quality":"High"}`, "adp-token", testKey, time.Date(2026, 11, 11, 22, 50, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	b, err := audible.Sign("POST", "/1.0/content/B0/licenserequest", `{"quality":"Low"}`, "adp-token", testKey, time.Date(2026, 11, 11, 22, 50, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if a == b {
		t.Error("the signatures must differ when the body differs")
	}
}

func TestSignRejectsANilKey(t *testing.T) {
	if _, err := audible.Sign("GET", "/1.0/library", "", "adp-token", nil, time.Time{}); err == nil {
		t.Error("Sign must reject a nil private key")
	}
}

func TestParsePrivateKeyRoundTrips(t *testing.T) {
	pemText := marshalPrivateKey(testKey)

	parsed, err := audible.ParsePrivateKey(pemText)
	if err != nil {
		t.Fatalf("ParsePrivateKey failed: %v", err)
	}
	if parsed == nil {
		t.Fatal("the parsed key is nil")
	}

	// The parsed key must produce the same signature.
	a, err := audible.Sign("GET", "/1.0/library", "", "adp", testKey, time.Date(2026, 11, 11, 22, 50, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	b, err := audible.Sign("GET", "/1.0/library", "", "adp", parsed, time.Date(2026, 11, 11, 22, 50, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if a != b {
		t.Error("the parsed key must produce the same signature")
	}
}

// helpers

func splitColon(s string) []string {
	return strings.SplitN(s, ":", 2)
}

func base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func marshalPrivateKey(key *rsa.PrivateKey) string {
	data := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: data}
	var buf bytes.Buffer
	if err := pem.Encode(&buf, block); err != nil {
		panic(err)
	}
	return buf.String()
}
