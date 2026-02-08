//go:build !windows
// +build !windows

package tui

import (
	"os/exec"
	"syscall"
)

// setupProcessGroup configures the command to run in a new process group (Unix/Linux/macOS)
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup kills the entire process group (Unix/Linux/macOS)
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd != nil && cmd.Process != nil {
		// Kill process group (negative PID kills the group)
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		cmd.Process.Kill()
		return cmd.Wait()
	}
	return nil
}
