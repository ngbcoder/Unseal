package audible

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/url"
)

// Login holds the data produced by the sign-in flow.
type Login struct {
	// Serial is the 40-character device serial.
	Serial string
	// ClientID is the hex encoding of "<serial>#A2CZJZGLK2JJVM".
	ClientID string
	// CodeVerifier is the PKCE code verifier.
	CodeVerifier string
	// CodeChallenge is the PKCE code challenge.
	CodeChallenge string
	// Region is the marketplace the login targets.
	Region string
	// UsePreAmazonLogin says whether the legacy Audible username path is
	// used instead of the Amazon one.
	UsePreAmazonLogin bool
}

// NewLogin builds a login for the given region.
//
// preAmazon selects the legacy Audible username path, which is only
// allowed for de, uk and us.
func NewLogin(region string, preAmazon bool) Login {
	serial := BuildDeviceSerial()
	login := Login{
		Serial:            serial,
		ClientID:          BuildClientID(serial),
		CodeVerifier:      CreateCodeVerifier(),
		Region:            region,
		UsePreAmazonLogin: preAmazon && LocaleFor(region).SupportsPreAmazonLogin,
	}
	login.CodeChallenge = CreateSHA256CodeChallenge(login.CodeVerifier)
	return login
}

// BuildDeviceSerial generates a 40-character serial from the alphabet
// A-Z a-z 0-9, using an unbiased source of randomness.
func BuildDeviceSerial() string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	const length = 40

	out := make([]byte, length)
	for i := range out {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return ""
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out)
}

// BuildClientID builds the client id from the device serial.
func BuildClientID(serial string) string {
	return hex.EncodeToString([]byte(serial + "#A2CZJZGLK2JJVM"))
}

// CreateCodeVerifier generates a PKCE code verifier.
func CreateCodeVerifier() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

// CreateSHA256CodeChallenge builds the PKCE code challenge from a verifier.
func CreateSHA256CodeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// BuildAuthURI assembles the Amazon or Audible sign-in URL.
//
// returnTo is the redirect URL the application is listening on, or empty
// when the user will copy the final URL out of the browser by hand.
func BuildAuthURI(login Login, returnTo string) string {
	host := AmazonWebHost(login.Region)
	if login.UsePreAmazonLogin {
		host = AudibleWebHost(login.Region)
	}

	loc := LocaleFor(login.Region)

	q := url.Values{}
	q.Set("openid.oa2.response_type", "code")
	q.Set("openid.oa2.code_challenge", login.CodeChallenge)
	q.Set("openid.oa2.code_challenge_method", "S256")
	q.Set("openid.oa2.client_id", "device:"+login.ClientID)
	q.Set("openid.oa2.scope", "device_auth_access")
	q.Set("openid.assoc_handle", "amzn_audible_ios_"+login.Region)
	q.Set("pageId", "amzn_audible_ios")
	q.Set("marketPlaceId", loc.MarketplaceID)
	q.Set("openid.identity", "http://specs.openid.net/auth/2.0/identifier_select")
	q.Set("openid.claimed_id", "http://specs.openid.net/auth/2.0/identifier_select")
	if returnTo != "" {
		q.Set("openid.return_to", returnTo)
	}

	return host + "?" + q.Encode()
}

// ParseRedirectURL extracts the authorisation code from the redirected URL.
func ParseRedirectURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	code := u.Query().Get("openid.oa2.authorization_code")
	if code == "" {
		return "", fmt.Errorf("audible: no authorisation code in the redirect URL")
	}
	return code, nil
}
