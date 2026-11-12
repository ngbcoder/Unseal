// Package util holds small, dependency-free helpers shared across Unseal.
//
// It is the Go counterpart of the original project's ExtensionsVarious:
// filename pruning, long-path normalisation, timestamp rounding and formatting,
// CRC32 checksums and assorted string helpers.
//
// Layer L0 (foundation). Everything here must be pure and side-effect free so
// it can be unit tested without any fixtures.
package util
