//go:build !windows

package brightness

import "fmt"

func (s *Service) GetBrightness() (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.log.Info("brightness action: get (mocked)")
	monitors := s.lastMonitors
	if len(monitors) == 0 {
		monitors = []MonitorBrightness{
			{ID: "monitor-0", Name: "Display 1 (Primary)", Level: 100, Method: "gamma"},
		}
	}
	return BrightnessState{Monitors: monitors}, nil
}

func (s *Service) SetBrightness(monitorID string, level int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.log.Info("brightness action: set (mocked)", "monitor", monitorID, "level", level)
	found := false
	for i := range s.lastMonitors {
		if s.lastMonitors[i].ID == monitorID || monitorID == "all" {
			s.lastMonitors[i].Level = level
			s.lastLevels[s.lastMonitors[i].ID] = level
			found = true
		}
	}
	if !found && monitorID != "all" {
		return fmt.Errorf("unknown monitor: %s", monitorID)
	}
	return nil
}
