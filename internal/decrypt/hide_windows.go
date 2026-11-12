//go:build windows

package decrypt

import (
	"os/exec"
	"syscall"
)

// hideConsole prevents a console window from flashing when ffmpeg is
// spawned on Windows.
func hideConsole(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags = 0x08000000 // CREATE_NO_WINDOW
}
