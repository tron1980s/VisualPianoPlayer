//go:build windows

package main

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

const (
	inputKeyboard = 1

	keyeventfKeyup = 0x0002

	whKeyboardLL = 13
	wmKeydown    = 0x0100
	wmSyskeydown = 0x0104

	llkhfInjected = 0x00000010

	vkLShift = 0xA0
	vkOEM1   = 0xBA
)

var (
	user32                    = syscall.NewLazyDLL("user32.dll")
	procCallNextHookEx        = user32.NewProc("CallNextHookEx")
	procDispatchMessage       = user32.NewProc("DispatchMessageW")
	procGetMessage            = user32.NewProc("GetMessageW")
	procPeekMessage           = user32.NewProc("PeekMessageW")
	procSendInput             = user32.NewProc("SendInput")
	procSetWindowsHookEx      = user32.NewProc("SetWindowsHookExW")
	procTranslateMessage      = user32.NewProc("TranslateMessage")
	procUnhookWindowsHookEx   = user32.NewProc("UnhookWindowsHookEx")
	windowsKeyboardHookProc   = syscall.NewCallback(keyboardHookProc)
	windowsKeyboardHookHandle uintptr
)

type keyboardInput struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type input struct {
	Type uint32
	_    uint32
	Ki   keyboardInput
	_    uint64
}

type kbdLLHookStruct struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type windowsPoint struct {
	X int32
	Y int32
}

type windowsMSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      windowsPoint
}

func RunGlobalHotkeyListener() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var msg windowsMSG
	procPeekMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 0)

	hook, _, err := procSetWindowsHookEx.Call(
		uintptr(whKeyboardLL),
		windowsKeyboardHookProc,
		0,
		0,
	)
	if hook == 0 {
		return fmt.Errorf("could not install Windows keyboard hook: %w", err)
	}
	windowsKeyboardHookHandle = hook
	defer func() {
		procUnhookWindowsHookEx.Call(hook)
		windowsKeyboardHookHandle = 0
	}()

	for {
		ret, _, err := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		switch int32(ret) {
		case -1:
			return fmt.Errorf("Windows hotkey message loop failed: %w", err)
		case 0:
			return nil
		default:
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
}

func keyboardHookProc(nCode uintptr, wParam uintptr, lParam uintptr) uintptr {
	code := int32(nCode)
	if code >= 0 && (wParam == wmKeydown || wParam == wmSyskeydown) {
		info := (*kbdLLHookStruct)(unsafe.Pointer(lParam))
		if info.Flags&llkhfInjected == 0 && handleHotkeyFromPlatform(int(info.VkCode), false) {
			return 1
		}
	}

	ret, _, _ := procCallNextHookEx.Call(windowsKeyboardHookHandle, nCode, wParam, lParam)
	return ret
}

func postPlatformKey(keyCode uint16, down bool, shift bool) {
	_ = shift

	flags := uint32(0)
	if !down {
		flags |= keyeventfKeyup
	}

	event := input{
		Type: inputKeyboard,
		Ki: keyboardInput{
			WVk:     keyCode,
			DwFlags: flags,
		},
	}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&event)), unsafe.Sizeof(event))
}

func platformLeftShiftKeyCode() uint16 {
	return vkLShift
}

func defaultHotkeyKeyCode() int {
	return vkOEM1
}
