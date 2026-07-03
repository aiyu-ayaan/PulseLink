//go:build windows

package clipboard

import (
	"syscall"
	"unsafe"
)

var (
	user32DLL           = syscall.NewLazyDLL("user32.dll")
	kernel32DLL         = syscall.NewLazyDLL("kernel32.dll")

	procOpenClipboard  = user32DLL.NewProc("OpenClipboard")
	procCloseClipboard = user32DLL.NewProc("CloseClipboard")
	procEmptyClipboard = user32DLL.NewProc("EmptyClipboard")
	procGetClipboard   = user32DLL.NewProc("GetClipboardData")
	procSetClipboard   = user32DLL.NewProc("SetClipboardData")

	procGlobalAlloc  = kernel32DLL.NewProc("GlobalAlloc")
	procGlobalFree   = kernel32DLL.NewProc("GlobalFree")
	procGlobalLock   = kernel32DLL.NewProc("GlobalLock")
	procGlobalUnlock = kernel32DLL.NewProc("GlobalUnlock")
)

const (
	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002
)

func (s *Service) GetText() (string, error) {
	r1, _, _ := procOpenClipboard.Call(0)
	if r1 == 0 {
		return "", nil
	}
	defer procCloseClipboard.Call()

	hMem, _, _ := procGetClipboard.Call(CF_UNICODETEXT)
	if hMem == 0 {
		return "", nil
	}

	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return "", nil
	}
	defer procGlobalUnlock.Call(hMem)

	// Traverse UTF-16 buffer to calculate string length
	p := (*uint16)(unsafe.Pointer(ptr))
	length := 0
	for {
		val := *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(length*2)))
		if val == 0 {
			break
		}
		length++
	}

	// Slice and convert
	words := make([]uint16, length)
	src := (*[1 << 28]uint16)(unsafe.Pointer(ptr))[:length]
	copy(words, src)

	return syscall.UTF16ToString(words), nil
}

func (s *Service) SetText(text string) error {
	r1, _, err := procOpenClipboard.Call(0)
	if r1 == 0 {
		return err
	}
	defer procCloseClipboard.Call()

	r1, _, err = procEmptyClipboard.Call()
	if r1 == 0 {
		return err
	}

	utf16, err := syscall.UTF16FromString(text)
	if err != nil {
		return err
	}

	size := uintptr(len(utf16) * 2)
	hMem, _, err := procGlobalAlloc.Call(GMEM_MOVEABLE, size)
	if hMem == 0 {
		return err
	}

	ptr, _, err := procGlobalLock.Call(hMem)
	if ptr == 0 {
		procGlobalFree.Call(hMem)
		return err
	}

	dst := (*[1 << 28]uint16)(unsafe.Pointer(ptr))[:len(utf16)]
	copy(dst, utf16)
	procGlobalUnlock.Call(hMem)

	r1, _, err = procSetClipboard.Call(CF_UNICODETEXT, hMem)
	if r1 == 0 {
		procGlobalFree.Call(hMem)
		return err
	}

	return nil
}
