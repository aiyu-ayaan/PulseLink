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

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
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

// ── Monitor enumeration ───────────────────────────────────────────────

var (
	enumMu    sync.Mutex
	enumOut   []monitorHW
	enumOnce  sync.Once
	enumCbPtr uintptr
)

// enumerateMonitors returns all active monitors in order. The callback is
// created exactly once: syscall.NewCallback allocations are permanent and
// capped process-wide, so creating one per call leaks until a panic.
func enumerateMonitors() []monitorHW {
	enumOnce.Do(func() {
		enumCbPtr = syscall.NewCallback(func(hMonitor, hdcMonitor, lprcMonitor, dwData uintptr) uintptr {
			var mi MONITORINFOEXW
			mi.Size = uint32(unsafe.Sizeof(mi))
			ret, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))
			if ret != 0 {
				enumOut = append(enumOut, monitorHW{
					hMonitor:   hMonitor,
					deviceName: syscall.UTF16ToString(mi.Device[:]),
					isPrimary:  mi.Flags&1 != 0,
				})
			}
			return 1
		})
	})
	enumMu.Lock()
	defer enumMu.Unlock()
	enumOut = nil
	procEnumDisplayMonitors.Call(0, 0, enumCbPtr, 0)
	out := make([]monitorHW, len(enumOut))
	copy(out, enumOut)
	return out
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

// ddcRaw maps a 0-100 percent onto the monitor's reported VCP brightness range.
func ddcRaw(level int, minVal, maxVal uint32) uint32 {
	if maxVal <= minVal {
		return uint32(level)
	}
	return minVal + uint32(level)*(maxVal-minVal)/100
}

// ddcGetBrightness reads brightness and the raw VCP range via DDC/CI with a
// 2-second timeout. level is normalised to 0-100.
func ddcGetBrightness(hMonitor uintptr) (level int, minVal, maxVal uint32, ok bool) {
	type result struct {
		level    int
		min, max uint32
		ok       bool
	}
	ch := make(chan result, 1)
	go func() {
		l, mn, mx, ok := ddcGetBrightnessSync(hMonitor)
		ch <- result{l, mn, mx, ok}
	}()
	select {
	case r := <-ch:
		return r.level, r.min, r.max, r.ok
	case <-time.After(2 * time.Second):
		return 0, 0, 0, false
	}
}

func ddcGetBrightnessSync(hMonitor uintptr) (int, uint32, uint32, bool) {
	var numPhysical uint32
	ret, _, _ := procGetNumberOfPhysicalMonitorsFromHMONITOR.Call(
		hMonitor, uintptr(unsafe.Pointer(&numPhysical)),
	)
	if ret == 0 || numPhysical == 0 {
		return 0, 0, 0, false
	}
	pms := make([]PHYSICAL_MONITOR, numPhysical)
	ret, _, _ = procGetPhysicalMonitorsFromHMONITOR.Call(
		hMonitor, uintptr(numPhysical), uintptr(unsafe.Pointer(&pms[0])),
	)
	if ret == 0 {
		return 0, 0, 0, false
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
			return pct, minVal, maxVal, true
		}
	}
	return 0, 0, 0, false
}

// ddcSetBrightness sets brightness via DDC/CI with a 2-second timeout. The
// second return distinguishes a timeout (outcome unknown, the late write may
// still land) from an outright failure.
func ddcSetBrightness(hMonitor uintptr, level int, minVal, maxVal uint32) (ok bool, timedOut bool) {
	ch := make(chan bool, 1)
	go func() {
		ch <- ddcSetBrightnessSync(hMonitor, level, minVal, maxVal)
	}()
	select {
	case ok := <-ch:
		return ok, false
	case <-time.After(2 * time.Second):
		return false, true
	}
}

