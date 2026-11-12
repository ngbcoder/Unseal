package model

// Voucher is the decrypted voucher, carrying the DRM key and IV.
type Voucher struct {
	Key   string `json:"key"`
	IV    string `json:"iv"`
	Rules []Rule `json:"rules"`
}

// Rule is one rule in the voucher.
type Rule struct {
	Name       string      `json:"name"`
	Parameters []Parameter `json:"parameters"`
}

// Parameter is one parameter of a rule.
type Parameter struct {
	ExpireDate string `json:"expireDate"`
	Type       string `json:"type"`
}
