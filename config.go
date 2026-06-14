package main

import "time"

type AppConfig struct {
	MIDIPath       string
	Mode           PlaybackMode
	Speed          float64
	Transpose      int
	DisableSustain bool
	AutoSustain    time.Duration
	LeadIn         time.Duration
	TapDuration    time.Duration
	InterKeyGap    time.Duration
	HotkeyCode     int
	ConsumeHotkey  bool
	KeyboardLayout KeyboardLayout
}

func DefaultConfig() AppConfig {
	return AppConfig{
		MIDIPath:       defaultMIDIPath(),
		Mode:           ModeHold,
		Speed:          1.2,
		LeadIn:         750 * time.Millisecond,
		TapDuration:    22 * time.Millisecond,
		InterKeyGap:    0,
		HotkeyCode:     defaultHotkeyKeyCode(),
		ConsumeHotkey:  true,
		KeyboardLayout: LayoutGerman,
	}
}

func (cfg AppConfig) SongOptions() SongOptions {
	return SongOptions{
		Mode:           cfg.Mode,
		Speed:          cfg.Speed,
		Transpose:      cfg.Transpose,
		DisableSustain: cfg.DisableSustain,
		AutoSustain:    cfg.AutoSustain,
	}
}

func (cfg AppConfig) PlayerOptions() PlayerOptions {
	return PlayerOptions{
		LeadIn:      cfg.LeadIn,
		TapDuration: cfg.TapDuration,
		InterKeyGap: cfg.InterKeyGap,
	}
}
