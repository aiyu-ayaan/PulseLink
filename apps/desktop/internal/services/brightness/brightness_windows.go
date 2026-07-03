//go:build windows

package brightness

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func (s *Service) GetBrightness() (any, error) {
	cmd := "Get-CimInstance -Namespace root/WMI -ClassName WmiMonitorBrightness | Select-Object -ExpandProperty CurrentBrightness"
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", cmd).Output()
	
	internalLevel := 50
	if err == nil {
		valStr := strings.TrimSpace(string(out))
		if val, err := strconv.Atoi(valStr); err == nil {
			internalLevel = val
		}
	} else {
		s.log.Debug("could not query built-in monitor brightness (likely a desktop PC without WMI battery/brightness support)")
		internalLevel = 100
	}

	return BrightnessState{
		Internal: internalLevel,
		External: 80, // External monitors DDC/CI mock / static status
	}, nil
}

func (s *Service) SetBrightness(targetType string, level int) error {
	s.log.Info("brightness action: set", "type", targetType, "level", level)
	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}

	if targetType == "internal" || targetType == "" {
		cmd := fmt.Sprintf("(Get-CimInstance -Namespace root/WMI -ClassName WmiMonitorBrightnessMethods).WmiSetBrightness(1, %d)", level)
		err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", cmd).Run()
		if err != nil {
			return fmt.Errorf("powershell set brightness: %w", err)
		}
	} else {
		s.log.Info("set external monitor brightness (DDC/CI mock path)", "level", level)
	}
	return nil
}
