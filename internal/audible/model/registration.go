package model

// RegistrationResponse is the payload of POST /auth/register.
type RegistrationResponse struct {
	Response  RegistrationResponseInner `json:"response"`
	RequestID string                    `json:"request_id"`
}

// RegistrationResponseInner is the inner response object.
type RegistrationResponseInner struct {
	Success RegistrationSuccess `json:"success"`
}

// RegistrationSuccess carries the extensions and the tokens.
type RegistrationSuccess struct {
	Extensions Extensions `json:"extensions"`
	Tokens     Tokens     `json:"tokens"`
	CustomerID string     `json:"customer_id"`
}

// Extensions carries the device and customer info.
type Extensions struct {
	DeviceInfo   DeviceInfo   `json:"device_info"`
	CustomerInfo CustomerInfo `json:"customer_info"`
}

// DeviceInfo is the device identity Audible assigned.
type DeviceInfo struct {
	DeviceName         string `json:"device_name"`
	DeviceSerialNumber string `json:"device_serial_number"`
	DeviceType         string `json:"device_type"`
}

// CustomerInfo is the customer identity Audible returned.
type CustomerInfo struct {
	AccountPool string `json:"account_pool"`
	UserID      string `json:"user_id"`
	HomeRegion  string `json:"home_region"`
	Name        string `json:"name"`
	GivenName   string `json:"given_name"`
}

// Tokens carries every credential group.
type Tokens struct {
	WebsiteCookies            []WebsiteCookie           `json:"website_cookies"`
	StoreAuthenticationCookie StoreAuthenticationCookie `json:"store_authentication_cookie"`
	MacDms                    MacDms                    `json:"mac_dms"`
	Bearer                    Bearer                    `json:"bearer"`
}

// WebsiteCookie is one cookie Audible set.
type WebsiteCookie struct {
	Path     string `json:"Path"`
	Secure   string `json:"Secure"`
	Value    string `json:"Value"`
	Expires  string `json:"Expires"`
	Domain   string `json:"Domain"`
	HttpOnly string `json:"HttpOnly"`
	Name     string `json:"Name"`
}

// StoreAuthenticationCookie is the store-authentication cookie.
type StoreAuthenticationCookie struct {
	Cookie string `json:"cookie"`
}

// MacDms carries the device private key and the ADP token.
type MacDms struct {
	DevicePrivateKey string `json:"device_private_key"`
	AdpToken         string `json:"adp_token"`
}

// Bearer carries the access and refresh tokens.
type Bearer struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    string `json:"expires_in"`
}

// TokenResponse is the payload of POST /auth/token.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}
