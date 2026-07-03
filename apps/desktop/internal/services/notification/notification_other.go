//go:build !windows

package notification

func (s *Service) ShowToast(title, message string) error {
	s.log.Info("notification action: toast (mocked)", "title", title, "message", message)
	return nil
}
