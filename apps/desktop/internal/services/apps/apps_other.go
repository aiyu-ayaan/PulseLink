//go:build !windows

package apps

import "errors"

func (s *Service) LaunchApp(name string) error {
	s.log.Info("apps action: launch (mocked)", "name", name)
	return nil
}
