//go:build windows

package tui

import "golang.org/x/sys/windows"

// terminateProcess ends a process using TerminateProcess. Windows has no
// SIGTERM equivalent that the kernel delivers to arbitrary processes.
func terminateProcess(pid int) error {
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	return windows.TerminateProcess(handle, 1)
}
