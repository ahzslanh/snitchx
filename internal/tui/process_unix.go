//go:build !windows

package tui

import "syscall"

// terminateProcess asks a process to shut down gracefully with SIGTERM.
func terminateProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}
