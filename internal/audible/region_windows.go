//go:build windows

package audible

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	procGetUserDefaultLocaleName = kernel32.NewProc("GetUserDefaultLocaleName")
)

// SystemLocale returns the user's format locale, for example "en-US".
//
// The format locale follows the user's region, which is a better signal for
// the Audible marketplace than the interface language does. It is read
// straight from the Win32 API, because the standard library does not expose
// it.
func SystemLocale() string {
	// LOCALE_NAME_MAX_LENGTH is 85.
	buf := make([]uint16, 85)

	r, _, _ := procGetUserDefaultLocaleName.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if r == 0 {
		return ""
	}

	return windows.UTF16ToString(buf)
}
