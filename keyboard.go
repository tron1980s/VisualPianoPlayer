package main

import "time"

const defaultShiftSettle = time.Millisecond

func pressStroke(stroke KeyStroke) {
	if stroke.Shift {
		markSyntheticKeyDown(platformLeftShiftKeyCode())
		postPlatformKey(platformLeftShiftKeyCode(), true, false)
		time.Sleep(defaultShiftSettle)
	}
	markSyntheticKeyDown(stroke.KeyCode)
	postPlatformKey(stroke.KeyCode, true, stroke.Shift)
	if stroke.Shift {
		time.Sleep(defaultShiftSettle)
		postPlatformKey(platformLeftShiftKeyCode(), false, false)
	}
}

func releaseStroke(stroke KeyStroke) {
	postPlatformKey(stroke.KeyCode, false, false)
}
