//go:build darwin

package main

/*
#cgo darwin LDFLAGS: -framework ApplicationServices -framework CoreFoundation
#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>

extern int goHandleKeyDownBridge(long long keyCode, unsigned long long flags, int autorepeat);

static CFMachPortRef gEventTap = NULL;

static CGEventRef keyboardTapCallback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *refcon) {
	if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
		if (gEventTap != NULL) {
			CGEventTapEnable(gEventTap, true);
		}
		return event;
	}

	if (type != kCGEventKeyDown) {
		return event;
	}

	long long keyCode = (long long)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);
	int autorepeat = (int)CGEventGetIntegerValueField(event, kCGKeyboardEventAutorepeat);
	unsigned long long flags = (unsigned long long)CGEventGetFlags(event);
	int consume = goHandleKeyDownBridge(keyCode, flags, autorepeat);
	if (consume) {
		return NULL;
	}
	return event;
}

static int runKeyboardTap(void) {
	CGEventMask mask = CGEventMaskBit(kCGEventKeyDown);
	gEventTap = CGEventTapCreate(kCGSessionEventTap,
		kCGHeadInsertEventTap,
		kCGEventTapOptionDefault,
		mask,
		keyboardTapCallback,
		NULL);
	if (gEventTap == NULL) {
		return 1;
	}

	CFRunLoopSourceRef source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, gEventTap, 0);
	if (source == NULL) {
		CFRelease(gEventTap);
		gEventTap = NULL;
		return 2;
	}

	CFRunLoopAddSource(CFRunLoopGetCurrent(), source, kCFRunLoopCommonModes);
	CGEventTapEnable(gEventTap, true);
	CFRunLoopRun();

	CFRelease(source);
	CFRelease(gEventTap);
	gEventTap = NULL;
	return 0;
}

static void postKeyEvent(unsigned short keyCode, int down, int shift) {
	CGEventRef event = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)keyCode, down ? true : false);
	if (event == NULL) {
		return;
	}
	if (shift) {
		CGEventSetFlags(event, kCGEventFlagMaskShift);
	}
	CGEventPost(kCGHIDEventTap, event);
	CFRelease(event);
}
*/
import "C"

import "fmt"

func RunGlobalHotkeyListener() error {
	result := int(C.runKeyboardTap())
	if result == 0 {
		return nil
	}
	if result == 1 {
		return fmt.Errorf("could not install macOS keyboard event tap; grant Accessibility and Input Monitoring permission to this binary or your terminal, then run it again")
	}
	return fmt.Errorf("could not create macOS run loop source for keyboard event tap")
}

func postPlatformKey(keyCode uint16, down bool, shift bool) {
	C.postKeyEvent(C.ushort(keyCode), cBool(down), cBool(shift))
}

func platformLeftShiftKeyCode() uint16 {
	return 56
}

func defaultHotkeyKeyCode() int {
	return 41
}

func cBool(value bool) C.int {
	if value {
		return 1
	}
	return 0
}
