//go:build !windows

package power

func (s *Service) Lock() error {
	s.log.Info("power action: lock (mocked)")
	return nil
}

func (s *Service) Sleep() error {
	s.log.Info("power action: sleep (mocked)")
	return nil
}

func (s *Service) Restart() error {
	s.log.Info("power action: restart (mocked)")
	return nil
}

func (s *Service) Shutdown() error {
	s.log.Info("power action: shutdown (mocked)")
	return nil
}
