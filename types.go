package main

import "time"

type PlaybackMode string

const (
	ModeTap  PlaybackMode = "tap"
	ModeHold PlaybackMode = "hold"
)

type KeyboardLayout string

const (
	LayoutGerman  KeyboardLayout = "german"
	LayoutEnglish KeyboardLayout = "english"
)

func NormalizeKeyboardLayout(value string) KeyboardLayout {
	switch KeyboardLayout(value) {
	case LayoutEnglish:
		return LayoutEnglish
	case LayoutGerman:
		return LayoutGerman
	default:
		return LayoutGerman
	}
}

type KeyStroke struct {
	Label   string
	KeyCode uint16
	Shift   bool
}

type ActionKind int

const (
	ActionTap ActionKind = iota
	ActionDown
	ActionUp
)

type Action struct {
	At     time.Duration
	Kind   ActionKind
	Note   int
	Stroke KeyStroke
}

type Song struct {
	Path     string
	Actions  []Action
	Duration time.Duration
	Stats    SongStats
}

type SongStats struct {
	Format            int
	Tracks            int
	TicksPerQuarter   int
	TempoChanges      int
	ControlChanges    int
	SustainDowns      int
	SustainUps        int
	NoteOns           int
	NoteOffs          int
	MappedActions     int
	SkippedOutOfRange int
	SkippedUnmapped   int
	LowestNote        int
	HighestNote       int
}

type SongOptions struct {
	Mode           PlaybackMode
	Speed          float64
	Transpose      int
	DisableSustain bool
	AutoSustain    time.Duration
}

func (s SongStats) HasNoteRange() bool {
	return s.LowestNote > 0 && s.HighestNote > 0
}
