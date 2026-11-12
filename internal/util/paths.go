package util

import (
	"path/filepath"
	"strings"
	"unicode/utf16"
)

// longPathThreshold is the length at which a Windows path is rewritten into
// the extended-length form.
//
// Windows refuses to open paths longer than 260 characters through the normal
// API. The extended-length form, introduced with the "\\?\" prefix, lifts that
// limit. The original implementation used the same threshold.
const longPathThreshold = 250

// uncPrefix is the extended-length prefix for a local path.
const uncPrefix = `\\?\`

// uncNetworkPrefix is the extended-length prefix for a UNC network path.
const uncNetworkPrefix = `\\?\UNC\`

// IsUnc reports whether path already carries the extended-length prefix.
func IsUnc(path string) bool {
	return strings.HasPrefix(path, uncPrefix)
}

// AsUnc returns path in the extended-length form.
//
// A local path becomes "\\?\C:\dir\file". A UNC path becomes
// "\\?\UNC\server\share". A path that is already extended is returned
// unchanged.
func AsUnc(path string) string {
	if IsUnc(path) {
		return path
	}

	// A path starting with two backslashes is a UNC network path.
	if strings.HasPrefix(path, `\\`) && !strings.HasPrefix(path, `\\\`) {
		return uncNetworkPrefix + path[2:]
	}

	return uncPrefix + path
}

// AsUncIfLong returns path in the extended-length form when it is long enough
// to need it, and the absolute path otherwise.
//
// The path is made absolute first, so a relative path is resolved against the
// working directory before its length is judged. When the path cannot be
// resolved it is used as given.
func AsUncIfLong(path string) string {
	if path == "" {
		return path
	}
	if IsUnc(path) {
		return path
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}

	if utf16Len(abs) < longPathThreshold {
		return abs
	}

	return AsUnc(abs)
}

// utf16Len returns the length of s in UTF-16 code units, which is how
// Windows and .NET measure a path.
func utf16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}
