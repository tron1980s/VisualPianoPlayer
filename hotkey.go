package main

import (
	"sync"
	"time"
)

type HotkeyConfig struct {
	KeyCode  int
	Consume  bool
	Debounce time.Duration
	OnPress  func()
}

type HotkeyBinding struct {
	KeyCode int
	Consume bool
	OnPress func()
}

var hotkeyState struct {
	mu        sync.Mutex
	bindings  map[int]HotkeyBinding
	debounce  time.Duration
	lastPress map[int]time.Time
	capture   func(int)
}

func ConfigureHotkey(cfg HotkeyConfig) {
	ConfigureHotkeyBindings([]HotkeyBinding{{
		KeyCode: cfg.KeyCode,
		Consume: cfg.Consume,
		OnPress: cfg.OnPress,
	}}, cfg.Debounce)
}

func ConfigureHotkeyBindings(bindings []HotkeyBinding, debounce time.Duration) {
	if debounce <= 0 {
		debounce = 250 * time.Millisecond
	}

	next := make(map[int]HotkeyBinding, len(bindings))
	for _, binding := range bindings {
		if binding.KeyCode <= 0 || binding.OnPress == nil {
			continue
		}
		next[binding.KeyCode] = binding
	}

	hotkeyState.mu.Lock()
	hotkeyState.bindings = next
	hotkeyState.debounce = debounce
	if hotkeyState.lastPress == nil {
		hotkeyState.lastPress = make(map[int]time.Time)
	}
	hotkeyState.mu.Unlock()
}

func CaptureNextPhysicalKey(callback func(int)) {
	hotkeyState.mu.Lock()
	hotkeyState.capture = callback
	hotkeyState.mu.Unlock()
}

func CancelKeyCapture() {
	hotkeyState.mu.Lock()
	hotkeyState.capture = nil
	hotkeyState.mu.Unlock()
}

func handleHotkeyFromPlatform(keyCode int, autorepeat bool) bool {
	if ignoreSyntheticKeyDown(keyCode) {
		return false
	}

	hotkeyState.mu.Lock()
	if hotkeyState.capture != nil && !autorepeat {
		callback := hotkeyState.capture
		hotkeyState.capture = nil
		hotkeyState.mu.Unlock()
		callback(keyCode)
		return true
	}

	binding, ok := hotkeyState.bindings[keyCode]
	if !ok {
		hotkeyState.mu.Unlock()
		return false
	}

	consume := binding.Consume
	if autorepeat {
		hotkeyState.mu.Unlock()
		return consume
	}

	now := time.Now()
	if last := hotkeyState.lastPress[keyCode]; !last.IsZero() && now.Sub(last) < hotkeyState.debounce {
		hotkeyState.mu.Unlock()
		return consume
	}
	hotkeyState.lastPress[keyCode] = now
	callback := binding.OnPress
	hotkeyState.mu.Unlock()

	callback()
	return consume
}
