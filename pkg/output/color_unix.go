//go:build !windows

package output

// enableWindowsANSI is a no-op on non-Windows platforms
func init() {
	// ANSI colors work natively on Unix-like systems
}
