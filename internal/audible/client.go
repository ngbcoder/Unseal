package audible

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Api is the Audible protocol client.
//
// It holds the profile it is acting for and the two HTTP clients, one for
// the Amazon host and one for the Audible host.
type Api struct {
	Profile *Profile
	Region  string

	Amazon  *HTTPClient
	Audible *HTTPClient
}

// NewApi creates an Api.
func NewApi(profile *Profile, region string) (*Api, error) {
	amazon, err := NewHTTPClient()
	if err != nil {
		return nil, err
	}
	audible, err := NewHTTPClient()
	if err != nil {
		return nil, err
	}
	return &Api{
		Profile: profile,
		Region:  region,
		Amazon:  amazon,
		Audible: audible,
	}, nil
}

// AmazonURL returns the Amazon API base URL.
func (a *Api) AmazonURL() string { return AmazonHost(a.Region) }

// AudibleURL returns the Audible API base URL.
func (a *Api) AudibleURL() string { return AudibleHost(a.Region) }

// makeRequest builds a request, optionally signed.
func (a *Api) makeRequest(ctx context.Context, method, baseURL, path string, body []byte, signed bool) (*http.Request, error) {
	u, err := url.Parse(baseURL + path)
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	if signed {
		if err := a.signRequest(req, body); err != nil {
			return nil, err
		}
	}

	return req, nil
}

// signRequest adds the ADP signature headers.
func (a *Api) signRequest(req *http.Request, body []byte) error {
	if a.Profile == nil {
		return fmt.Errorf("audible: no profile to sign with")
	}
	key, err := ParsePrivateKey(a.Profile.PrivateKey)
	if err != nil {
		return err
	}

	relativeURL := req.URL.RequestURI()
	bodyStr := ""
	if len(body) > 0 {
		bodyStr = string(body)
	}

	sig, err := Sign(req.Method, relativeURL, bodyStr, a.Profile.AdpToken, key, time.Now())
	if err != nil {
		return err
	}

	req.Header.Set("x-adp-token", a.Profile.AdpToken)
	req.Header.Set("x-adp-alg", "SHA256withRSA:1.0")
	req.Header.Set("x-adp-signature", sig)
	return nil
}

// sendForString sends a request and returns the response body as a string.
func (a *Api) sendForString(req *http.Request, client *HTTPClient) (string, error) {
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return string(data), fmt.Errorf("audible: HTTP %d: %s", resp.StatusCode, string(data))
	}

	return string(data), nil
}

// sendForBytes sends a request and returns the response body as bytes.
func (a *Api) sendForBytes(req *http.Request, client *HTTPClient) ([]byte, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return data, fmt.Errorf("audible: HTTP %d", resp.StatusCode)
	}

	return data, nil
}

// ensureProfile checks that a profile is set.
func (a *Api) ensureProfile() error {
	if a.Profile == nil {
		return fmt.Errorf("audible: no profile is selected")
	}
	return nil
}
