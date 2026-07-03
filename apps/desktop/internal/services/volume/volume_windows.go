//go:build windows

package volume

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

var (
	user32Windows  = syscall.NewLazyDLL("user32.dll")
	procKeybdEventWindows = user32Windows.NewProc("keybd_event")
)

const (
	VK_VOLUME_MUTE_WIN = 0xAD
	VK_VOLUME_DOWN_WIN = 0xAE
	VK_VOLUME_UP_WIN   = 0xAF
	KEYEVENTF_KEYUP_WIN = 0x0002
)

func pressKeyWin(vk byte) {
	procKeybdEventWindows.Call(uintptr(vk), 0, 0, 0)
	procKeybdEventWindows.Call(uintptr(vk), 0, KEYEVENTF_KEYUP_WIN, 0)
}

func (s *Service) VolumeUp() error {
	s.log.Info("volume action: up")
	pressKeyWin(VK_VOLUME_UP_WIN)
	return nil
}

func (s *Service) VolumeDown() error {
	s.log.Info("volume action: down")
	pressKeyWin(VK_VOLUME_DOWN_WIN)
	return nil
}

func (s *Service) VolumeMute() error {
	s.log.Info("volume action: mute")
	pressKeyWin(VK_VOLUME_MUTE_WIN)
	return nil
}

const psVolumeHelper = `
Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
[Guid("5CDF2C82-841E-4546-9722-0CF74078229A"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IAudioEndpointVolume {
    int RegisterControlChangeNotify(IntPtr pNotify);
    int UnregisterControlChangeNotify(IntPtr pNotify);
    int GetChannelCount(out uint pnChannelCount);
    int SetMasterVolumeLevel(float fLevelDB, ref Guid pguidEventContext);
    int SetMasterVolumeLevelScalar(float fLevel, ref Guid pguidEventContext);
    int GetMasterVolumeLevel(out float pfLevelDB);
    int GetMasterVolumeLevelScalar(out float pfLevel);
    int SetChannelVolumeLevel(uint nChannel, float fLevelDB, ref Guid pguidEventContext);
    int SetChannelVolumeLevelScalar(uint nChannel, float fLevel, ref Guid pguidEventContext);
    int GetChannelVolumeLevel(out float pfLevelDB);
    int GetChannelVolumeLevelScalar(uint nChannel, out float pfLevel);
    int SetMute([MarshalAs(UnmanagedType.Bool)] bool bMute, ref Guid pguidEventContext);
    int GetMute([MarshalAs(UnmanagedType.Bool)] out bool pbMute);
}
[Guid("D666063F-1587-4E43-81F1-B948E807363F"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IMMDevice {
    int Activate(ref Guid iid, int dwClsCtx, IntPtr pActivationParams, out object ppInterface);
}
[Guid("A95664D2-9614-4F35-A746-DE8DB63617E6"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IMMDeviceEnumerator {
    int EnumAudioEndpoints(int dataFlow, int dwStateMask, out object ppDevices);
    int GetDefaultAudioEndpoint(int dataFlow, int role, out IMMDevice ppDevice);
}
[ComImport, Guid("BCDE0395-E52F-467C-8E3D-C4579291692E")]
class MMDeviceEnumeratorCom { }
public class VolumeControl {
    public static float GetVolume() {
        var enumerator = (IMMDeviceEnumerator)new MMDeviceEnumeratorCom();
        IMMDevice device;
        enumerator.GetDefaultAudioEndpoint(0, 1, out device);
        object interfacePointer;
        Guid iid = new Guid("5CDF2C82-841E-4546-9722-0CF74078229A");
        device.Activate(ref iid, 23, IntPtr.Zero, out interfacePointer);
        var volume = (IAudioEndpointVolume)interfacePointer;
        float vol;
        volume.GetMasterVolumeLevelScalar(out vol);
        return vol * 100;
    }
    public static void SetVolume(float level) {
        var enumerator = (IMMDeviceEnumerator)new MMDeviceEnumeratorCom();
        IMMDevice device;
        enumerator.GetDefaultAudioEndpoint(0, 1, out device);
        object interfacePointer;
        Guid iid = new Guid("5CDF2C82-841E-4546-9722-0CF74078229A");
        device.Activate(ref iid, 23, IntPtr.Zero, out interfacePointer);
        var volume = (IAudioEndpointVolume)interfacePointer;
        Guid guid = Guid.Empty;
        volume.SetMasterVolumeLevelScalar(level / 100f, ref guid);
    }
    public static bool GetMute() {
        var enumerator = (IMMDeviceEnumerator)new MMDeviceEnumeratorCom();
        IMMDevice device;
        enumerator.GetDefaultAudioEndpoint(0, 1, out device);
        object interfacePointer;
        Guid iid = new Guid("5CDF2C82-841E-4546-9722-0CF74078229A");
        device.Activate(ref iid, 23, IntPtr.Zero, out interfacePointer);
        var volume = (IAudioEndpointVolume)interfacePointer;
        bool mute;
        volume.GetMute(out mute);
        return mute;
    }
}
"@
`

func (s *Service) GetVolume() (any, error) {
	cmd := psVolumeHelper + "\n[VolumeControl]::GetVolume(); [VolumeControl]::GetMute()"
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", cmd).Output()
	if err != nil {
		return nil, fmt.Errorf("powershell get volume: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\r\n")
	if len(lines) < 2 {
		lines = strings.Split(strings.TrimSpace(string(out)), "\n")
	}
	if len(lines) < 2 {
		return VolumeState{Level: 50, Muted: false}, nil
	}

	volVal, _ := strconv.ParseFloat(strings.TrimSpace(lines[0]), 32)
	muteVal := strings.ToLower(strings.TrimSpace(lines[1])) == "true"

	return VolumeState{
		Level: int(volVal),
		Muted: muteVal,
	}, nil
}

func (s *Service) SetVolume(level int) error {
	s.log.Info("volume action: set", "level", level)
	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}
	cmd := fmt.Sprintf("%s\n[VolumeControl]::SetVolume(%d)", psVolumeHelper, level)
	err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", cmd).Run()
	if err != nil {
		return fmt.Errorf("powershell set volume: %w", err)
	}
	return nil
}
