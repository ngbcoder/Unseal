package audible

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ngbcoder/Unseal/internal/audible/model"
)

// RefreshToken refreshes the access token if it is close to expiring.
//
// It posts to /auth/token and updates the profile in place.
func (a *Api) RefreshToken(ctx context.Context) error {
	if a.Profile == nil {
		return fmt.Errorf("audible: no profile to refresh")
	}
	if a.Profile.IsValid() {
		return nil
	}

	values := url.Values{}
	values.Set("app_name", "Audible")
	values.Set("app_version", "4.60")
	values.Set("source_token", a.Profile.Token.RefreshToken)
	values.Set("requested_token_type", "access_token")
	values.Set("source_token_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", a.AmazonURL()+"/auth/token", strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("x-amzn-identity-auth-domain", "api.amazon."+LocaleFor(string(a.Profile.Region)).TLD)

	resp, err := a.Amazon.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("audible: token refresh failed: HTTP %d: %s", resp.StatusCode, string(data))
	}

	var tok model.TokenResponse
	if err := json.Unmarshal(data, &tok); err != nil {
		return fmt.Errorf("audible: cannot parse the token response: %w", err)
	}

	a.Profile.Token.AccessToken = tok.AccessToken
	if tok.ExpiresIn != "" {
		var seconds int
		if _, err := fmt.Sscanf(tok.ExpiresIn, "%d", &seconds); err == nil {
			a.Profile.Token.Expiration = time.Now().Add(time.Duration(seconds) * time.Second)
		}
	}
	return nil
}
