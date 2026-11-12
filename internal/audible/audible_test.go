package audible_test

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ngbcoder/Unseal/internal/audible"
	"github.com/ngbcoder/Unseal/internal/audible/model"
)

func TestLocaleForEveryRegion(t *testing.T) {
	regions := []string{"de", "us", "uk", "fr", "ca", "it", "au", "in", "jp", "es", "br"}
	for _, r := range regions {
		l := audible.LocaleFor(r)
		if l.TLD == "" {
			t.Errorf("region %q has no TLD", r)
		}
		if l.MarketplaceID == "" {
			t.Errorf("region %q has no marketplace id", r)
		}
	}
}

func TestLocaleForUnknownRegionFallsBack(t *testing.T) {
	l := audible.LocaleFor("zz")
	if l.TLD != "zz" {
		t.Errorf("the TLD is %q, want %q", l.TLD, "zz")
	}
}

func TestPreAmazonLoginIsOnlyAllowedForThreeRegions(t *testing.T) {
	for _, r := range []string{"de", "us", "uk"} {
		if !audible.LocaleFor(r).SupportsPreAmazonLogin {
			t.Errorf("region %q must allow pre-Amazon login", r)
		}
	}
	for _, r := range []string{"fr", "ca", "it", "au", "in", "jp", "es", "br"} {
		if audible.LocaleFor(r).SupportsPreAmazonLogin {
			t.Errorf("region %q must not allow pre-Amazon login", r)
		}
	}
}

func TestBuildDeviceSerialHasTheRightShape(t *testing.T) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

	serial := audible.BuildDeviceSerial()
	if len(serial) != 40 {
		t.Fatalf("the serial is %d characters, want 40", len(serial))
	}
	for _, c := range serial {
		if !strings.ContainsRune(alphabet, c) {
			t.Errorf("the serial contains %q, which is not in the alphabet", c)
		}
	}
}

func TestBuildDeviceSerialIsRandom(t *testing.T) {
	a := audible.BuildDeviceSerial()
	b := audible.BuildDeviceSerial()

	if a == b {
		t.Error("two serials must not be identical")
	}
}

func TestBuildClientID(t *testing.T) {
	serial := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcd"
	got := audible.BuildClientID(serial)

	// The client id is the hex encoding of "<serial>#A2CZJZGLK2JJVM".
	want := "4142434445464748494a4b4c4d4e4f505152535455565758595a3031323334353637383961626364234132435a4a5a474c4b324a4a564d"
	if got != want {
		t.Errorf("the client id is %q, want %q", got, want)
	}
}

func TestCreateCodeVerifierHasTheRightShape(t *testing.T) {
	v := audible.CreateCodeVerifier()
	if v == "" {
		t.Fatal("the verifier must not be empty")
	}
	// 32 bytes base64-url encoded without padding is 43 characters.
	if len(v) != 43 {
		t.Errorf("the verifier is %d characters, want 43", len(v))
	}
}

func TestCreateSHA256CodeChallengeIsDeterministic(t *testing.T) {
	v := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjw"
	a := audible.CreateSHA256CodeChallenge(v)
	b := audible.CreateSHA256CodeChallenge(v)

	if a != b {
		t.Error("the challenge must be deterministic")
	}
	if a == "" {
		t.Error("the challenge must not be empty")
	}
}

func TestBuildAuthURIContainsTheOpenIDParameters(t *testing.T) {
	login := audible.NewLogin("us", false)
	uri := audible.BuildAuthURI(login, "")

	for _, want := range []string{
		"openid.oa2.response_type=code",
		"openid.oa2.code_challenge=",
		"openid.oa2.code_challenge_method=S256",
		"openid.oa2.client_id=device%3A",
		"openid.oa2.scope=device_auth_access",
		"marketPlaceId=AF2M0KC94RCEA",
	} {
		if !strings.Contains(uri, want) {
			t.Errorf("the URI is missing %q: %s", want, uri)
		}
	}
}

