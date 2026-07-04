//go:build windows

package brightness

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

type RECT struct {
	Left, Top, Right, Bottom int32
}

type MONITORINFOEXW struct {
	Size      uint32
	RcMonitor RECT
	RcWork    RECT
	Flags     uint32
	Device    [32]uint16
}

type PHYSICAL_MONITOR struct {
	HPhysicalMonitor uintptr
	Description      [128]uint16
}

type GammaRamp struct {
	Red   [256]uint16
	Green [256]uint16
	Blue  [256]uint16
}

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	dxva2  = syscall.NewLazyDLL("dxva2.dll")
	gdi32  = syscall.NewLazyDLL("gdi32.dll")

	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW     = user32.NewProc("GetMonitorInfoW")

	procGetNumberOfPhysicalMonitorsFromHMONITOR = dxva2.NewProc("GetNumberOfPhysicalMonitorsFromHMONITOR")
	procGetPhysicalMonitorsFromHMONITOR         = dxva2.NewProc("GetPhysicalMonitorsFromHMONITOR")
	procDestroyPhysicalMonitors                 = dxva2.NewProc("DestroyPhysicalMonitors")
	procGetMonitorBrightness                    = dxva2.NewProc("GetMonitorBrightness")
	procSetMonitorBrightness                    = dxva2.NewProc("SetMonitorBrightness")

	procCreateDCW          = gdi32.NewProc("CreateDCW")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procGetDeviceGammaRamp = gdi32.NewProc("GetDeviceGammaRamp")
	procSetDeviceGammaRamp = gdi32.NewProc("SetDeviceGammaRamp")
)

func getMonitors() []uintptr {
	var monitors []uintptr
	callback := syscall.NewCallback(func(hMonitor uintptr, hdcMonitor uintptr, lprcMonitor uintptr, dwData uintptr) uintptr {
		monitors = append(monitors, hMonitor)
		return 1
	})
	procEnumDisplayMonitors.Call(0, 0, callback, 0)
	return monitors
}

func (s *Service) GetBrightness() (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Get internal screen brightness via WMI
	internalLevel := s.lastInternalLevel
	cmd := "Get-CimInstance -Namespace root/WMI -ClassName WmiMonitorBrightness | Select-Object -ExpandProperty CurrentBrightness"
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", cmd).Output()
	if err == nil {
		valStr := strings.TrimSpace(string(out))
		if val, err := strconv.Atoi(valStr); err == nil {
			internalLevel = val
			s.lastInternalLevel = val
		}
	} else {
		s.log.Debug("could not query built-in monitor brightness (likely a desktop PC)")
	}

	// 2. Get external screen brightness
	externalLevel := s.lastExternalLevel
	
	// Query external display brightness via DDC/CI (if compatibility mode is disabled)
	if !s.cfg().MaxCompatibilityMode {
		monitors := getMonitors()
		for _, hMonitor := range monitors {
			var numPhysicalMonitors uint32
			ret, _, _ := procGetNumberOfPhysicalMonitorsFromHMONITOR.Call(
				hMonitor,
				uintptr(unsafe.Pointer(&numPhysicalMonitors)),
			)
			if ret != 0 && numPhysicalMonitors > 0 {
				physicalMonitors := make([]PHYSICAL_MONITOR, numPhysicalMonitors)
				ret, _, _ = procGetPhysicalMonitorsFromHMONITOR.Call(
					hMonitor,
					uintptr(numPhysicalMonitors),
					uintptr(unsafe.Pointer(&physicalMonitors[0])),
				)
				if ret != 0 {
					for _, pm := range physicalMonitors {
						var minVal, curVal, maxVal uint32
						retVal, _, _ := procGetMonitorBrightness.Call(
							pm.HPhysicalMonitor,
							uintptr(unsafe.Pointer(&minVal)),
							uintptr(unsafe.Pointer(&curVal)),
							uintptr(unsafe.Pointer(&maxVal)),
						)
						if retVal != 0 {
							externalLevel = int(curVal)
							s.lastExternalLevel = int(curVal)
							break
						}
					}
					procDestroyPhysicalMonitors.Call(
						uintptr(numPhysicalMonitors),
						uintptr(unsafe.Pointer(&physicalMonitors[0])),
					)
				}
			}
		}
	}

	return BrightnessState{
		Internal: internalLevel,
		External: externalLevel,
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

	s.mu.Lock()
	if targetType == "internal" || targetType == "" {
		s.lastInternalLevel = level
	} else {
		s.lastExternalLevel = level
	}
	s.mu.Unlock()

	if targetType == "internal" || targetType == "" {
		cmd := fmt.Sprintf("(Get-CimInstance -Namespace root/WMI -ClassName WmiMonitorBrightnessMethods).WmiSetBrightness(1, %d)", level)
		err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", cmd).Run()
		if err != nil {
			return fmt.Errorf("powershell set brightness: %w", err)
		}
	} else {
		// Set external display brightness
		maxComp := s.cfg().MaxCompatibilityMode
		ddcSuccess := false

		if !maxComp {
			// Try DDC/CI
			monitors := getMonitors()
			for _, hMonitor := range monitors {
				var numPhysicalMonitors uint32
				ret, _, _ := procGetNumberOfPhysicalMonitorsFromHMONITOR.Call(
					hMonitor,
					uintptr(unsafe.Pointer(&numPhysicalMonitors)),
				)
				if ret != 0 && numPhysicalMonitors > 0 {
					physicalMonitors := make([]PHYSICAL_MONITOR, numPhysicalMonitors)
					ret, _, _ = procGetPhysicalMonitorsFromHMONITOR.Call(
						hMonitor,
						uintptr(numPhysicalMonitors),
						uintptr(unsafe.Pointer(&physicalMonitors[0])),
					)
					if ret != 0 {
						for _, pm := range physicalMonitors {
							retVal, _, _ := procSetMonitorBrightness.Call(
								pm.HPhysicalMonitor,
								uintptr(level),
							)
							if retVal != 0 {
								ddcSuccess = true
							}
						}
						procDestroyPhysicalMonitors.Call(
							uintptr(numPhysicalMonitors),
							uintptr(unsafe.Pointer(&physicalMonitors[0])),
						)
					}
				}
			}
		}

		// Fallback to software Gamma Ramp if compatibility mode is enabled or DDC/CI failed
		if maxComp || !ddcSuccess {
			s.log.Info("using software gamma ramp for external brightness control")
			monitors := getMonitors()
			for _, hMonitor := range monitors {
				var mi MONITORINFOEXW
				mi.Size = uint32(unsafe.Sizeof(mi))
				ret, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))
				if ret != 0 {
					// Apply software dimming to the monitor
					deviceName := &mi.Device[0]
					hdc, _, _ := procCreateDCW.Call(
						0,
						uintptr(unsafe.Pointer(deviceName)),
						0,
						0,
					)
					if hdc != 0 {
						var ramp GammaRamp
						// Ensure it doesn't go below 10% brightness to keep the screen visible
						minVal := uint32(10)
						scaledLevel := minVal + (uint32(level) * (100 - minVal) / 100)
						for i := 0; i < 256; i++ {
							val := (uint32(i) * 257 * scaledLevel) / 100
							if val > 65535 {
								val = 65535
							}
							ramp.Red[i] = uint16(val)
							ramp.Green[i] = uint16(val)
							ramp.Blue[i] = uint16(val)
						}
						procSetDeviceGammaRamp.Call(hdc, uintptr(unsafe.Pointer(&ramp)))
						procDeleteDC.Call(hdc)
					}
				}
			}
		}
	}
	return nil
}
