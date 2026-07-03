//go:build windows

package apps

import (
	"errors"
	"os/exec"
)

func (s *Service) LaunchApp(name string) error {
	s.log.Info("apps action: launch", "name", name)
	for _, app := range predefinedApps {
		if app.Name == name {
			cmd := exec.Command(app.Path)
			err := cmd.Start()
			if err != nil {
				return err
			}
			return nil
		}
	}
	return errors.New("app not found")
}
