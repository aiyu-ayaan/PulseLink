//go:build !windows

package volume

func (s *Service) VolumeUp() error {
	s.log.Info("volume action: up (mocked)")
	return nil
}

func (s *Service) VolumeDown() error {
	s.log.Info("volume action: down (mocked)")
	return nil
}

func (s *Service) VolumeMute() error {
	s.log.Info("volume action: mute (mocked)")
	return nil
}

func (s *Service) GetVolume() (any, error) {
	s.log.Info("volume action: get (mocked)")
	return VolumeState{Level: 75, Muted: false}, nil
}

func (s *Service) SetVolume(level int) error {
	s.log.Info("volume action: set (mocked)", "level", level)
	return nil
}
