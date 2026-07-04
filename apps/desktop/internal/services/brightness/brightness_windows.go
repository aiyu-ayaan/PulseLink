//go:build windows

package brightness

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
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
	procSetDeviceGammaRamp = gdi32.NewProc("SetDeviceGammaRamp")
)

// monitorHW holds information about a single display.
type monitorHW struct {
	hMonitor   uintptr
	deviceName string
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
				isPrimary:  mi.Flags&1 != 0,
			})
		}
		return 1
	})
	procEnumDisplayMonitors.Call(0, 0, callback, 0)
	return result
}

// ── WMI helpers (for laptop built-in panels) ──────────────────────────

// wmiGetBrightness reads the built-in display brightness via WMI.
// Returns level 0-100 and true, or 0/false if WMI is unavailable (desktop PC).
func wmiGetBrightness() (int, bool) {
	type result struct {
		level int
		ok    bool
	}
	ch := make(chan result, 1)
	go func() {
		cmd := `(Get-CimInstance -Namespace root/WMI -ClassName WmiMonitorBrightness -ErrorAction SilentlyContinue).CurrentBrightness`
		out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", cmd).Output()
		if err != nil {
			ch <- result{0, false}
			return
		}
		val, err := strconv.Atoi(strings.TrimSpace(string(out)))
		if err != nil {
			ch <- result{0, false}
			return
		}
		ch <- result{val, true}
	}()
	select {
	case r := <-ch:
		return r.level, r.ok
	case <-time.After(3 * time.Second):
		return 0, false
	}
}

// wmiSetBrightness sets the built-in display brightness via WMI.
// Returns true on success, false if WMI is unavailable.
func wmiSetBrightness(level int) bool {
	ch := make(chan bool, 1)
	go func() {
		cmd := fmt.Sprintf(
			`$m = Get-CimInstance -Namespace root/WMI -ClassName WmiMonitorBrightnessMethods -ErrorAction SilentlyContinue; if ($m) { $m | Invoke-CimMethod -MethodName WmiSetBrightness -Arguments @{Timeout=1; Brightness=%d} | Out-Null; $true } else { $false }`,
			level)
		out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", cmd).Output()
		if err != nil {
			ch <- false
			return
		}
		ch <- strings.TrimSpace(string(out)) == "True"
	}()
	select {
	case ok := <-ch:
		return ok
	case <-time.After(3 * time.Second):
		return false
	}
}

// ── DDC/CI helpers (with goroutine timeout) ───────────────────────────

// ddcGetBrightness reads brightness via DDC/CI with a 2-second timeout.
func ddcGetBrightness(hMonitor uintptr) (int, bool) {
	type result struct {
		level int
		ok    bool
	}
	ch := make(chan result, 1)
	go func() {
		l, ok := ddcGetBrightnessSync(hMonitor)
		ch <- result{l, ok}
	}()
	select {
	case r := <-ch:
		return r.level, r.ok
	case <-time.After(2 * time.Second):
		return 0, false
	}
}

