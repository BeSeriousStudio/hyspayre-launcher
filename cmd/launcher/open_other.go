//go:build !windows

package main

import (
	"os/exec"
)

func openAppWindow(url, dir string) error { return openExternal(url) }
func openExternal(url string) error       { return exec.Command("xdg-open", url).Start() }
