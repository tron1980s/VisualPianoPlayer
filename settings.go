package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const settingsFileName = "settings.json"

type UserSettings struct {
	EnableSlotHotkeys  bool      `json:"enable_slot_hotkeys"`
	ConsumeSlotHotkeys bool      `json:"consume_slot_hotkeys"`
	SlotPaths          [9]string `json:"slot_paths"`
}

func DefaultUserSettings() UserSettings {
	return UserSettings{
		EnableSlotHotkeys:  true,
		ConsumeSlotHotkeys: true,
	}
}

func LoadUserSettings() UserSettings {
	settings := DefaultUserSettings()
	data, err := os.ReadFile(settingsFileName)
	if err != nil {
		return settings
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return DefaultUserSettings()
	}
	return settings
}

func UserSettingsFileExists() bool {
	_, err := os.Stat(settingsFileName)
	return err == nil
}

func SaveUserSettings(settings UserSettings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(settingsFileName, data, 0644)
}

func (settings UserSettings) HasSlotPaths() bool {
	for _, path := range settings.SlotPaths {
		if path != "" {
			return true
		}
	}
	return false
}

func FillEmptySlotsFromSongs(settings UserSettings, extraPaths ...string) UserSettings {
	choices := DiscoverMIDIFiles(extraPaths...)
	for index := range settings.SlotPaths {
		if index >= len(choices) {
			break
		}
		settings.SlotPaths[index] = storeSongPath(choices[index].Path)
	}
	return settings
}

func storeSongPath(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	wd, err := os.Getwd()
	if err != nil {
		return abs
	}
	rel, err := filepath.Rel(wd, abs)
	if err != nil || rel == "." || rel == "" || rel == ".." || startsWithParent(rel) {
		return abs
	}
	return rel
}

func resolveSongPath(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	wd, err := os.Getwd()
	if err != nil {
		return path
	}
	return filepath.Join(wd, path)
}

func startsWithParent(path string) bool {
	return len(path) >= 3 && path[:3] == ".."+string(filepath.Separator)
}
