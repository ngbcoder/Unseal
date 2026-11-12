// Package audible implements the Audible wire protocol.
//
// Responsibilities:
//
//   - device registration (POST /auth/register) and token refresh
//     (POST /auth/token)
//   - ADP request signing (x-adp-token / x-adp-alg / x-adp-signature)
//   - library listing (GET /1.0/library) and catalog lookup
//   - licence acquisition (POST /1.0/content/{asin}/licenserequest) including
//     voucher decryption
//   - CDN payload download
//
// The package is the highest-risk area of Unseal: the server-side contract can
// change without notice. Every cryptographic routine here is covered by
// fixed-vector unit tests.
//
// Layer L3 (domain). It depends on L0–L2 and never on the user interface.
package audible
