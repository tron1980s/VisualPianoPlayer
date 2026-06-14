//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func OpenNativeMIDIFile() (string, bool, error) {
	script := `set pickedFile to choose file with prompt "Choose a MIDI file" of type {"mid", "midi", "public.midi-audio"}
POSIX path of pickedFile`
	output, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if strings.Contains(strings.ToLower(message), "user canceled") || strings.Contains(strings.ToLower(message), "cancelled") {
			return "", false, nil
		}
		if message == "" {
			message = err.Error()
		}
		return "", false, fmt.Errorf("macOS file picker failed: %s", message)
	}
	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", false, nil
	}
	return path, true, nil
}
