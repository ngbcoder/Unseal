package audible

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/ngbcoder/Unseal/internal/audible/model"
)

// GetDownloadLicense requests a licence for a title and decrypts the
// voucher it carries.
//
// On success the licence's Voucher field is populated with the DRM key and
// IV, and the offline URL and the content size are available on the
// licence's ContentMetadata.
func (a *Api) GetDownloadLicense(ctx context.Context, asin, quality string) (*model.ContentLicense, error) {
	if err := a.ensureProfile(); err != nil {
		return nil, err
	}

	reqBody := model.LicenseRequest{
		ConsumptionType:   "Download",
		SupportedDrmTypes: []string{"Adrm", "Mpeg"},
		Quality:           quality,
		ResponseGroups:    "last_position_heard,pdf_url,content_reference,chapter_info",
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	path := "/1.0/content/" + asin + "/licenserequest"
	req, err := a.makeRequest(ctx, "POST", a.AmazonURL(), path, data, true)
	if err != nil {
		return nil, err
	}

	body, err := a.sendForString(req, a.Amazon)
	if err != nil {
		return nil, err
	}

	var resp model.LicenseResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("audible: cannot parse the licence response: %w", err)
	}

	licence := resp.ContentLicense
	if licence.ASIN == "" {
		licence.ASIN = asin
	}

	// Validate and, when granted, decrypt the voucher.
	if err := decryptLicense(&licence, a.Profile.DeviceInfo.Type, a.Profile.DeviceInfo.Serial, a.Profile.CustomerInfo.AccountID); err != nil {
		return &licence, err
	}

	return &licence, nil
}

// decryptLicense validates a licence response and decrypts its voucher.
//
// The key and IV are derived from
//
//	SHA256(ASCII(deviceType + serial + accountId + asin))
//
// split into two halves. The licence response is then decrypted with
// AES-128-CBC and no padding, and the voucher is the JSON up to the first
// zero byte.
func decryptLicense(licence *model.ContentLicense, deviceType, serial, accountID string) error {
	if licence == nil {
		return fmt.Errorf("audible: nil licence")
	}

	if licence.StatusCode != "Granted" {
		if licence.StatusCode == "Denied" && len(licence.LicenseDenialReasons) > 0 {
			// Logged by the caller; nothing to do here.
		}
		return fmt.Errorf("audible: licence %s", licence.StatusCode)
	}

	hashable := deviceType + serial + accountID + licence.ASIN
	hash := sha256.Sum256([]byte(hashable))

	key := hash[:16]
	iv := hash[16:32]

	encrypted, err := base64.StdEncoding.DecodeString(licence.LicenseResponse)
	if err != nil {
		return fmt.Errorf("audible: cannot decode the licence response: %w", err)
	}

	// Truncate to a multiple of the block size, matching the original.
	n := len(encrypted) & 0x7ffffff0
	if n == 0 {
		return fmt.Errorf("audible: the licence response is too short")
	}
	encrypted = encrypted[:n]

	plain, err := aesCBCDecrypt(key, iv, encrypted)
	if err != nil {
		return fmt.Errorf("audible: cannot decrypt the licence: %w", err)
	}

	// The voucher is the JSON up to the first zero byte.
	end := len(plain)
	for i, b := range plain {
		if b == 0 {
			end = i
			break
		}
	}
	plain = plain[:end]

	var voucher model.Voucher
	if err := json.Unmarshal(plain, &voucher); err != nil {
		return fmt.Errorf("audible: cannot parse the voucher: %w", err)
	}
	if voucher.Key == "" || voucher.IV == "" {
		return fmt.Errorf("audible: the voucher is incomplete")
	}

	licence.Voucher = &voucher
	return nil
}

// aesCBCDecrypt decrypts src with AES-CBC and no padding.
func aesCBCDecrypt(key, iv, src []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(src)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("audible: the ciphertext length is not a multiple of the block size")
	}
	dst := make([]byte, len(src))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(dst, src)
	return dst, nil
}
