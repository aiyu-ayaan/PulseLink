//go:build windows

package brightness

import (
	"fmt"
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

// monitorHW holds information about a single display.
type monitorHW struct {
	hMonitor   uintptr
	deviceName string // e.g. `\\.\DISPLAY1`
	isPrimary  bool
}

// enumerateMonitors returns all active monitors in order.
func enumerateMonitors() []monitorHW {
	var result []monitorHW
	callback := syscall.NewCallback(func(hMonitor, hdcMonitor, lprcMonitor, dwData uintptr) uintptr {
		var mi MONITORINFOEXW
		mi.Size = uint32(unsafe.Sizeof(mi))
		ret, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))
		if ret != 0 {
			name := syscall.UTF16ToString(mi.Device[:])
			result = append(result, monitorHW{
				hMonitor:   hMonitor,
				deviceName: name,
				isPrimary:  mi.Flags&1 != 0, // MONITORINFOF_PRIMARY
			})
		}
		return 1
	})
	procEnumDisplayMonitors.Call(0, 0, callback, 0)
	return result
}

// tryDDCGetBrightness attempts to read brightness via DDC/CI. Returns level (0-100) and ok.
func tryDDCGetBrightness(hMonitor uintptr) (int, bool) {
	var numPhysical uint32
	ret, _, _ := procGetNumberOfPhysicalMonitorsFromHMONITOR.Call(
		hMonitor, uintptr(unsafe.Pointer(&numPhysical)),
	)
	if ret == 0 || numPhysical == 0 {
		return 0, false
	}
	pms := make([]PHYSICAL_MONITOR, numPhysical)
	ret, _, _ = procGetPhysicalMonitorsFromHMONITOR.Call(
		hMonitor, uintptr(numPhysical), uintptr(unsafe.Pointer(&pms[0])),
	)
	if ret == 0 {
		return 0, false
	}
	defer procDestroyPhysicalMonitors.Call(uintptr(numPhysical), uintptr(unsafe.Pointer(&pms[0])))

	for _, pm := range pms {
		var minVal, curVal, maxVal uint32
		r, _, _ := procGetMonitorBrightness.Call(
			pm.HPhysicalMonitor,
			uintptr(unsafe.Pointer(&minVal)),
			uintptr(unsafe.Pointer(&curVal)),
			uintptr(unsafe.Pointer(&maxVal)),
		)
		if r != 0 && maxVal > minVal {
			pct := int((curVal - minVal) * 100 / (maxVal - minVal))
			return pct, true
		}
	}
	return 0, false
}

// tryDDCSetBrightness attempts to set brightness via DDC/CI. Returns true on success.
func tryDDCSetBrightness(hMonitor uintptr, level int) bool {
	var numPhysical uint32
	ret, _, _ := procGetNumberOfPhysicalMonitorsFromHMONITOR.Call(
		hMonitor, uintptr(unsafe.Pointer(&numPhysical)),
	)
	if ret == 0 || numPhysical == 0 {
		return false
	}
	pms := make([]PHYSICAL_MONITOR, numPhysical)
	ret, _, _ = procGetPhysicalMonitorsFromHMONITOR.Call(
		hMonitor, uintptr(numPhysical), uintptr(unsafe.Pointer(&pms[0])),
	)
	if ret == 0 {
		return false
	}
	defer procDestroyPhysicalMonitors.Call(uintptr(numPhysical), uintptr(unsafe.Pointer(&pms[0])))

	ok := false
	for _, pm := range pms {
		r, _, _ := procSetMonitorBrightness.Call(pm.HPhysicalMonitor, uintptr(level))
		if r != 0 {
			ok = true
		}
	}
	return ok
}

// setGammaRamp adjusts the software gamma ramp on a display to simulate brightness.
func setGammaRamp(deviceName string, level int) bool {
	namePtr, err := syscall.UTF16PtrFromString(deviceName)
	if err != nil {
		return false
	}
	hdc, _, _ := procCreateDCW.Call(0, uintptr(unsafe.Pointer(namePtr)), 0, 0)
	if hdc == 0 {
		return false
	}
	defer procDeleteDC.Call(hdc)

	var ramp GammaRamp
	clamped := level
	if clamped < 5 {
		clamped = 5
	}
	for i := 0; i < 256; i++ {
		val := uint32(i) * 257 * uint32(clamped) / 100
		if val > 65535 {
			val = 65535
		}
		ramp.Red[i] = uint16(val)
		ramp.Green[i] = uint16(val)
		ramp.Blue[i] = uint16(val)
	}
	ret, _, _ := procSetDeviceGammaRamp.Call(hdc, uintptr(unsafe.Pointer(&ramp)))
	return ret != 0
}

