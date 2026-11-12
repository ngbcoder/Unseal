package audible

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/ngbcoder/Unseal/internal/util"
)

// Sign signs a request's data string with the device private key.
//
// The data string is:
//
//	<METHOD>\n<relative URL>\n<time>\n<request body or empty>\n<ADP token>
//
// The time is the current UTC time rounded down to a ten-minute boundary,
// formatted as an XML date-time. The signature is RSA-SHA256 over the
// SHA-256 digest of the data string, PKCS#1 v1.5 padding, base64-encoded,
// followed by a colon and the time.
func Sign(method, relativeURL, body, adpToken string, privateKey *rsa.PrivateKey, now time.Time) (string, error) {
	if privateKey == nil {
		return "", fmt.Errorf("audible: no private key to sign with")
	}

	bucket := now.UTC().Truncate(10 * time.Minute)
	timeStr := util.ToXmlTime(bucket)

	data := method + "\n" + relativeURL + "\n" + timeStr + "\n" + body + "\n" + adpToken

	digest := sha256.Sum256([]byte(data))
	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("audible: cannot sign the request: %w", err)
	}

	return base64.StdEncoding.EncodeToString(sig) + ":" + timeStr, nil
}

// ParsePrivateKey imports a PEM-encoded RSA private key.
//
// Audible returns the key as a PEM-encoded PKCS#1 or PKCS#8 private key.
func ParsePrivateKey(pemText string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemText))
	if block == nil {
		return nil, fmt.Errorf("audible: no PEM block in the private key")
	}

	// Try PKCS#1 first, which is what Audible sends.
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	// Fall back to PKCS#8.
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("audible: cannot parse the private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("audible: the private key is not an RSA key")
	}
	return key, nil
}
