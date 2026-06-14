//go:build !darwin && !windows

package main

import "fmt"

func OpenNativeMIDIFile() (string, bool, error) {
	return "", false, fmt.Errorf("native MIDI picker is implemented for macOS and Windows only")
}