// getGammaLevel reads the current gamma ramp and infers a brightness percentage.
func getGammaLevel(deviceName string) (int, bool) {
	namePtr, err := syscall.UTF16PtrFromString(deviceName)
	if err != nil {
		return 0, false
	}
	hdc, _, _ := procCreateDCW.Call(0, uintptr(unsafe.Pointer(namePtr)), 0, 0)
	if hdc == 0 {
		return 0, false
	}
	defer procDeleteDC.Call(hdc)

	var ramp GammaRamp
	ret, _, _ := procGetDeviceGammaRamp.Call(hdc, uintptr(unsafe.Pointer(&ramp)))
	if ret == 0 {
		return 0, false
	}
	mid := int(ramp.Red[128])
	pct := mid * 100 / 32896
	if pct > 100 {
		pct = 100
	}
	if pct < 0 {
		pct = 0
	}
	return pct, true
}

func (s *Service) GetBrightness() (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	monitors := enumerateMonitors()
	if len(monitors) == 0 {
		return BrightnessState{Monitors: s.lastMonitors}, nil
	}

	maxComp := s.cfg().MaxCompatibilityMode
	var result []MonitorBrightness

	for i, mon := range monitors {
		mb := MonitorBrightness{
			ID:   fmt.Sprintf("monitor-%d", i),
			Name: monitorLabel(mon, i),
		}

		if !maxComp {
			if level, ok := tryDDCGetBrightness(mon.hMonitor); ok {
				mb.Level = level
				mb.Method = "ddc"
				result = append(result, mb)
				continue
			}
		}

		// Fallback: read software gamma ramp
		if level, ok := getGammaLevel(mon.deviceName); ok {
			mb.Level = level
			mb.Method = "gamma"
		} else {
			mb.Level = s.getLastLevel(mb.ID)
			mb.Method = "gamma"
		}
		result = append(result, mb)
	}

	s.lastMonitors = result
	return BrightnessState{Monitors: result}, nil
}

func (s *Service) SetBrightness(monitorID string, level int) error {
	s.log.Info("brightness set", "monitor", monitorID, "level", level)
	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	monitors := enumerateMonitors()
	maxComp := s.cfg().MaxCompatibilityMode

	targetIdx := -1
	for i := range monitors {
		id := fmt.Sprintf("monitor-%d", i)
		if id == monitorID {
			targetIdx = i
			break
		}
	}

	applyAll := monitorID == "all"

	if targetIdx < 0 && !applyAll {
		switch strings.ToLower(monitorID) {
		case "internal", "":
			if len(monitors) > 0 {
				targetIdx = 0
			}
		case "external":
			if len(monitors) > 1 {
				targetIdx = 1
			} else if len(monitors) > 0 {
				targetIdx = 0
			}
		default:
			return fmt.Errorf("unknown monitor: %s", monitorID)
		}
	}

	apply := func(idx int) {
		mon := monitors[idx]
		id := fmt.Sprintf("monitor-%d", idx)

		if !maxComp {
			if tryDDCSetBrightness(mon.hMonitor, level) {
				s.setLastLevel(id, level)
				s.log.Info("brightness set via DDC/CI", "monitor", id, "level", level)
				return
			}
		}

		if setGammaRamp(mon.deviceName, level) {
			s.setLastLevel(id, level)
			s.log.Info("brightness set via gamma ramp", "monitor", id, "level", level)
		} else {
			s.log.Warn("failed to set brightness", "monitor", id)
		}
	}

	if applyAll {
		for i := range monitors {
			apply(i)
		}
	} else if targetIdx >= 0 && targetIdx < len(monitors) {
		apply(targetIdx)
	}

	return nil
}

func monitorLabel(mon monitorHW, idx int) string {
	label := fmt.Sprintf("Display %d", idx+1)
	if mon.isPrimary {
		label += " (Primary)"
	}
	return label
}
