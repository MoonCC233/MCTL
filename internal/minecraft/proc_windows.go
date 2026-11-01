//go:build windows

package minecraft

import (
	"os/exec"
	"syscall"
)

// hideWindow prevents console flashes when probing Java or launching the game.
func hideWindow(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
}
