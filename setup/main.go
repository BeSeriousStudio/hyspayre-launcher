//go:build windows

// Hyspayre Launcher Setup 0.2.5
// Per-user Windows installer; no administrator rights and no security bypass.
package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

//go:embed payload.zip welcome-banner.png hyspayre.ico
var bundled embed.FS

const (
	appName     = "Hyspayre Launcher"
	appVersion  = "0.2.5"
	regPath     = `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\HyspayreLauncher`
	mbOK        = 0x00000000
	mbYesNo     = 0x00000004
	mbIconInfo  = 0x00000040
	mbIconWarn  = 0x00000030
	mbIconError = 0x00000010
	idYes       = 6
)

var msgProc = syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")

func dialog(title, body string, flags uintptr) int {
	a, _ := syscall.UTF16PtrFromString(title)
	b, _ := syscall.UTF16PtrFromString(body)
	r, _, _ := msgProc.Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(a)), flags)
	return int(r)
}
func ask(body string) bool {
	return dialog("Hyspayre Launcher Setup", body, mbYesNo|mbIconInfo) == idYes
}
func alert(body string) { dialog("Hyspayre Launcher Setup", body, mbOK|mbIconInfo) }
func fail(err error)    { dialog("Hyspayre Launcher – Fehler", err.Error(), mbOK|mbIconError) }

