//go:build !windows

package sysinfo

import "os"

func (s *Service) GetSysInfo() (any, error) {
	hostname, _ := os.Hostname()
	return SysInfoState{
		Hostname:     hostname,
		OS:           "MockOS",
		CPUUsage:     15.0,
		RAMTotal:     16384,
		RAMFree:      12000,
		BatteryPct:   90,
		IsCharging:   false,
		MonitorCount: 1,
	}, nil
}
