package main

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type SongChoice struct {
	Label string
	Path  string
}

func DiscoverMIDIFiles(extraPaths ...string) []SongChoice {
	paths := discoverMIDIPaths(extraPaths...)
	baseCounts := make(map[string]int, len(paths))
	for _, path := range paths {
		baseCounts[strings.ToLower(filepath.Base(path))]++
	}

	choices := make([]SongChoice, 0, len(paths))
	for _, path := range paths {
		label := filepath.Base(path)
		if baseCounts[strings.ToLower(label)] > 1 {
			if rel, err := filepath.Rel(".", path); err == nil {
				label = rel
			}
		}
		choices = append(choices, SongChoice{
			Label: label,
			Path:  path,
		})
	}

	sort.SliceStable(choices, func(i, j int) bool {
		return strings.ToLower(choices[i].Label) < strings.ToLower(choices[j].Label)
	})
	return choices
}

func discoverMIDIPaths(extraPaths ...string) []string {
	seen := make(map[string]bool)
	var paths []string

	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" || !IsMIDIPath(path) {
			return
		}
		resolved := resolveSongPath(path)
		info, err := os.Stat(resolved)
		if err != nil || info.IsDir() {
			return
		}
		abs, err := filepath.Abs(resolved)
		if err != nil {
			abs = resolved
		}
		key := abs
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if seen[key] {
			return
		}
		seen[key] = true
		paths = append(paths, abs)
	}

	for _, path := range extraPaths {
		add(path)
	}

	for _, pattern := range []string{"*.mid", "*.midi", filepath.Join(importedMIDIDir, "*.mid"), filepath.Join(importedMIDIDir, "*.midi")} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		sort.Strings(matches)
		for _, match := range matches {
			add(match)
		}
	}

	return paths
}
