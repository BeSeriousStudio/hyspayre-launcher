//go:build windows

package social

import (
	"errors"
	"syscall"
	"unsafe"
)

// Windows DPAPI encrypts remembered refresh/access tokens for the currently
// logged-on Windows user. No hardcoded passphrase or plaintext token file.
type dataBlob struct {
	Size uint32
	Data *byte
}

var cryptoDLL = syscall.NewLazyDLL("crypt32.dll")
var procProtect = cryptoDLL.NewProc("CryptProtectData")
var procUnprotect = cryptoDLL.NewProc("CryptUnprotectData")
var kernelDLL = syscall.NewLazyDLL("kernel32.dll")
var procLocalFree = kernelDLL.NewProc("LocalFree")

const uiForbidden = 0x1

func cryptDPAPI(raw []byte, encrypt bool) ([]byte, error) {
	if len(raw) == 0 {
		return nil, errors.New("Leere Daten")
	}
	input := dataBlob{Size: uint32(len(raw)), Data: &raw[0]}
	var output dataBlob
	var ok uintptr
	var err error
	if encrypt {
		desc, _ := syscall.UTF16PtrFromString("Hyspayre Launcher Session")
		ok, _, err = procProtect.Call(uintptr(unsafe.Pointer(&input)), uintptr(unsafe.Pointer(desc)), 0, 0, 0, uiForbidden, uintptr(unsafe.Pointer(&output)))
	} else {
		ok, _, err = procUnprotect.Call(uintptr(unsafe.Pointer(&input)), 0, 0, 0, 0, uiForbidden, uintptr(unsafe.Pointer(&output)))
	}
	if ok == 0 {
		return nil, err
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(output.Data)))
	if output.Size == 0 || output.Size > 1000000 {
		return nil, errors.New("Ungültiger DPAPI-Puffer")
	}
	// Copy result before freeing Windows-owned allocation.
	out := append([]byte(nil), unsafe.Slice(output.Data, int(output.Size))...)
	return out, nil
}
func protect(raw []byte) ([]byte, error)   { return cryptDPAPI(raw, true) }
func unprotect(raw []byte) ([]byte, error) { return cryptDPAPI(raw, false) }
