//go:build !darwin && !windows

package main

import "fmt"

func RunGlobalHotkeyListener() error {
	return fmt.Errorf("global hotkey and key playback are currently implemented for macOS and Windows only")
}

func postPlatformKey(keyCode uint16, down bool, shift bool) {
	_, _, _ = keyCode, down, shift
}

func platformLeftShiftKeyCode() uint16 {
	return 0
}

func defaultHotkeyKeyCode() int {
	return 0
}