func ddcGetBrightnessSync(hMonitor uintptr) (int, bool) {
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

// ddcSetBrightness sets brightness via DDC/CI with a 2-second timeout.
func ddcSetBrightness(hMonitor uintptr, level int) bool {
	ch := make(chan bool, 1)
	go func() {
		ch <- ddcSetBrightnessSync(hMonitor, level)
	}()
	select {
	case ok := <-ch:
		return ok
	case <-time.After(2 * time.Second):
		return false
	}
}

func ddcSetBrightnessSync(hMonitor uintptr, level int) bool {
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

// ── Gamma ramp helpers ────────────────────────────────────────────────

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

// ── Probe (runs once at startup to detect capabilities) ───────────────

// probeResult records what method works for each monitor.
type probeResult struct {
	method string // "wmi", "ddc", "gamma"
	level  int
}

// probeMonitors runs once at startup. It tests each method per monitor
// and caches which one works so subsequent get/set calls are instant.
func (s *Service) probeMonitors() {
	monitors := enumerateMonitors()
	if len(monitors) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// First check if WMI is available (laptop built-in display).
	hasWMI := false
	wmiLevel := 0
	if lvl, ok := wmiGetBrightness(); ok {
		hasWMI = true
		wmiLevel = lvl
		s.log.Info("WMI brightness available (laptop built-in display)")
	}

	maxComp := s.cfg().MaxCompatibilityMode
	var result []MonitorBrightness
	s.methodCache = make(map[string]string)

	for i, mon := range monitors {
		id := fmt.Sprintf("monitor-%d", i)
		mb := MonitorBrightness{
			ID:   id,
			Name: monitorLabel(mon, i),
		}

		// For the primary monitor on a laptop, WMI controls the real backlight
		if hasWMI && mon.isPrimary {
			mb.Level = wmiLevel
			mb.Method = "wmi"
			s.methodCache[id] = "wmi"
			s.lastLevels[id] = wmiLevel
			result = append(result, mb)
			continue
		}

		// Try DDC/CI (with timeout)
		if !maxComp {
			if level, ok := ddcGetBrightness(mon.hMonitor); ok {
				mb.Level = level
				mb.Method = "ddc"
				s.methodCache[id] = "ddc"
				s.lastLevels[id] = level
				result = append(result, mb)
				continue
			}
		}

		// Fallback: gamma ramp
		mb.Level = 100
		mb.Method = "gamma"
		s.methodCache[id] = "gamma"
		s.lastLevels[id] = 100
		result = append(result, mb)
	}

	s.lastMonitors = result
	s.monitorHW = monitors
	s.probed = true
	s.log.Info("brightness probe complete", "monitors", len(result))
}

// ── Public API ────────────────────────────────────────────────────────

func (s *Service) GetBrightness() (any, error) {
	if !s.probed {
		s.probeMonitors()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return BrightnessState{Monitors: s.lastMonitors}, nil
}

func (s *Service) SetBrightness(monitorID string, level int) error {
	s.log.Info("brightness set", "monitor", monitorID, "level", level)
	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}

	if !s.probed {
		s.probeMonitors()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	applyAll := monitorID == "all"

	// Resolve target index
	targetIdx := -1
	for i := range s.monitorHW {
		if fmt.Sprintf("monitor-%d", i) == monitorID {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 && !applyAll {
		switch strings.ToLower(monitorID) {
		case "internal", "":
			if len(s.monitorHW) > 0 {
				targetIdx = 0
			}
		case "external":
			if len(s.monitorHW) > 1 {
				targetIdx = 1
			} else if len(s.monitorHW) > 0 {
				targetIdx = 0
			}
		default:
			return fmt.Errorf("unknown monitor: %s", monitorID)
		}
	}

	apply := func(idx int) {
		id := fmt.Sprintf("monitor-%d", idx)
		method := s.methodCache[id]
		mon := s.monitorHW[idx]

		switch method {
		case "wmi":
			// Run WMI in background — don't block the response
			go func() {
				if wmiSetBrightness(level) {
					s.log.Info("brightness set via WMI", "monitor", id, "level", level)
				} else {
					s.log.Warn("WMI set brightness failed, trying gamma", "monitor", id)
					setGammaRamp(mon.deviceName, level)
				}
			}()
		case "ddc":
			// Run DDC/CI in background — don't block the response
			go func() {
				if ddcSetBrightness(mon.hMonitor, level) {
					s.log.Info("brightness set via DDC/CI", "monitor", id, "level", level)
				} else {
					s.log.Warn("DDC/CI set failed, trying gamma", "monitor", id)
					setGammaRamp(mon.deviceName, level)
				}
			}()
		default:
			// Gamma is instant — do it inline
			if setGammaRamp(mon.deviceName, level) {
				s.log.Info("brightness set via gamma ramp", "monitor", id, "level", level)
			} else {
				s.log.Warn("gamma ramp set failed", "monitor", id)
			}
		}

		// Update cache immediately so the response is instant
		s.lastLevels[id] = level
		for j := range s.lastMonitors {
			if s.lastMonitors[j].ID == id {
				s.lastMonitors[j].Level = level
				break
			}
		}
	}

	if applyAll {
		for i := range s.monitorHW {
			apply(i)
		}
	} else if targetIdx >= 0 && targetIdx < len(s.monitorHW) {
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

// probeResultCache is used by concurrent probing.
var probeOnce sync.Once
