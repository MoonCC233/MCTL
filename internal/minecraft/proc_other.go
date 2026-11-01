//go:build !windows

package minecraft

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}
