//go:build !darwin && !windows

package main

func platformKeyCodeForBase(base rune) (uint16, bool) {
	_ = base
	return 0, false
}

func platformName() string {
	return "unsupported platform"
}

func platformHotkeyHelp() string {
	return "global hotkeys and key playback are implemented for macOS and Windows"
}
