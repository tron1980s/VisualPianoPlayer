//go:build windows

package main

import (
	"os/exec"
	"strings"
)

func OpenNativeMIDIFile() (string, bool, error) {
	script := `
Add-Type -AssemblyName System.Windows.Forms
$dialog = New-Object System.Windows.Forms.OpenFileDialog
$dialog.Title = "Choose a MIDI file"
$dialog.Filter = "MIDI files (*.mid;*.midi)|*.mid;*.midi|All files (*.*)|*.*"
if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
	[Console]::WriteLine($dialog.FileName)
}
`

	cmd := exec.Command("powershell.exe", "-NoProfile", "-STA", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		return "", false, err
	}

	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", false, nil
	}
	return path, true, nil
}
