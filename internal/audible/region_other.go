//go:build !windows

package audible

import "os"

// SystemLocale returns the user's locale from the environment.
func SystemLocale() string {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(name); v != "" {
			return v
		}
	}
	return ""
}
