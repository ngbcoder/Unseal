package model

// LicenseResponse is the payload of POST /1.0/content/{asin}/licenserequest.
type LicenseResponse struct {
	ContentLicense ContentLicense `json:"content_license"`
	ResponseGroups []string       `json:"response_groups"`
}

// ContentLicense is the licence Audible grants for one title.
type ContentLicense struct {
	// Granted or Denied.
	StatusCode string `json:"status_code"`
	// The asin the licence is for.
	ASIN string `json:"asin"`
	// The encrypted voucher, base64.
	LicenseResponse string `json:"license_response"`
	// The denial reasons, when status_code is Denied.
	LicenseDenialReasons []LicenseDenialReason `json:"license_denial_reasons"`
	// The metadata that travels with the licence.
	ContentMetadata ContentMetadata `json:"content_metadata"`
	// The voucher, after decryption.
	Voucher *Voucher `json:"-"`
}

// LicenseDenialReason is one reason Audible gave for refusing a licence.
type LicenseDenialReason struct {
	Message         string `json:"message"`
	RejectionReason string `json:"rejectionReason"`
	ValidationType  string `json:"validationType"`
}

// ContentMetadata carries the chapter tree, the size and the download URL.
type ContentMetadata struct {
	ChapterInfo      *ChapterInfo     `json:"chapter_info"`
	ContentReference ContentReference `json:"content_reference"`
	ContentUrl       ContentUrl       `json:"content_url"`
}

// ChapterInfo is the chapter tree of a title.
type ChapterInfo struct {
	BrandIntroDurationMs *int      `json:"brandIntroDurationMs"`
	BrandOutroDurationMs *int      `json:"brandOutroDurationMs"`
	Chapters             []Chapter `json:"chapters"`
	IsAccurate           *bool     `json:"is_accurate"`
	RuntimeLengthMs      *int      `json:"runtime_length_ms"`
	RuntimeLengthSec     *int      `json:"runtime_length_sec"`
}

// Chapter is one entry in the chapter tree.
type Chapter struct {
	LengthMs       *int      `json:"length_ms"`
	StartOffsetMs  *int      `json:"start_offset_ms"`
	StartOffsetSec *int      `json:"start_offset_sec"`
	Title          string    `json:"title"`
	Chapters       []Chapter `json:"chapters"`
}

// ContentReference carries the size and the codec of the payload.
type ContentReference struct {
	ACR           string `json:"acr"`
	ASIN          string `json:"asin"`
	ContentFormat string `json:"content_format"`
	ContentSize   *int64 `json:"content_size_in_bytes"`
	FileVersion   string `json:"file_version"`
	Marketplace   string `json:"marketplace"`
	Sku           string `json:"sku"`
	Tempo         string `json:"tempo"`
	Version       string `json:"version"`
}

// ContentUrl carries the download address of the payload.
type ContentUrl struct {
	OfflineUrl string `json:"offline_url"`
}

// LicenseRequest is the body of the license request.
type LicenseRequest struct {
	ConsumptionType   string   `json:"consumption_type"`
	SupportedDrmTypes []string `json:"supported_drm_types"`
	Quality           string   `json:"quality"`
	ResponseGroups    string   `json:"response_groups"`
}

// Voucher is the decrypted voucher, carrying the DRM key and IV.
//
// It is defined in this package's voucher.go.
