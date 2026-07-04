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

func TestDirectCallSetVolumeScalar(t *testing.T) {
	levels := []int{0, 20, 50, 75, 100}
	for _, lvl := range levels {
		runtime.LockOSThread()
		
		if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
			t.Fatalf("CoInitializeEx failed: %v", err)
		}

		var mmde *wca.IMMDeviceEnumerator
		if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
			t.Fatalf("CoCreateInstance failed: %v", err)
		}

		var mmd *wca.IMMDevice
		if err := mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmd); err != nil {
			t.Fatalf("GetDefaultAudioEndpoint failed: %v", err)
		}

		var aev *wca.IAudioEndpointVolume
		if err := mmd.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &aev); err != nil {
			t.Fatalf("Activate failed: %v", err)
		}

		targetScalar := float32(lvl) / 100
		fnPtr := aev.VTable().SetMasterVolumeLevelScalar
		fmt.Printf("DIRECT TEST - fnPtr: %X, aev: %X\n", fnPtr, uintptr(unsafe.Pointer(aev)))
		hr := callSetVolumeScalar(
			fnPtr,
			uintptr(unsafe.Pointer(aev)),
			targetScalar,
			0,
		)
		if hr != 0 {
			t.Errorf("Failed to set scalar to %v: HRESULT %X", targetScalar, hr)
		} else {
			var actualScalar float32
			if err := aev.GetMasterVolumeLevelScalar(&actualScalar); err != nil {
				t.Errorf("Failed to get scalar after setting %v: %v", targetScalar, err)
			} else {
				fmt.Printf("DIRECT LOOP TEST - Set Level: %v -> Target Scalar: %v, Actual Scalar: %v, Calculated Level: %v\n",
					lvl, targetScalar, actualScalar, int(math.Round(float64(actualScalar)*100)))
			}
		}

		aev.Release()
		mmd.Release()
		mmde.Release()
		ole.CoUninitialize()
		runtime.UnlockOSThread()
	}
}

func TestSetVolumeWindowsService(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	bus := eventbus.New()

	svc := New(log, bus)

	levels := []int{0, 20, 50, 75, 100}
	for _, lvl := range levels {
		st, err := svc.SetVolume(lvl)
		if err != nil {
			t.Fatalf("SetVolume(%d) failed: %v", lvl, err)
		}
		if st.Level != lvl {
			t.Errorf("SetVolume(%d) expected return level %d, got %d", lvl, lvl, st.Level)
		}

		current, err := svc.GetVolume()
		if err != nil {
			t.Fatalf("GetVolume() failed: %v", err)
		}
		if current.Level != lvl {
			t.Errorf("After SetVolume(%d), GetVolume() returned level %d, expected %d", lvl, current.Level, lvl)
		}
	}
}
