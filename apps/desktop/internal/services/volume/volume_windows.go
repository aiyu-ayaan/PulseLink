//go:build windows

package volume

import (
	"math"
	"runtime"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

// withEndpointVolume opens the default render device's IAudioEndpointVolume,
// runs fn against it, and tears everything down. COM must be initialised on the
// same OS thread that uses the interface, so we pin the goroutine for the call.
// This is microsecond-cheap — no subprocess — so it is safe to call on every
// request and poll, unlike the old PowerShell approach.
func withEndpointVolume(fn func(*wca.IAudioEndpointVolume) error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		return err
	}
	defer ole.CoUninitialize()

	var mmde *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
		return err
	}
	defer mmde.Release()

	var mmd *wca.IMMDevice
	if err := mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmd); err != nil {
		return err
	}
	defer mmd.Release()

	var deviceID string
	if err := mmd.GetId(&deviceID); err == nil {
		println("DEBUG: withEndpointVolume device ID =", deviceID)
	}

	var aev *wca.IAudioEndpointVolume
	if err := mmd.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &aev); err != nil {
		return err
	}
	defer aev.Release()

	return fn(aev)
}

// Declare assembly function to invoke SetMasterVolumeLevelScalar with floating-point parameters mapped to XMM registers (Windows AMD64).
func callSetVolumeScalar(fn uintptr, aev uintptr, levelBits uint32, eventContextGUID uintptr) uintptr

// readState reads the current scalar level (0-1) and mute flag.
func readState(aev *wca.IAudioEndpointVolume) (VolumeState, error) {
	var scalar float32
	if err := aev.GetMasterVolumeLevelScalar(&scalar); err != nil {
		return VolumeState{}, err
	}
	var mutedVal int32
	if err := aev.GetMute((*bool)(unsafe.Pointer(&mutedVal))); err != nil {
		return VolumeState{}, err
	}
	println("DEBUG: readState aev =", uintptr(unsafe.Pointer(aev)), "scalar =", scalar)
	return VolumeState{Level: int(math.Round(float64(scalar) * 100)), Muted: mutedVal != 0}, nil
}

func (s *Service) GetVolume() (VolumeState, error) {
	var st VolumeState
	err := withEndpointVolume(func(aev *wca.IAudioEndpointVolume) error {
		var e error
		st, e = readState(aev)
		return e
	})
	return st, err
}

func (s *Service) SetVolume(level int) (VolumeState, error) {
	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}
	s.log.Info("volume action: set", "level", level)

	err := withEndpointVolume(func(aev *wca.IAudioEndpointVolume) error {
		targetScalar := float32(level) / 100
		println("DEBUG: SetVolume targetScalar =", targetScalar, "bits =", math.Float32bits(targetScalar))
		hr := callSetVolumeScalar(
			aev.VTable().SetMasterVolumeLevelScalar,
			uintptr(unsafe.Pointer(aev)),
			math.Float32bits(targetScalar),
			0,
		)
		if hr != 0 {
			return ole.NewError(hr)
		}
		return nil
	})
	if err != nil {
		return VolumeState{}, err
	}

	time.Sleep(200 * time.Millisecond)
	return s.GetVolume()
}

// step nudges the current scalar by delta, clamped to [0,1], and returns the
// resulting state. Used by up/down so the exact new level is reported back.
func (s *Service) step(delta float32) (VolumeState, error) {
	err := withEndpointVolume(func(aev *wca.IAudioEndpointVolume) error {
		var cur float32
		if e := aev.GetMasterVolumeLevelScalar(&cur); e != nil {
			return e
		}
		next := cur + delta
		if next < 0 {
			next = 0
		}
		if next > 1 {
			next = 1
		}
		hr := callSetVolumeScalar(
			aev.VTable().SetMasterVolumeLevelScalar,
			uintptr(unsafe.Pointer(aev)),
			math.Float32bits(next),
			0,
		)
		if hr != 0 {
			return ole.NewError(hr)
		}
		return nil
	})
	if err != nil {
		return VolumeState{}, err
	}

	time.Sleep(200 * time.Millisecond)
	return s.GetVolume()
}

func (s *Service) VolumeUp() (VolumeState, error) {
	s.log.Info("volume action: up")
	return s.step(0.02)
}

func (s *Service) VolumeDown() (VolumeState, error) {
	s.log.Info("volume action: down")
	return s.step(-0.02)
}

func (s *Service) VolumeMute() (VolumeState, error) {
	s.log.Info("volume action: mute")
	var st VolumeState
	err := withEndpointVolume(func(aev *wca.IAudioEndpointVolume) error {
		var mutedVal int32
		if e := aev.GetMute((*bool)(unsafe.Pointer(&mutedVal))); e != nil {
			return e
		}
		if e := aev.SetMute(mutedVal == 0, nil); e != nil {
			return e
		}
		var e error
		st, e = readState(aev)
		return e
	})
	return st, err
}
