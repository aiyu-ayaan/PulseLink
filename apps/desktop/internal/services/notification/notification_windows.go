//go:build windows

package notification

import (
	"fmt"
	"os/exec"
	"strings"
)

func (s *Service) ShowToast(title, message string) error {
	s.log.Info("notification action: toast", "title", title, "message", message)
	
	escapedTitle := strings.ReplaceAll(title, "'", "''")
	escapedMsg := strings.ReplaceAll(message, "'", "''")
	
	psCmd := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$toastXml = [xml]$template.GetXml()
$toastXml.GetElementsByTagName('text')[0].AppendChild($toastXml.CreateTextNode('%s')) | Out-Null
$toastXml.GetElementsByTagName('text')[1].AppendChild($toastXml.CreateTextNode('%s')) | Out-Null
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($toastXml.OuterXml)
$toast = New-Object Windows.UI.Notifications.ToastNotification $xml
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('PulseLink').Show($toast)
`, escapedTitle, escapedMsg)

	return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Run()
}
