//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func openAppWindow(url, appDir string) error {
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "Application", "msedge.exe"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			cmd := exec.Command(p, "--app="+url, "--new-window", "--no-first-run", "--user-data-dir="+filepath.Join(appDir, "WindowProfile"))
			if err := cmd.Start(); err == nil {
				cmd.Process.Release()
				return nil
			}
		}
	}
	// Fallback to the system default browser if Edge was uninstalled.
	return openExternal(url)
}
func openExternal(url string) error {
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Browser konnte nicht geöffnet werden: %w", err)
	}
	return cmd.Process.Release()
}
