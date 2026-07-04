//go:build !windows

package media

func (s *Service) PlayPause() error {
	s.log.Info("media action: play_pause (mocked)")
	return nil
}

func (s *Service) Next() error {
	s.log.Info("media action: next (mocked)")
	return nil
}

func (s *Service) Previous() error {
	s.log.Info("media action: previous (mocked)")
	return nil
}

func (s *Service) StopMedia() error {
	s.log.Info("media action: stop (mocked)")
	return nil
}

func (s *Service) GetMediaState() (MediaState, error) {
	return MediaState{
		Title:      "Mock Track",
		Artist:     "Mock Artist",
		AlbumTitle: "Mock Album",
		Status:     "Playing",
	}, nil
}

