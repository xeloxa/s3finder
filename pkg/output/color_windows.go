//go:build windows

package output

import (
	"os"
	"runtime"

	"golang.org/x/sys/windows"
)

func init() {
	enableWindowsANSI()
}

// enableWindowsANSI enables ANSI escape sequence processing on Windows 10+
func enableWindowsANSI() {
	if runtime.GOOS != "windows" {
		return
	}

	stdout := windows.Handle(os.Stdout.Fd())
	var mode uint32

	err := windows.GetConsoleMode(stdout, &mode)
	if err != nil {
		return
	}

	// ENABLE_VIRTUAL_TERMINAL_PROCESSING enables ANSI escape codes
	const ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004
	mode |= ENABLE_VIRTUAL_TERMINAL_PROCESSING

	_ = windows.SetConsoleMode(stdout, mode)
}
