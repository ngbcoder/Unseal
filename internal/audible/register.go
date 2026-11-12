package audible

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ngbcoder/Unseal/internal/audible/model"
	"github.com/ngbcoder/Unseal/internal/types"
)

// Register registers a new device and returns the resulting profile.
//
// It performs the registration handshake documented in design.md section
// 9.2: it builds the request body from the login and the authorisation
// code, posts it to /auth/register, and consumes the response into a
// profile.
func Register(ctx context.Context, region string, login Login, authCode string, preAmazon bool) (*Profile, error) {
	client, err := NewHTTPClient()
	if err != nil {
		return nil, err
	}

	host := AmazonHost(region)
	if preAmazon && LocaleFor(region).SupportsPreAmazonLogin {
		host = AudibleHost(region)
	}

	body := map[string]any{
		"requested_token_type": []string{"bearer", "mac_dms", "website_cookies", "store_authentication_cookie"},
		"cookies": map[string]any{
			"website_cookies": []any{},
			"domain":          ".amazon." + LocaleFor(region).TLD,
		},
		"registration_data": map[string]any{
			"domain":           "Device",
			"app_version":      "4.60",
			"device_serial":    login.Serial,
			"device_type":      "A2CZJZGLK2JJVM",
			"device_name":      "%FIRST_NAME%%FIRST_NAME_POSSESSIVE_STRING%%DUPE_STRATEGY_1ST%Audible for iPhone",
			"os_version":       "26.2",
			"software_version": "46000856",
			"device_model":     "iPhone",
			"app_name":         "Audible",
		},
		"auth_data": map[string]any{
			"client_id":          login.ClientID,
			"authorization_code": authCode,
			"code_verifier":      login.CodeVerifier,
			"code_algorithm":     "SHA-256",
			"client_domain":      "DeviceLegacy",
		},
		"requested_extensions": []string{"device_info", "customer_info"},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", host+"/auth/register", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("audible: registration failed: HTTP %d: %s", resp.StatusCode, string(respData))
	}

	var reg model.RegistrationResponse
	if err := json.Unmarshal(respData, &reg); err != nil {
		return nil, fmt.Errorf("audible: cannot parse the registration response: %w", err)
	}

	profile, err := consumeRegistration(region, login, &reg)
	if err != nil {
		return nil, err
	}
	profile.PreAmazon = preAmazon
	return profile, nil
}

// consumeRegistration turns a registration response into a profile.
func consumeRegistration(region string, login Login, reg *model.RegistrationResponse) (*Profile, error) {
	if reg == nil || reg.Response.Success.Tokens.Bearer.AccessToken == "" {
		return nil, fmt.Errorf("audible: the registration response is incomplete")
	}

	succ := reg.Response.Success

	profile := &Profile{
		Region:            types.Region(region),
		AuthorizationCode: login.CodeVerifier,
		CodeVerifier:      login.CodeVerifier,
		DeviceInfo: DeviceInfo{
			Type:   succ.Extensions.DeviceInfo.DeviceType,
			Name:   succ.Extensions.DeviceInfo.DeviceName,
			Serial: succ.Extensions.DeviceInfo.DeviceSerialNumber,
		},
		CustomerInfo: CustomerInfo{
			Name:      succ.Extensions.CustomerInfo.Name,
			AccountID: succ.Extensions.CustomerInfo.UserID,
		},
		Token: TokenBearer{
			AccessToken:  succ.Tokens.Bearer.AccessToken,
			RefreshToken: succ.Tokens.Bearer.RefreshToken,
		},
		PrivateKey:         succ.Tokens.MacDms.DevicePrivateKey,
		AdpToken:           succ.Tokens.MacDms.AdpToken,
		StoreAuthentCookie: succ.Tokens.StoreAuthenticationCookie.Cookie,
	}

	// Parse the token expiration.
	if succ.Tokens.Bearer.ExpiresIn != "" {
		// expires_in is a number of seconds as a string.
		var seconds int
		if _, err := fmt.Sscanf(succ.Tokens.Bearer.ExpiresIn, "%d", &seconds); err == nil {
			profile.Token.Expiration = time.Now().Add(time.Duration(seconds) * time.Second)
		}
	}

	// Cookies.
	for _, c := range succ.Tokens.WebsiteCookies {
		profile.Cookies = append(profile.Cookies, Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  c.Expires,
			Secure:   c.Secure == "true",
			HttpOnly: c.HttpOnly == "true",
		})
	}

	return profile, nil
}
