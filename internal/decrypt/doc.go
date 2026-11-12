// Package decrypt removes the DRM from a downloaded Audible payload.
//
// The package exposes a Decryptor interface. The shipped implementation
// delegates to ffmpeg's native aax demuxer, which understands the AAXC
// container and accepts the account-scoped key and IV directly:
//
//	ffmpeg -audible_key <hex> -audible_iv <hex> -i in.aaxc -c copy out.m4b
//
// A pure-Go implementation can be added later behind the same interface.
//
// Layer L3 (domain).
package decrypt
