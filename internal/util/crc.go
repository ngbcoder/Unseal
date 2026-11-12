package util

import (
	"hash/crc32"
)

// CRC32 returns the CRC-32 (IEEE 802.3, reflected, polynomial 0xEDB88320)
// checksum of b.
//
// This matches the original implementation's Crc32 class exactly. It is used to
// compare account aliases for equality without storing them in the clear.
func CRC32(b []byte) uint32 {
	return crc32.ChecksumIEEE(b)
}

// CRC32String returns the CRC-32 checksum of s.
func CRC32String(s string) uint32 {
	return crc32.ChecksumIEEE([]byte(s))
}
