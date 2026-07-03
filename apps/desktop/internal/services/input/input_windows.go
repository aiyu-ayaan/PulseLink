//go:build windows

package input

import (
	"syscall"
	"unsafe"
)

var (
	user32Input       = syscall.NewLazyDLL("user32.dll")
	procMouseEvent    = user32Input.NewProc("mouse_event")
	procKeybdEvent    = user32Input.NewProc("keybd_event")
	procSetCursorPos  = user32Input.NewProc("SetCursorPos")
	procGetCursorPos  = user32Input.NewProc("GetCursorPos")
)

const (
	MOUSEEVENTF_LEFTDOWN   = 0x0002
	MOUSEEVENTF_LEFTUP     = 0x0004
	MOUSEEVENTF_RIGHTDOWN  = 0x0008
	MOUSEEVENTF_RIGHTUP    = 0x0010
	MOUSEEVENTF_MIDDLEDOWN = 0x0020
	MOUSEEVENTF_MIDDLEUP   = 0x0040
)

type POINT struct {
	X int32
	Y int32
}

func (s *Service) MouseMove(x, y int, relative bool) error {
	s.log.Debug("input action: mouse_move", "x", x, "y", y, "relative", relative)
	if relative {
		var pt POINT
		r1, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		if r1 != 0 {
			newX := int(pt.X) + x
			newY := int(pt.Y) + y
			procSetCursorPos.Call(uintptr(newX), uintptr(newY))
		}
	} else {
		procSetCursorPos.Call(uintptr(x), uintptr(y))
	}
	return nil
}

func (s *Service) MouseClick(button string) error {
	s.log.Debug("input action: mouse_click", "button", button)
	var downFlags, upFlags uintptr
	switch button {
	case "left", "":
		downFlags = MOUSEEVENTF_LEFTDOWN
		upFlags = MOUSEEVENTF_LEFTUP
	case "right":
		downFlags = MOUSEEVENTF_RIGHTDOWN
		upFlags = MOUSEEVENTF_RIGHTUP
	case "middle":
		downFlags = MOUSEEVENTF_MIDDLEDOWN
		upFlags = MOUSEEVENTF_MIDDLEUP
	}

	procMouseEvent.Call(downFlags, 0, 0, 0, 0)
	procMouseEvent.Call(upFlags, 0, 0, 0, 0)
	return nil
}

func (s *Service) KeyPress(key string) error {
	s.log.Info("input action: keypress", "key", key)
	if len(key) == 1 {
		char := byte(key[0])
		vk := char
		if char >= 'a' && char <= 'z' {
			vk = char - 32
		}
		
		procKeybdEvent.Call(uintptr(vk), 0, 0, 0)
		procKeybdEvent.Call(uintptr(vk), 0, 2, 0) // KEYUP
	}
	return nil
}
