package util

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenURL opens a URL in the user's default browser.
//
// It never uses ShellExecute directly, so the same code works on every
// platform.
func OpenURL(url string) error {
	switch runtime.GOOS {
	case "windows":
		return openWindows(url)
	case "darwin":
		return openCommand("open", url)
	default:
		return openCommand("xdg-open", url)
	}
}

// openCommand runs a helper program with the URL as its only argument.
func openCommand(name, url string) error {
	cmd := exec.Command(name, url)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("util: cannot open %s: %w", url, err)
	}
	// The helper detaches; do not wait for it.
	return nil
}
