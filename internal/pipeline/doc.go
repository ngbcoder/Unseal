// Package pipeline orchestrates the download, decrypt and export stages.
//
// For each conversion the pipeline runs licence acquisition, payload download,
// DRM removal and export, persisting the conversion state after every
// transition so an interrupted run can be resumed.
//
// Downloads are serialised per book because each needs its own licence
// round-trip. Decryption is CPU-bound and is throttled by a semaphore so that
// only a bounded number of decryptions run at once.
//
// Layer L3 (domain).
package pipeline
