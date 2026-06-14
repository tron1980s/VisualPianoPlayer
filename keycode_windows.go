//go:build windows

package main

func platformKeyCodeForBase(base rune) (uint16, bool) {
	if base >= 'a' && base <= 'z' {
		return uint16(base - 'a' + 'A'), true
	}
	if base >= '0' && base <= '9' {
		return uint16(base), true
	}
	return 0, false
}

func platformName() string {
	return "Windows"
}

func platformHotkeyHelp() string {
	return "Windows virtual-key code; German ö is usually VK_OEM_1 / 0xBA / 186"
}
