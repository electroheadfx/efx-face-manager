//go:build windows
// +build windows

package tui

import (
	"os/exec"
)

// setupProcessGroup is a no-op on Windows
func setupProcessGroup(cmd *exec.Cmd) {
	// Windows doesn't support process groups the same way
	// Process will be killed directly
}

// killProcessGroup kills the process on Windows
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd != nil && cmd.Process != nil {
		return cmd.Process.Kill()
	}
	return nil
}
