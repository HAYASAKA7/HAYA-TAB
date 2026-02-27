//go:build windows

package store

import (
	"strings"
	"syscall"
	"unsafe"
)

// detectSystemLanguageTag returns the BCP 47 language tag from the Windows API.
func detectSystemLanguageTag() string {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetUserDefaultLocaleName")

	buf := make([]uint16, 85) // LOCALE_NAME_MAX_LENGTH
	ret, _, _ := proc.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if ret == 0 {
		return ""
	}

	return strings.TrimSpace(syscall.UTF16ToString(buf))
}
