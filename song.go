package main

import (
	"fmt"
	"math"
	"sort"
	"time"
)

func LoadSong(path string, keyMap map[int]KeyStroke, opts SongOptions) (*Song, error) {
	if opts.Mode == "" {
		opts.Mode = ModeHold
	}
	if opts.Speed <= 0 {
		return nil, fmt.Errorf("speed must be greater than zero")
	}

	midi, err := parseMIDIFile(path)
	if err != nil {
		return nil, err
	}
	midiHasSustain := midi.SustainDowns > 0 || midi.SustainUps > 0
	autoSustain := time.Duration(0)
	if !midiHasSustain && opts.AutoSustain > 0 {
		autoSustain = scaledDuration(opts.AutoSustain, opts.Speed)
	}

	stats := SongStats{
		Format:          midi.Format,
		Tracks:          midi.Tracks,
		TicksPerQuarter: midi.TicksPerQuarter,
		TempoChanges:    midi.TempoChanges,
		ControlChanges:  midi.ControlChanges,
		SustainDowns:    midi.SustainDowns,
		SustainUps:      midi.SustainUps,
		NoteOns:         midi.NoteOns,
		NoteOffs:        midi.NoteOffs,
	}

	tempo := 500000
	lastTick := int64(0)
	elapsedMicroseconds := int64(0)
	actions := make([]Action, 0, midi.NoteOns)
	activeNotes := make(map[int]int)
	sustainedNotes := make(map[int]KeyStroke)
	sustainDown := false

	for _, event := range midi.Events {
		if event.Tick < lastTick {
			return nil, fmt.Errorf("MIDI events are not sorted")
		}
		elapsedMicroseconds += (event.Tick - lastTick) * int64(tempo) / int64(midi.TicksPerQuarter)
		lastTick = event.Tick

		if event.Kind == rawTempo {
			if event.TempoUSPerQuarter > 0 {
				tempo = event.TempoUSPerQuarter
			}
			continue
		}

		at := scaledDuration(time.Duration(elapsedMicroseconds)*time.Microsecond, opts.Speed)
		if event.Kind == rawControlChange {
			if event.Controller == 64 && opts.Mode == ModeHold && !opts.DisableSustain {
				if event.Value >= 64 {
					sustainDown = true
				} else if sustainDown {
					sustainDown = false
					actions = appendSustainReleaseActions(actions, at, sustainedNotes, activeNotes)
					clear(sustainedNotes)
				}
			}
			continue
		}

		note := event.Note + opts.Transpose
		if note < lowestPianoMIDINote || note > highestPianoMIDINote {
			if event.Kind == rawNoteOn {
				stats.SkippedOutOfRange++
			}
			continue
		}

		stroke, ok := keyMap[note]
		if !ok {
			if event.Kind == rawNoteOn {
				stats.SkippedUnmapped++
			}
			continue
		}

		if event.Kind == rawNoteOn {
			stats.MappedActions++
			if stats.LowestNote == 0 || note < stats.LowestNote {
				stats.LowestNote = note
			}
			if note > stats.HighestNote {
				stats.HighestNote = note
			}
		}

		switch opts.Mode {
		case ModeTap:
			if event.Kind == rawNoteOn {
				actions = append(actions, Action{
					At:     at,
					Kind:   ActionTap,
					Note:   note,
					Stroke: stroke,
				})
			}
		case ModeHold:
			if event.Kind == rawNoteOn {
				if _, ok := sustainedNotes[note]; ok && activeNotes[note] == 0 {
					actions = append(actions, Action{
						At:     at,
						Kind:   ActionUp,
						Note:   note,
						Stroke: stroke,
					})
					delete(sustainedNotes, note)
				}
				activeNotes[note]++
				actions = append(actions, Action{
					At:     at,
					Kind:   ActionDown,
					Note:   note,
					Stroke: stroke,
				})
				continue
			}

			if activeNotes[note] > 0 {
				activeNotes[note]--
			}
			if sustainDown && activeNotes[note] == 0 && !opts.DisableSustain {
				sustainedNotes[note] = stroke
				continue
			}
			actions = append(actions, Action{
				At:     at + autoSustain,
				Kind:   ActionUp,
				Note:   note,
				Stroke: stroke,
			})
		default:
			return nil, fmt.Errorf("unknown playback mode %q", opts.Mode)
		}
	}

	sort.SliceStable(actions, func(i, j int) bool {
		if actions[i].At != actions[j].At {
			return actions[i].At < actions[j].At
		}
		return actionPriority(actions[i].Kind) < actionPriority(actions[j].Kind)
	})

	duration := time.Duration(0)
	if len(actions) > 0 {
		duration = actions[len(actions)-1].At
	}

	return &Song{
		Path:     path,
		Actions:  actions,
		Duration: duration,
		Stats:    stats,
	}, nil
}

func appendSustainReleaseActions(actions []Action, at time.Duration, sustained map[int]KeyStroke, active map[int]int) []Action {
	notes := make([]int, 0, len(sustained))
	for note := range sustained {
		if active[note] == 0 {
			notes = append(notes, note)
		}
	}
	sort.Ints(notes)
	for _, note := range notes {
		actions = append(actions, Action{
			At:     at,
			Kind:   ActionUp,
			Note:   note,
			Stroke: sustained[note],
		})
	}
	return actions
}

func scaledDuration(d time.Duration, speed float64) time.Duration {
	if speed == 1 {
		return d
	}
	return time.Duration(math.Round(float64(d) / speed))
}

func actionPriority(kind ActionKind) int {
	switch kind {
	case ActionUp:
		return 0
	case ActionDown:
		return 1
	case ActionTap:
		return 2
	default:
		return 3
	}
}
