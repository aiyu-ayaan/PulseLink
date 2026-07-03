//go:build windows

package power

import (
	"os/exec"
)

func (s *Service) Lock() error {
	s.log.Info("power action: lock")
	return exec.Command("rundll32.exe", "user32.dll,LockWorkStation").Run()
}

func (s *Service) Sleep() error {
	s.log.Info("power action: sleep")
	return exec.Command("rundll32.exe", "powrprof.dll,SetSuspendState", "0,1,0").Run()
}

func (s *Service) Restart() error {
	s.log.Info("power action: restart")
	return exec.Command("shutdown.exe", "/r", "/t", "0").Run()
}

func (s *Service) Shutdown() error {
	s.log.Info("power action: shutdown")
	return exec.Command("shutdown.exe", "/s", "/t", "0").Run()
}