func ddcSetBrightnessSync(hMonitor uintptr, level int, minVal, maxVal uint32) bool {
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

	raw := ddcRaw(level, minVal, maxVal)
	ok := false
	for _, pm := range pms {
		r, _, _ := procSetMonitorBrightness.Call(pm.HPhysicalMonitor, uintptr(raw))
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

// ── Probe (detects the working method per monitor) ────────────────────

// probeMonitors tests each method per monitor and caches which one works so
// subsequent get/set calls are instant. Hardware reads run outside the lock —
// the WMI query alone can take seconds.
func (s *Service) probeMonitors() {
	monitors := enumerateMonitors()
	if len(monitors) == 0 {
		return
	}

	wmiLevel, hasWMI := wmiGetBrightness()
	if hasWMI {
		s.log.Info("WMI brightness available (laptop built-in panel)")
	}
	maxComp := s.cfg().MaxCompatibilityMode

	type probed struct {
		method   string
		level    int
		min, max uint32
	}
	infos := make([]probed, len(monitors))
	for i, mon := range monitors {
		if !maxComp {
			if level, mn, mx, ok := ddcGetBrightness(mon.hMonitor); ok {
				infos[i] = probed{method: "ddc", level: level, min: mn, max: mx}
			}
		}
	}
	// WMI drives the built-in backlight. It belongs to the first display that
	// DDC/CI cannot reach (laptop panels don't speak DDC), regardless of which
	// monitor Windows calls primary. Everything else falls back to gamma.
	wmiTaken := false
	for i := range infos {
		if infos[i].method != "" {
			continue
		}
		if hasWMI && !wmiTaken {
			infos[i] = probed{method: "wmi", level: wmiLevel}
			wmiTaken = true
		} else {
			infos[i] = probed{method: "gamma", level: -1} // level resolved from cache below
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.methodCache = make(map[string]string)
	s.ddcRange = make(map[string][2]uint32)
	var result []MonitorBrightness
	for i, mon := range monitors {
		id := fmt.Sprintf("monitor-%d", i)
		info := infos[i]
		level := info.level
		if level < 0 { // gamma has no readback; keep the last level we applied
			if v, ok := s.lastLevels[id]; ok {
				level = v
			} else {
				level = 100
			}
		}
		s.methodCache[id] = info.method
		if info.method == "ddc" {
			s.ddcRange[id] = [2]uint32{info.min, info.max}
		}
		s.lastLevels[id] = level
		result = append(result, MonitorBrightness{
			ID:     id,
			Name:   monitorLabel(mon, i),
			Level:  level,
			Method: info.method,
		})
	}
	s.lastMonitors = result
	s.monitorHW = monitors
	s.probed = true
	s.log.Info("brightness probe complete", "monitors", len(result))
}

// ensureProbed runs the probe once (or again after settings invalidate it).
func (s *Service) ensureProbed() {
	s.probeMu.Lock()
	defer s.probeMu.Unlock()
	s.mu.Lock()
	done := s.probed
	s.mu.Unlock()
	if !done {
		s.probeMonitors()
	}
}

// maybeRefresh re-reads hardware levels in the background (throttled) so the
// UI reflects changes made on the monitor itself, and re-probes when displays
// were added or removed. Publishes brightness.changed when anything moved.
func (s *Service) maybeRefresh() {
	s.mu.Lock()
	if s.refreshing || time.Since(s.lastRefresh) < 5*time.Second {
		s.mu.Unlock()
		return
	}
	s.refreshing = true
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			s.refreshing = false
			s.lastRefresh = time.Now()
			s.mu.Unlock()
		}()

		monitors := enumerateMonitors()
		s.mu.Lock()
		countChanged := len(monitors) != len(s.monitorHW)
		s.mu.Unlock()

		changed := false
		if countChanged {
			s.log.Info("display configuration changed, re-probing")
			s.probeMu.Lock()
			s.probeMonitors()
			s.probeMu.Unlock()
			changed = true
		} else {
			for i := range monitors {
				id := fmt.Sprintf("monitor-%d", i)
				s.mu.Lock()
				// HMONITOR handles go stale across display mode changes;
				// refresh them while we have a current enumeration.
				s.monitorHW[i].hMonitor = monitors[i].hMonitor
				method := s.methodCache[id]
				busy := s.inFlight[id]
				last := s.lastLevels[id]
				s.mu.Unlock()
				if busy {
					continue // don't fight an in-flight write
				}
				var level int
				var ok bool
				switch method {
				case "ddc":
					level, _, _, ok = ddcGetBrightness(monitors[i].hMonitor)
				case "wmi":
					level, ok = wmiGetBrightness()
				}
				if ok && level != last {
					s.mu.Lock()
					s.lastLevels[id] = level
					for j := range s.lastMonitors {
						if s.lastMonitors[j].ID == id {
							s.lastMonitors[j].Level = level
						}
					}
					s.mu.Unlock()
					changed = true
				}
			}
		}
		if changed {
			s.publishState()
		}
	}()
}

func (s *Service) publishState() {
	s.mu.Lock()
	state := BrightnessState{Monitors: append([]MonitorBrightness(nil), s.lastMonitors...)}
	s.mu.Unlock()
	s.bus.Publish(eventbus.Event{Topic: "brightness.changed", Payload: state})
}

// ── Public API ────────────────────────────────────────────────────────

func (s *Service) GetBrightness() (any, error) {
	s.ensureProbed()
	s.maybeRefresh()
	s.mu.Lock()
	defer s.mu.Unlock()
	return BrightnessState{Monitors: append([]MonitorBrightness(nil), s.lastMonitors...)}, nil
}

func (s *Service) SetBrightness(monitorID string, level int) error {
	s.log.Info("brightness set", "monitor", monitorID, "level", level)
	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}
	s.ensureProbed()

	s.mu.Lock()
	defer s.mu.Unlock()

	var targets []int
	if monitorID == "all" {
		for i := range s.monitorHW {
			targets = append(targets, i)
		}
	} else {
		targetIdx := -1
		for i := range s.monitorHW {
			if fmt.Sprintf("monitor-%d", i) == monitorID {
				targetIdx = i
				break
			}
		}
		if targetIdx < 0 {
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
		if targetIdx >= 0 {
			targets = append(targets, targetIdx)
		}
	}

	for _, idx := range targets {
		id := fmt.Sprintf("monitor-%d", idx)
		// Update the cache immediately so the response is instant.
		s.lastLevels[id] = level
		for j := range s.lastMonitors {
			if s.lastMonitors[j].ID == id {
				s.lastMonitors[j].Level = level
				break
			}
		}
		// Latest-wins queue: one worker per monitor serialises the slow
		// DDC/WMI bus and coalesces slider spam down to the newest value.
		s.pending[id] = level
		if !s.inFlight[id] {
			s.inFlight[id] = true
			go s.applyWorker(idx, id)
		}
	}
	return nil
}

// applyWorker drains pending levels for one monitor, applying the newest value
// each round. DDC/CI is a slow serial bus — concurrent writes stall or corrupt
// it, so all hardware access for a monitor funnels through here.
func (s *Service) applyWorker(idx int, id string) {
	for {
		s.mu.Lock()
		level, ok := s.pending[id]
		if !ok || idx >= len(s.monitorHW) {
			delete(s.pending, id)
			s.inFlight[id] = false
			s.mu.Unlock()
			return
		}
		delete(s.pending, id)
		method := s.methodCache[id]
		mon := s.monitorHW[idx]
		rng, hasRng := s.ddcRange[id]
		gammaDim := s.gammaDim[id]
		s.mu.Unlock()

		hwOK, timedOut := false, false
		switch method {
		case "wmi":
			hwOK = wmiSetBrightness(level)
		case "ddc":
			mn, mx := uint32(0), uint32(100)
			if hasRng {
				mn, mx = rng[0], rng[1]
			}
			hwOK, timedOut = ddcSetBrightness(mon.hMonitor, level, mn, mx)
		}

		switch {
		case hwOK:
			s.log.Info("brightness applied", "monitor", id, "level", level, "method", method)
			if gammaDim {
				// A previous fallback dimmed via gamma; undo it now that the
				// real backlight responds again, or the screen stays dark.
				setGammaRamp(mon.deviceName, 100)
				s.mu.Lock()
				s.gammaDim[id] = false
				s.mu.Unlock()
			}
		case method == "gamma":
			if setGammaRamp(mon.deviceName, level) {
				s.mu.Lock()
				s.gammaDim[id] = level < 100
				s.mu.Unlock()
				s.log.Info("brightness applied via gamma ramp", "monitor", id, "level", level)
			} else {
				s.log.Warn("gamma ramp set failed", "monitor", id)
			}
		case timedOut:
			// Outcome unknown — the late DDC write may still land. Don't stack
			// a gamma dim on top of it.
			s.log.Warn("hardware brightness set timed out", "monitor", id, "method", method)
		default:
			s.log.Warn("hardware brightness set failed, falling back to gamma", "monitor", id, "method", method)
			if setGammaRamp(mon.deviceName, level) {
				s.mu.Lock()
				s.gammaDim[id] = level < 100
				s.mu.Unlock()
			}
		}
	}
}

func monitorLabel(mon monitorHW, idx int) string {
	label := fmt.Sprintf("Display %d", idx+1)
	if mon.isPrimary {
		label += " (Primary)"
	}
	return label
}
