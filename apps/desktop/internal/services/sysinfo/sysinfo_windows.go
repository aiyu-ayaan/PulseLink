//go:build windows

package sysinfo

import (
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

type SYSTEM_POWER_STATUS struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	SystemStatusFlag    byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

var (
	kernel32SysInfo           = syscall.NewLazyDLL("kernel32.dll")
	user32SysInfo             = syscall.NewLazyDLL("user32.dll")
	procGetSystemPowerStatus  = kernel32SysInfo.NewProc("GetSystemPowerStatus")
	procGetSystemMetrics      = user32SysInfo.NewProc("GetSystemMetrics")
)

const SM_CMONITORS = 80

func (s *Service) GetSysInfo() (any, error) {
	hostname, _ := os.Hostname()

	var sps SYSTEM_POWER_STATUS
	r1, _, _ := procGetSystemPowerStatus.Call(uintptr(unsafe.Pointer(&sps)))
	
	batteryPct := 100
	isCharging := true
	if r1 != 0 {
		if sps.BatteryLifePercent != 255 {
			batteryPct = int(sps.BatteryLifePercent)
		}
		isCharging = sps.ACLineStatus == 1
	}

	monitorsCount, _, _ := procGetSystemMetrics.Call(SM_CMONITORS)
	if monitorsCount == 0 {
		monitorsCount = 1
	}

	cpuVal := 12.0
	var ramTotal, ramFree uint64 = 16384, 8192

	cmd := "Get-CimInstance Win32_OperatingSystem | Select-Object TotalVisibleMemorySize, FreePhysicalMemory; (Get-CimInstance Win32_Processor | Measure-Object -Property LoadPercentage -Average).Average"
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", cmd).Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\r\n")
		if len(lines) < 2 {
			lines = strings.Split(strings.TrimSpace(string(out)), "\n")
		}
		
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				t, err1 := strconv.ParseUint(parts[0], 10, 64)
				f, err2 := strconv.ParseUint(parts[1], 10, 64)
				if err1 == nil && err2 == nil {
					ramTotal = t / 1024
					ramFree = f / 1024
				}
			} else if len(parts) == 1 {
				if cpu, err := strconv.ParseFloat(parts[0], 64); err == nil {
					cpuVal = cpu
				}
			}
		}
	}

	return SysInfoState{
		Hostname:     hostname,
		OS:           "Windows",
		CPUUsage:     math.Round(cpuVal*100) / 100,
		RAMTotal:     ramTotal,
		RAMFree:      ramFree,
		BatteryPct:   batteryPct,
		IsCharging:   isCharging,
		MonitorCount: int(monitorsCount),
	}, nil
}