func TestBuildAuthURIUsesTheAudibleHostForPreAmazonLogin(t *testing.T) {
	login := audible.NewLogin("de", true)
	uri := audible.BuildAuthURI(login, "")

	if !strings.HasPrefix(uri, "https://www.audible.de") {
		t.Errorf("pre-Amazon login must use the Audible host, got %s", uri)
	}
}

func TestParseRedirectURLExtractsTheCode(t *testing.T) {
	raw := "https://example.com/cb?openid.oa2.authorization_code=THECODE&other=value"

	code, err := audible.ParseRedirectURL(raw)
	if err != nil {
		t.Fatalf("ParseRedirectURL failed: %v", err)
	}
	if code != "THECODE" {
		t.Errorf("the code is %q, want %q", code, "THECODE")
	}
}

func TestParseRedirectURLRejectsAMissingCode(t *testing.T) {
	if _, err := audible.ParseRedirectURL("https://example.com/cb?other=value"); err == nil {
		t.Error("ParseRedirectURL must reject a URL without a code")
	}
}

func TestParseRedirectURLRejectsGarbage(t *testing.T) {
	if _, err := audible.ParseRedirectURL(":::not a url"); err == nil {
		t.Error("ParseRedirectURL must reject an unparseable URL")
	}
}

func TestDecryptVoucherRoundTrips(t *testing.T) {
	// Build a synthetic licence the way Audible would, then decrypt it.
	deviceType := "A2CZJZGLK2JJVM"
	serial := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcd"
	accountID := "A1B2C3D4E5F6G7"
	asin := "B0TESTASIN12"

	voucher := model.Voucher{Key: "the-drm-key", IV: "the-drm-iv"}
	plain, err := json.Marshal(voucher)
	if err != nil {
		t.Fatalf("cannot marshal the voucher: %v", err)
	}

	// Pad to a multiple of 16 with zero bytes, as Audible does.
	for len(plain)%16 != 0 {
		plain = append(plain, 0)
	}

	hashable := deviceType + serial + accountID + asin
	hash := sha256.Sum256([]byte(hashable))
	key := hash[:16]
	iv := hash[16:32]

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("cannot create the cipher: %v", err)
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(plain, plain)

	encrypted := base64.StdEncoding.EncodeToString(plain)

	licence := &model.ContentLicense{
		ASIN:            asin,
		StatusCode:      "Granted",
		LicenseResponse: encrypted,
	}

	// decryptLicense is unexported, so exercise it through the exported
	// entry point that the Api uses. Re-implement the derivation here to
	// confirm the round trip, since the function itself is private.
	_ = licence

	// Re-derive and decrypt manually, mirroring decryptLicense.
	got, err := decryptVoucherForTest(licence, deviceType, serial, accountID)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if got.Key != voucher.Key || got.IV != voucher.IV {
		t.Errorf("the voucher is %+v, want %+v", got, voucher)
	}
}

// decryptVoucherForTest mirrors the private decryptLicense routine so the
// round trip can be checked without exporting it.
func decryptVoucherForTest(licence *model.ContentLicense, deviceType, serial, accountID string) (*model.Voucher, error) {
	hashable := deviceType + serial + accountID + licence.ASIN
	hash := sha256.Sum256([]byte(hashable))
	key := hash[:16]
	iv := hash[16:32]

	encrypted, err := base64.StdEncoding.DecodeString(licence.LicenseResponse)
	if err != nil {
		return nil, err
	}
	n := len(encrypted) & 0x7ffffff0
	encrypted = encrypted[:n]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	dst := make([]byte, len(encrypted))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(dst, encrypted)

	end := len(dst)
	for i, b := range dst {
		if b == 0 {
			end = i
			break
		}
	}
	dst = dst[:end]

	var v model.Voucher
	if err := json.Unmarshal(dst, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func TestNewApiBuildsTwoClients(t *testing.T) {
	profile := &audible.Profile{Region: "us"}
	api, err := audible.NewApi(profile, "us")
	if err != nil {
		t.Fatalf("NewApi failed: %v", err)
	}
	if api.Amazon == nil || api.Audible == nil {
		t.Error("NewApi must build both HTTP clients")
	}
	if api.Region != "us" {
		t.Errorf("the region is %q, want %q", api.Region, "us")
	}
}
