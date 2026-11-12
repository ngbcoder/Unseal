//go:build windows

package util

import (
	"fmt"
	"os/exec"
	"syscall"
)

// openWindows opens a URL with the system's default handler.
//
// rundll32 with url.dll is the way to open a URL without going through
// ShellExecute, which keeps the behaviour the same on every platform.
func openWindows(url string) error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("util: cannot open %s: %w", url, err)
	}
	return nil
}
