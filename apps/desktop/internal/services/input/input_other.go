//go:build !windows

package input

func (s *Service) MouseMove(x, y int, relative bool) error {
	s.log.Info("input action: mouse_move (mocked)", "x", x, "y", y, "relative", relative)
	return nil
}

func (s *Service) MouseClick(button string) error {
	s.log.Info("input action: mouse_click (mocked)", "button", button)
	return nil
}

func (s *Service) KeyPress(key string) error {
	s.log.Info("input action: keypress (mocked)", "key", key)
	return nil
}
