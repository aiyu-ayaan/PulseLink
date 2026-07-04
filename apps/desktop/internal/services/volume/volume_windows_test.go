//go:build windows

package volume

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"runtime"
	"testing"
	"unsafe"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

func TestCombinedVolumeSetting(t *testing.T) {
	runtime.LockOSThread()
	
	// 1. Direct call test
	func() {
		if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
			t.Fatalf("CoInitializeEx failed: %v", err)
		}
		defer ole.CoUninitialize()

		var mmde *wca.IMMDeviceEnumerator
		if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
			t.Fatalf("CoCreateInstance failed: %v", err)
		}
		defer mmde.Release()

		var mmd *wca.IMMDevice
		if err := mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmd); err != nil {
			t.Fatalf("GetDefaultAudioEndpoint failed: %v", err)
		}
		defer mmd.Release()

		var deviceID string
		if err := mmd.GetId(&deviceID); err == nil {
			fmt.Printf("COMBINED TEST - Direct call device ID: %s\n", deviceID)
		}

		var aev *wca.IAudioEndpointVolume
		if err := mmd.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &aev); err != nil {
			t.Fatalf("Activate failed: %v", err)
		}
		defer aev.Release()

		targetScalar := float32(0.2)
		hr := callSetVolumeScalar(
			aev.VTable().SetMasterVolumeLevelScalar,
			uintptr(unsafe.Pointer(aev)),
			math.Float32bits(targetScalar),
			0,
		)
		if hr != 0 {
			t.Fatalf("Direct call failed: HRESULT %X", hr)
		}
		var actualScalar float32
		if err := aev.GetMasterVolumeLevelScalar(&actualScalar); err != nil {
			t.Fatalf("GetMasterVolumeLevelScalar failed: %v", err)
		}
		var levelDB float32
		_ = aev.GetMasterVolumeLevel(&levelDB)
		fmt.Printf("COMBINED TEST - Direct call set to 0.2, read back scalar: %v, dB: %v\n", actualScalar, levelDB)
	}()

	// 2. Service call test
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	bus := eventbus.New()
	svc := New(log, bus)

	st, err := svc.SetVolume(20)
	if err != nil {
		t.Fatalf("Service SetVolume failed: %v", err)
	}
	
	// Read again using a fresh direct COM connection to see what was actually set on the PC
	func() {
		if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
			t.Fatalf("CoInitializeEx failed: %v", err)
		}
		defer ole.CoUninitialize()

		var mmde *wca.IMMDeviceEnumerator
		if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
			t.Fatalf("CoCreateInstance failed: %v", err)
		}
		defer mmde.Release()

		var mmd *wca.IMMDevice
		if err := mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmd); err != nil {
			t.Fatalf("GetDefaultAudioEndpoint failed: %v", err)
		}
		defer mmd.Release()

		var deviceID string
		if err := mmd.GetId(&deviceID); err == nil {
			fmt.Printf("COMBINED TEST - Direct read back device ID: %s\n", deviceID)
		}

		var aev *wca.IAudioEndpointVolume
		if err := mmd.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &aev); err != nil {
			t.Fatalf("Activate failed: %v", err)
		}
		defer aev.Release()

		var actualScalar float32
		_ = aev.GetMasterVolumeLevelScalar(&actualScalar)
		var levelDB float32
		_ = aev.GetMasterVolumeLevel(&levelDB)
		fmt.Printf("COMBINED TEST - After Service SetVolume(20), direct read back scalar: %v, dB: %v (service returned Level: %d)\n", actualScalar, levelDB, st.Level)
	}()

	runtime.UnlockOSThread()
}
