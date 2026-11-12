package db

// EConversionState is the state of a conversion.
//
// The numeric order is semantically meaningful: several code paths compare
// against localLocked with < and > to decide whether a download or a
// decryption is still required. New states must be inserted with care, and
// the icon map and tooltip resources must be updated alongside.
//
// The values mirror the original EConversionState enum exactly.
type EConversionState int

// The conversion states, in their semantically significant order.
const (
	StateUnknown          EConversionState = 0  // Not in the account's library anymore
	StateRemote           EConversionState = 1  // Owned, nothing local yet
	StateDownload         EConversionState = 2  // Staged for download (transient UI state)
	StateLicenseGranted   EConversionState = 3  // License obtained, key/IV known
	StateLicenseDenied    EConversionState = 4  // Audible refused the license
	StateDownloading      EConversionState = 5  // Payload transfer in flight
	StateDownloadError    EConversionState = 6  // Transfer failed
	StateLocalLocked      EConversionState = 7  // Encrypted .aaxc is on disk
	StateUnlocking        EConversionState = 8  // Decryption in flight
	StateUnlockingFailed  EConversionState = 9  // Decryption failed
	StateLocalUnlocked    EConversionState = 10 // Decrypted .m4b is on disk
	StateExported         EConversionState = 11 // .aax + sidecar JSON written
	StateConverting       EConversionState = 12 // Export in flight
	StateConverted        EConversionState = 13 // Fully exported
	StateConvertedUnknown EConversionState = 14 // Exported artefact missing from disk
	StateConversionError  EConversionState = 15 // Export failed
)

// String returns the name of the state.
func (s EConversionState) String() string {
	names := []string{
		"unknown", "remote", "download", "license_granted", "license_denied",
		"downloading", "download_error", "local_locked", "unlocking",
		"unlocking_failed", "local_unlocked", "exported", "converting",
		"converted", "converted_unknown", "conversion_error",
	}
	if int(s) >= 0 && int(s) < len(names) {
		return names[s]
	}
	return "unknown"
}

// IsPersistent reports whether the state represents a result that should be
// saved to disk.
//
// The transient states (download, downloading, unlocking, converting) are
// never saved; the others are.
func (s EConversionState) IsPersistent() bool {
	switch s {
	case StateDownload, StateDownloading, StateUnlocking, StateConverting:
		return false
	}
	return true
}

// ECodec is the audio codec of a title.
type ECodec int

// The known codecs.
const (
	CodecUnknown ECodec = iota
	CodecAAX4464        // aax_44_64
	CodecAAX44128
	CodecAAX44192
)

// EDeliveryType is how Audible delivers a title.
type EDeliveryType int

// The delivery types.
const (
	DeliveryUnknown EDeliveryType = iota
	DeliveryStream
	DeliveryDownload
)

// EDownloadQuality is the audio quality requested from Audible.
type EDownloadQuality int

// The download qualities, from lowest to highest.
const (
	DownloadQualityLow EDownloadQuality = iota
	DownloadQualityNormal
	DownloadQualityHigh
	DownloadQualityExtreme
)

// String returns the name of the quality.
func (q EDownloadQuality) String() string {
	names := []string{"Low", "Normal", "High", "Extreme"}
	if int(q) >= 0 && int(q) < len(names) {
		return names[q]
	}
	return "unknown"
}
