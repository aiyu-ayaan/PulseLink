//go:build !windows

package brightness

func (s *Service) GetBrightness() (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.log.Info("brightness action: get (mocked)")
	return BrightnessState{Internal: s.lastInternalLevel, External: s.lastExternalLevel}, nil
}

func (s *Service) SetBrightness(targetType string, level int) error {
	s.mu.Lock()
	if targetType == "internal" || targetType == "" {
		s.lastInternalLevel = level
	} else {
		s.lastExternalLevel = level
	}
	s.mu.Unlock()
	s.log.Info("brightness action: set (mocked)", "type", targetType, "level", level)
	return nil
}
