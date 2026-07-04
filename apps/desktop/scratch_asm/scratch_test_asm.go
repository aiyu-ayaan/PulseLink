package main

import (
	"fmt"
	"math"
	"runtime"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

// Declare the assembly function
func callSetVolumeScalar(fn uintptr, aev uintptr, level float32, eventContextGUID uintptr) uintptr

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		fmt.Printf("CoInitializeEx failed: %v\n", err)
		return
	}
	defer ole.CoUninitialize()

	var mmde *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
		fmt.Printf("CoCreateInstance failed: %v\n", err)
		return
	}
	defer mmde.Release()

	var mmd *wca.IMMDevice
	if err := mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmd); err != nil {
		fmt.Printf("GetDefaultAudioEndpoint failed: %v\n", err)
		return
	}
	defer mmd.Release()

	var aev *wca.IAudioEndpointVolume
	if err := mmd.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &aev); err != nil {
		fmt.Printf("Activate failed: %v\n", err)
		return
	}
	defer aev.Release()

	// Set to multiple levels
	levels := []int{0, 20, 50, 75, 100}
	for _, lvl := range levels {
		targetScalar := float32(lvl) / 100
		
		// Call using our assembly function
		hr := callSetVolumeScalar(
			aev.VTable().SetMasterVolumeLevelScalar,
			uintptr(unsafe.Pointer(aev)),
			targetScalar,
			0,
		)
		if hr != 0 {
			fmt.Printf("Failed to set scalar to %v: HRESULT %X\n", targetScalar, hr)
			continue
		}

		var actualScalar float32
		if err := aev.GetMasterVolumeLevelScalar(&actualScalar); err != nil {
			fmt.Printf("Failed to get scalar after setting %v: %v\n", targetScalar, err)
			continue
		}
		fmt.Printf("Set Level: %v -> Target Scalar: %v, Actual Scalar: %v, Calculated Level: %v\n",
			lvl, targetScalar, actualScalar, int(math.Round(float64(actualScalar)*100)))
	}
}
