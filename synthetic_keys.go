package main

import (
	"sync"
	"time"
)

const syntheticKeyTTL = 1500 * time.Millisecond

var syntheticKeys struct {
	mu      sync.Mutex
	pending map[int][]time.Time
}

func markSyntheticKeyDown(keyCode uint16) {
	now := time.Now()
	expiresAt := now.Add(syntheticKeyTTL)
	key := int(keyCode)

	syntheticKeys.mu.Lock()
	defer syntheticKeys.mu.Unlock()

	if syntheticKeys.pending == nil {
		syntheticKeys.pending = make(map[int][]time.Time)
	}
	cleanupSyntheticKeyQueueLocked(key, now)
	queue := syntheticKeys.pending[key]
	if len(queue) > 256 {
		queue = queue[len(queue)-256:]
	}
	syntheticKeys.pending[key] = append(queue, expiresAt)
}

func ignoreSyntheticKeyDown(keyCode int) bool {
	now := time.Now()

	syntheticKeys.mu.Lock()
	defer syntheticKeys.mu.Unlock()

	if syntheticKeys.pending == nil {
		return false
	}
	cleanupSyntheticKeyQueueLocked(keyCode, now)
	queue := syntheticKeys.pending[keyCode]
	if len(queue) == 0 {
		return false
	}
	if len(queue) == 1 {
		delete(syntheticKeys.pending, keyCode)
		return true
	}
	syntheticKeys.pending[keyCode] = queue[1:]
	return true
}

func cleanupSyntheticKeyQueueLocked(keyCode int, now time.Time) {
	queue := syntheticKeys.pending[keyCode]
	start := 0
	for start < len(queue) && now.After(queue[start]) {
		start++
	}
	if start == len(queue) {
		delete(syntheticKeys.pending, keyCode)
		return
	}
	if start > 0 {
		syntheticKeys.pending[keyCode] = append(queue[:0], queue[start:]...)
	}
}
