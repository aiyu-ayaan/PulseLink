//go:build !windows

package clipboard

func (s *Service) GetText() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastText, nil
}

func (s *Service) SetText(text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastText = text
	return nil
}
