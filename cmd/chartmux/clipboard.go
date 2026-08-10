package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func copyPNG(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve PNG path: %w", err)
	}
	if _, err := os.Stat(absolute); err != nil {
		return fmt.Errorf("open PNG for clipboard: %w", err)
	}

	switch runtime.GOOS {
	case "darwin":
		script := "on run argv\nset the clipboard to (read (POSIX file (item 1 of argv)) as «class PNGf»)\nend run"
		if output, err := exec.Command("osascript", "-e", script, absolute).CombinedOutput(); err != nil {
			return fmt.Errorf("copy PNG with osascript: %s", commandError(err, output))
		}
		return nil
	case "linux":
		return copyPNGLinux(absolute)
	case "windows":
		script := "Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $image=[System.Drawing.Image]::FromFile($args[0]); [System.Windows.Forms.Clipboard]::SetImage($image); $image.Dispose()"
		if output, err := exec.Command("powershell", "-NoProfile", "-STA", "-Command", script, absolute).CombinedOutput(); err != nil {
			return fmt.Errorf("copy PNG with PowerShell: %s", commandError(err, output))
		}
		return nil
	default:
		return fmt.Errorf("clipboard image copy is not supported on %s", runtime.GOOS)
	}
}

func copyPNGLinux(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := exec.LookPath("wl-copy"); err == nil {
		command := exec.Command("wl-copy", "--type", "image/png")
		command.Stdin = file
		if output, err := command.CombinedOutput(); err != nil {
			return fmt.Errorf("copy PNG with wl-copy: %s", commandError(err, output))
		}
		return nil
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		if output, err := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-i", path).CombinedOutput(); err != nil {
			return fmt.Errorf("copy PNG with xclip: %s", commandError(err, output))
		}
		return nil
	}
	return fmt.Errorf("clipboard copy requires wl-copy or xclip on Linux")
}

func commandError(err error, output []byte) string {
	if len(output) == 0 {
		return err.Error()
	}
	return strings.TrimSpace(string(output))
}
