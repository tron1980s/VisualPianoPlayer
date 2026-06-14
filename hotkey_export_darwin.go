//go:build darwin

package main

/*
#include <stdint.h>
*/
import "C"

//export goHandleKeyDownBridge
func goHandleKeyDownBridge(keyCode C.longlong, flags C.ulonglong, autorepeat C.int) C.int {
	_ = flags
	if handleHotkeyFromPlatform(int(keyCode), autorepeat != 0) {
		return 1
	}
	return 0
}
