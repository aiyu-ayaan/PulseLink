//go:build !windows

package brightness

func (s *Service) GetBrightness() (any, error) {
	s.log.Info("brightness action: get (mocked)")
	return BrightnessState{Internal: 80, External: 70}, nil
}

func (s *Service) SetBrightness(targetType string, level int) error {
	s.log.Info("brightness action: set (mocked)", "type", targetType, "level", level)
	return nil
}
