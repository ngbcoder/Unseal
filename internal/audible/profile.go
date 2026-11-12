package audible

import (
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/ngbcoder/Unseal/internal/types"
)

// Profile is a registered Audible device.
//
// It holds the same credential groups that the original C# Profile
// carries, so the protocol client can sign requests and refresh tokens.
type Profile struct {
	// Region is the marketplace.
	Region types.Region
	// PreAmazon says whether the legacy Audible username path was used.
	PreAmazon bool
	// Alias is the name the user gave the account.
	Alias string

	// AuthorizationCode is the code Audible returned at sign-in.
	AuthorizationCode string
	// CodeVerifier is the PKCE verifier used at sign-in.
	CodeVerifier string

	// Token carries the bearer access and refresh tokens.
	Token TokenBearer
	// DeviceInfo is the device identity Audible assigned.
	DeviceInfo DeviceInfo
	// CustomerInfo is the customer identity Audible returned.
	CustomerInfo CustomerInfo

	// Cookies are the website cookies Audible set.
	Cookies []Cookie
	// StoreAuthentCookie is the store-authentication cookie.
	StoreAuthentCookie string

	// PrivateKey is the device private key, PEM-encoded.
	PrivateKey string
	// AdpToken is the ADP token used to sign requests.
	AdpToken string
}

// TokenBearer carries the bearer access and refresh tokens.
type TokenBearer struct {
	AccessToken  string
	RefreshToken string
	Expiration   time.Time
}

// DeviceInfo is the device identity Audible assigned.
type DeviceInfo struct {
	Type   string
	Name   string
	Serial string
}

// CustomerInfo is the customer identity Audible returned.
type CustomerInfo struct {
	Name      string
	AccountID string
}

// Cookie is one cookie Audible set.
type Cookie struct {
	Name     string
	Value    string
	Domain   string
	Path     string
	Expires  string
	Secure   bool
	HttpOnly bool
}

// IsValid reports whether the access token is still valid for more than
// five minutes.
func (p *Profile) IsValid() bool {
	if p == nil || p.Token.AccessToken == "" {
		return false
	}
	return time.Now().Add(5 * time.Minute).Before(p.Token.Expiration)
}

// AliasKey returns the profile's alias key.
func (p *Profile) AliasKey() types.ProfileAliasKey {
	return types.ProfileAliasKey{
		Region:       p.Region,
		AccountAlias: p.Alias,
	}
}

// HTTPClient is the transport used to talk to Audible.
//
// It carries an explicit cookie jar, disables auto-redirect and enables
// gzip/deflate decompression, matching the original HttpClientEx.
type HTTPClient struct {
	client *http.Client
	jar    http.CookieJar
}

// NewHTTPClient creates a client.
func NewHTTPClient() (*HTTPClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &HTTPClient{
		jar: jar,
		client: &http.Client{
			Jar: jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Auto-redirect is disabled deliberately: the Amazon sign-in
				// flow relies on the user copying the final URL.
				return http.ErrUseLastResponse
			},
			Transport: &http.Transport{
				DisableCompression: false,
			},
		},
	}, nil
}

// Do sends a request.
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

// Jar returns the cookie jar.
func (c *HTTPClient) Jar() http.CookieJar { return c.jar }
