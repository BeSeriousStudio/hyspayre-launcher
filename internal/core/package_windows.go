//go:build windows

package core

import (
    "syscall"
    "unsafe"
)

// IsMSIX reports whether Windows launched the executable with package identity.
// GetCurrentPackageFullName returns ERROR_INSUFFICIENT_BUFFER (122) when the
// process has a package identity and the caller has not supplied a buffer.
func IsMSIX() bool {
    var needed uint32
    proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentPackageFullName")
    code, _, _ := proc.Call(uintptr(unsafe.Pointer(&needed)), 0)
    return code == 122
}
