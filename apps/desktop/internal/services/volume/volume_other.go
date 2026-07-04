//go:build !windows

package volume

// Mock implementation for non-Windows dev builds. Keeps an in-memory level so
// set/up/down/mute behave and report state consistently, matching the Windows
// signatures.

func (s *Service) GetVolume() (VolumeState, error) {
	return s.mock, nil
}

func (s *Service) SetVolume(level int) (VolumeState, error) {
	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}
	s.log.Info("volume action: set (mocked)", "level", level)
	s.mock.Level = level
	return s.mock, nil
}

func (s *Service) VolumeUp() (VolumeState, error) {
	s.log.Info("volume action: up (mocked)")
	return s.SetVolume(s.mock.Level + 2)
}

func (s *Service) VolumeDown() (VolumeState, error) {
	s.log.Info("volume action: down (mocked)")
	return s.SetVolume(s.mock.Level - 2)
}

func (s *Service) VolumeMute() (VolumeState, error) {
	s.log.Info("volume action: mute (mocked)")
	s.mock.Muted = !s.mock.Muted
	return s.mock, nil
}
