//go:build !windows

package decrypt

import "os/exec"

// hideConsole is a no-op on platforms other than Windows.
func hideConsole(cmd *exec.Cmd) {}
