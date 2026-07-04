//go:build windows

package media

import (
	"encoding/json"
	"os/exec"
	"syscall"
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	procKeybdEvent = user32.NewProc("keybd_event")
)

const (
	VK_MEDIA_NEXT_TRACK = 0xB5
	VK_MEDIA_PREV_TRACK = 0xB6
	VK_MEDIA_STOP       = 0xB2
	VK_MEDIA_PLAY_PAUSE = 0xB3
	KEYEVENTF_KEYUP     = 0x0002
)

const getMediaScript = `Add-Type -AssemblyName System.Runtime.WindowsRuntime
$asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object { 
    $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -like 'IAsyncOperation*' 
})[0]

function Await($WinRtTask, $ResultType) {
    $asTask = $asTaskGeneric.MakeGenericMethod($ResultType)
    $netTask = $asTask.Invoke($null, @($WinRtTask))
    $netTask.Wait(-1) | Out-Null
    return $netTask.Result
}

try {
    $sessionManagerType = [Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager, Windows.Media.Control, ContentType=WindowsRuntime]
    $managerTask = [Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager]::RequestAsync()
    $sessionManager = Await $managerTask $sessionManagerType

    $session = $sessionManager.GetCurrentSession()
    if ($session) {
        $propsTask = $session.TryGetMediaPropertiesAsync()
        $propsType = [Windows.Media.Control.GlobalSystemMediaTransportControlsSessionMediaProperties, Windows.Media.Control, ContentType=WindowsRuntime]
        $props = Await $propsTask $propsType

        $playbackInfo = $session.GetPlaybackInfo()
        $status = $playbackInfo.PlaybackStatus.ToString()

        $result = @{
            title      = $props.Title
            artist     = $props.Artist
            albumTitle = $props.AlbumTitle
            status     = $status
        }
        $result | ConvertTo-Json -Compress
    } else {
        "{}"
    }
} catch {
    "{}"
}`

func (s *Service) GetMediaState() (MediaState, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", getMediaScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		s.log.Error("failed to run powershell get media script", "err", err)
		return MediaState{}, err
	}

	var state MediaState
	if err := json.Unmarshal(out, &state); err != nil {
		s.log.Debug("empty or invalid media state json from powershell", "output", string(out))
		return MediaState{}, nil
	}

	return state, nil
}

func pressKey(vk byte) {
	procKeybdEvent.Call(uintptr(vk), 0, 0, 0)
	procKeybdEvent.Call(uintptr(vk), 0, KEYEVENTF_KEYUP, 0)
}

func (s *Service) PlayPause() error {
	s.log.Info("media action: play_pause")
	pressKey(VK_MEDIA_PLAY_PAUSE)
	return nil
}

func (s *Service) Next() error {
	s.log.Info("media action: next")
	pressKey(VK_MEDIA_NEXT_TRACK)
	return nil
}

func (s *Service) Previous() error {
	s.log.Info("media action: previous")
	pressKey(VK_MEDIA_PREV_TRACK)
	return nil
}

func (s *Service) StopMedia() error {
	s.log.Info("media action: stop")
	pressKey(VK_MEDIA_STOP)
	return nil
}

