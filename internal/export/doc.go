// Package export writes the decrypted audiobook into the interchange format
// consumed by AAX Audio Converter.
//
// For each title it produces:
//
//	<name>.aax                    — a copy of the decrypted .m4b
//	content_metadata_<ASIN>.json — the chapter tree and runtimes
//	<ASIN>.json                   — a compact product record
//	series_titles_<SERIES>.json   — the ordered book list of each series
//
// Multi-part books are exported part by part; the series sidecar is written
// only once to avoid duplicates, and leading chapters that repeat the brand
// intro are dropped.
//
// Layer L3 (domain).
package export
