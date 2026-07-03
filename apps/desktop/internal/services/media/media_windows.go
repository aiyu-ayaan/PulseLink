//go:build windows

package media

import (
	"syscall"
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	procKeybdEvent = user32.NewProc("keybd_event")
)

const (
	VK_MEDIA_NEXT_TRACK = 0xB5
	VK_MEDIA_PREV_TRACK = 0xB6
	VK_MEDIA_STOP       = 0xB2
	VK_MEDIA_PLAY_PAUSE = 0xB3
	KEYEVENTF_KEYUP     = 0x0002
)

func pressKey(vk byte) {
	procKeybdEvent.Call(uintptr(vk), 0, 0, 0)
	procKeybdEvent.Call(uintptr(vk), 0, KEYEVENTF_KEYUP, 0)
}

func (s *Service) PlayPause() error {
	s.log.Info("media action: play_pause")
	pressKey(VK_MEDIA_PLAY_PAUSE)
	return nil
}

func (s *Service) Next() error {
	s.log.Info("media action: next")
	pressKey(VK_MEDIA_NEXT_TRACK)
	return nil
}

func (s *Service) Previous() error {
	s.log.Info("media action: previous")
	pressKey(VK_MEDIA_PREV_TRACK)
	return nil
}

func (s *Service) StopMedia() error {
	s.log.Info("media action: stop")
	pressKey(VK_MEDIA_STOP)
	return nil
}