func localPath() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return "", errors.New("LOCALAPPDATA fehlt. Bitte über ein normales Windows-Benutzerkonto anmelden.")
	}
	return filepath.Join(base, "Programs", "Hyspayre Launcher"), nil
}
func isRegularInDir(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && !filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
func command(program string, args ...string) error {
	_, err := commandOutput(program, args...)
	return err
}
func commandOutput(program string, args ...string) ([]byte, error) {
	c := exec.Command(program, args...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := c.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s: %w (%s)", filepath.Base(program), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}
func fallbackWelcome(dir string) (bool, bool) {
	if !ask("Hyspayre Launcher " + appVersion + " für diesen Windows-Benutzer installieren?\n\nInstallationsort:\n" + dir + "\n\nKeine Administratorrechte erforderlich.") {
		return false, false
	}
	desktop := ask("Soll eine Verknüpfung auf dem Desktop erstellt werden?\n\n(Der Startmenü-Eintrag wird immer angelegt.)")
	return true, desktop
}
func welcomeDialog(dir string) (bool, bool, error) {
	banner, err := bundled.ReadFile("welcome-banner.png")
	if err != nil {
		return false, false, err
	}
	tempDir, err := os.MkdirTemp("", "hyspayre-setup-ui-")
	if err != nil {
		return false, false, err
	}
	defer os.RemoveAll(tempDir)

	bannerPath := filepath.Join(tempDir, "welcome-banner.png")
	iconPath := filepath.Join(tempDir, "hyspayre.ico")
	if err := os.WriteFile(bannerPath, banner, 0600); err != nil {
		return false, false, err
	}

	icon, err := bundled.ReadFile("hyspayre.ico")
	if err != nil {
		return false, false, err
	}
	if err := os.WriteFile(iconPath, icon, 0600); err != nil {
		return false, false, err
	}
	script := filepath.Join(tempDir, "welcome.ps1")
	content := `param([string]$Banner,[string]$Icon,[string]$InstallDir)
$ErrorActionPreference='Stop'
Add-Type -AssemblyName PresentationFramework,PresentationCore
[xml]$xaml=@"
<Window xmlns="http://schemas.microsoft.com/winfx/2006/xaml/presentation"
        xmlns:x="http://schemas.microsoft.com/winfx/2006/xaml"
        Title="Hyspayre Launcher Setup" Width="940" Height="795"
        WindowStartupLocation="CenterScreen" ResizeMode="NoResize" Background="#080F1B">
  <Border Margin="16" CornerRadius="14" BorderBrush="#37536E" BorderThickness="1" Background="#0C1424">
    <Grid Margin="12">
      <Grid.RowDefinitions>
        <RowDefinition Height="Auto"/>
        <RowDefinition Height="*"/>
      </Grid.RowDefinitions>
      <Image x:Name="BrandImage" Grid.Row="0" Height="510" Stretch="Uniform" />
      <Grid Grid.Row="1" Margin="20,8,20,10">
        <Grid.RowDefinitions>
          <RowDefinition Height="Auto"/>
          <RowDefinition Height="Auto"/>
          <RowDefinition Height="*"/>
        </Grid.RowDefinitions>
        <TextBlock Grid.Row="0" Text="Hyspayre Launcher 0.2.4" Foreground="#C7D9EA" FontSize="16" FontWeight="SemiBold" Margin="0,0,0,8" />
        <TextBlock Grid.Row="1" Text="Deine Spielebibliothek, dein Account und deine Freunde – alles an einem Ort." Foreground="#9FB4CB" FontSize="14" Margin="0,0,0,14" />
        <Grid Grid.Row="2">
          <Grid.ColumnDefinitions>
            <ColumnDefinition Width="*"/>
            <ColumnDefinition Width="Auto"/>
          </Grid.ColumnDefinitions>
          <StackPanel Grid.Column="0" VerticalAlignment="Center">
            <TextBlock Text="Installationsort" Foreground="#80C9FF" FontSize="12" FontWeight="SemiBold" />
            <TextBox x:Name="InstallPath" IsReadOnly="True" Width="440" Height="32" Margin="0,6,0,8" Padding="8,5" HorizontalAlignment="Left" Background="#101F36" Foreground="#E6ECF5" BorderBrush="#41617E" />
            <CheckBox x:Name="DesktopCheck" Content="Desktop-Verknüpfung erstellen" IsChecked="True" Foreground="#D8E4F0" />
          </StackPanel>
          <StackPanel Grid.Column="1" Orientation="Horizontal" VerticalAlignment="Bottom" Margin="12,0,0,0">
            <Button x:Name="CancelButton" Content="Abbrechen" Width="112" Height="40" Margin="0,0,10,0" Background="#192943" Foreground="#D7E5F5" BorderBrush="#3B5879" />
            <Button x:Name="InstallButton" Content="Installieren" Width="140" Height="40" Background="#80C9FF" Foreground="#07101E" FontWeight="Bold" BorderBrush="#80C9FF" />
          </StackPanel>
        </Grid>
      </Grid>
    </Grid>
  </Border>
</Window>
"@
$reader=New-Object System.Xml.XmlNodeReader $xaml
$window=[Windows.Markup.XamlReader]::Load($reader)
$window.Icon=[System.Windows.Media.Imaging.BitmapFrame]::Create([uri](Resolve-Path -LiteralPath $Icon).Path)
$window.FindName('BrandImage').Source=[System.Windows.Media.Imaging.BitmapImage]::new([uri](Resolve-Path -LiteralPath $Banner).Path)
$installPath=$window.FindName('InstallPath')
$desktopCheck=$window.FindName('DesktopCheck')
$installButton=$window.FindName('InstallButton')
$cancelButton=$window.FindName('CancelButton')
$installPath.Text=$InstallDir
$script:Result='CANCEL|0'
$installButton.Add_Click({ $script:Result='INSTALL|' + ([int][bool]$desktopCheck.IsChecked); $window.Close() })
$cancelButton.Add_Click({ $window.Close() })
$null=$window.ShowDialog()
Write-Output $script:Result
`
	if err := os.WriteFile(script, []byte(content), 0600); err != nil {
		return false, false, err
	}
	out, err := commandOutput("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-STA", "-File", script, bannerPath, iconPath, dir)
	if err != nil {
		return false, false, err
	}
	result := strings.TrimSpace(string(out))
	if strings.HasPrefix(result, "INSTALL|") {
		return true, strings.HasSuffix(result, "1"), nil
	}
	return false, false, nil
}
func psLiteral(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func createShortcuts(exe, ico string, desktop bool) error {
	script := `$ErrorActionPreference='Stop';` +
		`$shell=New-Object -ComObject WScript.Shell;` +
		`$target=` + psLiteral(exe) + `;` +
		`$icon=` + psLiteral(ico) + `;` +
		`$start=[System.IO.Path]::Combine([Environment]::GetFolderPath('Programs'),'Hyspayre Launcher.lnk');` +
		`$link=$shell.CreateShortcut($start);$link.TargetPath=$target;$link.WorkingDirectory=[System.IO.Path]::GetDirectoryName($target);$link.IconLocation=$icon;$link.Description='Hyspayre Spielebibliothek, Updates und Community';$link.Save();`
	if desktop {
		script += `$d=[System.IO.Path]::Combine([Environment]::GetFolderPath('Desktop'),'Hyspayre Launcher.lnk');` +
			`$lnk=$shell.CreateShortcut($d);$lnk.TargetPath=$target;$lnk.WorkingDirectory=[System.IO.Path]::GetDirectoryName($target);$lnk.IconLocation=$icon;$lnk.Description='Hyspayre Spielebibliothek, Updates und Community';$lnk.Save();`
	}
	return command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
}
func removeShortcuts() error {
	script := `$a=[System.IO.Path]::Combine([Environment]::GetFolderPath('Programs'),'Hyspayre Launcher.lnk');` +
		`$b=[System.IO.Path]::Combine([Environment]::GetFolderPath('Desktop'),'Hyspayre Launcher.lnk');` +
		`Remove-Item -LiteralPath $a -ErrorAction SilentlyContinue;Remove-Item -LiteralPath $b -ErrorAction SilentlyContinue;`
	return command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
}
func addRegistry(location, uninstallPath, appPath string) error {
	type val struct{ name, value, kind string }
	vals := []val{
		{"DisplayName", appName, "REG_SZ"},
		{"DisplayVersion", appVersion, "REG_SZ"},
		{"Publisher", "Hyspayre", "REG_SZ"},
		{"InstallLocation", location, "REG_SZ"},
		{"DisplayIcon", filepath.Join(location, "hyspayre.ico") + ",0", "REG_SZ"},
		{"UninstallString", `"` + uninstallPath + `" --uninstall`, "REG_SZ"},
		{"NoModify", "1", "REG_DWORD"},
		{"NoRepair", "1", "REG_DWORD"},
	}
	for _, v := range vals {
		if err := command("reg.exe", "add", regPath, "/v", v.name, "/t", v.kind, "/d", v.value, "/f"); err != nil {
			return err
		}
	}
	return nil
}
func removeRegistry() error { return command("reg.exe", "delete", regPath, "/f") }

func files() (map[string][]byte, error) {
	raw, err := bundled.ReadFile("payload.zip")
	if err != nil {
		return nil, err
	}
	z, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, err
	}
	allowed := map[string]int64{"Hyspayre-Launcher.exe": 15 << 20, "hyspayre.ico": 1 << 20}
	out := map[string][]byte{}
	for _, f := range z.File {
		lim, okay := allowed[f.Name]
		if !okay || f.FileInfo().IsDir() || f.UncompressedSize64 > uint64(lim) {
			return nil, fmt.Errorf("Unzulässiger Dateiinhalt: %s", f.Name)
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(r, lim+1))
		r.Close()
		if err != nil || int64(len(b)) > lim {
			return nil, errors.New("Installationspaket ist defekt")
		}
		out[f.Name] = b
	}
	for name := range allowed {
		if len(out[name]) == 0 {
			return nil, fmt.Errorf("Datei fehlt im Setup: %s", name)
		}
	}
	check := sha256.Sum256(out["Hyspayre-Launcher.exe"])
	if hex.EncodeToString(check[:]) != expectedLauncherSHA256 {
		return nil, errors.New("Die Launcher-Prüfsumme stimmt nicht: Installation abgebrochen")
	}
	return out, nil
}
func atomicCopy(path string, data []byte) error {
	folder := filepath.Dir(path)
	if err := os.MkdirAll(folder, 0700); err != nil {
		return err
	}
	newFile := path + ".new"
	if err := os.WriteFile(newFile, data, 0600); err != nil {
		return err
	}
	oldFile := path + ".old"
	os.Remove(oldFile)
	previous := false
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, oldFile); err != nil {
			os.Remove(newFile)
			return fmt.Errorf("Bitte Hyspayre Launcher schließen und Setup erneut starten: %w", err)
		}
		previous = true
	}
	if err := os.Rename(newFile, path); err != nil {
		if previous {
			_ = os.Rename(oldFile, path)
		}
		os.Remove(newFile)
		return err
	}
	if previous {
		os.Remove(oldFile)
	}
	return nil
}
func install() {
	dir, err := localPath()
	if err != nil {
		fail(err)
		return
	}
	proceed, desktop, err := welcomeDialog(dir)
	if err != nil {
		proceed, desktop = fallbackWelcome(dir)
	}
	if !proceed {
		return
	}
	items, err := files()
	if err != nil {
		fail(err)
		return
	}
	for _, name := range []string{"Hyspayre-Launcher.exe", "hyspayre.ico"} {
		if err = atomicCopy(filepath.Join(dir, name), items[name]); err != nil {
			fail(err)
			return
		}
	}
	me, err := os.Executable()
	if err != nil {
		fail(err)
		return
	}
	self, err := os.ReadFile(me)
	if err != nil {
		fail(err)
		return
	}
	uninstall := filepath.Join(dir, "Uninstall-Hyspayre.exe")
	if err = atomicCopy(uninstall, self); err != nil {
		fail(err)
		return
	}
	exe := filepath.Join(dir, "Hyspayre-Launcher.exe")
	regErr := addRegistry(dir, uninstall, exe)
	shortErr := createShortcuts(exe, filepath.Join(dir, "hyspayre.ico"), desktop)
	if regErr != nil || shortErr != nil {
		alert("Der Launcher wurde installiert, aber Windows konnte nicht alle Verknüpfungen oder Programmeinträge anlegen.\n\nDu findest ihn unter:\n" + exe + "\n\nDetails:\n" + fmt.Sprint(regErr) + "\n" + fmt.Sprint(shortErr))
	}
	if ask("Hyspayre Launcher wurde installiert!\n\nInstallationsort: " + dir + "\n\nLauncher jetzt starten?") {
		c := exec.Command(exe)
		if err := c.Start(); err == nil {
			c.Process.Release()
		} else {
			fail(err)
		}
	}
}
func randomName() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return strconv.Itoa(os.Getpid())
	}
	return hex.EncodeToString(b)
}
func stageUninstall() {
	me, err := os.Executable()
	if err != nil {
		fail(err)
		return
	}
	content, err := os.ReadFile(me)
	if err != nil {
		fail(err)
		return
	}
	folder := filepath.Join(os.TempDir(), "HyspayreUninstall-"+randomName())
	if err = os.MkdirAll(folder, 0700); err != nil {
		fail(err)
		return
	}
	temp := filepath.Join(folder, "Hyspayre-Uninstall.exe")
	if err = os.WriteFile(temp, content, 0600); err != nil {
		fail(err)
		return
	}
	c := exec.Command(temp, "--perform-uninstall")
	if err = c.Start(); err != nil {
		fail(err)
		return
	}
	c.Process.Release()
}
func uninstall() {
	dir, err := localPath()
	if err != nil {
		fail(err)
		return
	}
	if !ask("Hyspayre Launcher von diesem PC deinstallieren?\n\nDie separat installierten Spiele und deine Einstellungen unter LOCALAPPDATA\\HyspayreLauncher bleiben aus Sicherheitsgründen erhalten.") {
		return
	}
	for _, name := range []string{"Hyspayre-Launcher.exe", "hyspayre.ico", "Uninstall-Hyspayre.exe"} {
		path := filepath.Join(dir, name)
		if !isRegularInDir(dir, path) {
			fail(errors.New("Ungültiger Installationspfad"))
			return
		}
		if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
			fail(fmt.Errorf("Bitte den laufenden Launcher zuerst beenden: %w", err))
			return
		}
	}
	_ = os.Remove(dir)
	_ = removeShortcuts()
	_ = removeRegistry()
	alert("Hyspayre Launcher wurde deinstalliert.\n\nSpiele und Profildaten sind erhalten geblieben.\nDu kannst sie später manuell in LOCALAPPDATA\\HyspayreLauncher entfernen.")
}
func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--uninstall":
			stageUninstall()
			return
		case "--perform-uninstall":
			uninstall()
			return
		}
	}
	install()
}
