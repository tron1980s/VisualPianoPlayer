package main

import (
	"fmt"
	"strings"
)

const (
	lowestPianoMIDINote  = 21
	highestPianoMIDINote = 108
)

var whiteNoteClasses = map[int]bool{
	0:  true, // C
	2:  true, // D
	4:  true, // E
	5:  true, // F
	7:  true, // G
	9:  true, // A
	11: true, // B
}

var shiftedUSLabels = map[rune]rune{
	'!': '1',
	'@': '2',
	'#': '3',
	'$': '4',
	'%': '5',
	'^': '6',
	'&': '7',
	'*': '8',
	'(': '9',
	')': '0',
}

func Roblox88KeyMap(layout KeyboardLayout) (map[int]KeyStroke, error) {
	layout = NormalizeKeyboardLayout(string(layout))
	whiteLabels := strings.Fields("1 3 4 6 8 9 q e t 1 2 3 4 5 6 7 8 9 0 q w e r t y u i o p a s d f g h j k l z x c v b n m u o p s f h j")
	blackLabels := strings.Fields("2 5 7 0 w r ! @ $ % ^ * ( Q W E T Y I O P S D G H J L Z C V B y i a d g")
	swapLabels(whiteLabels, "y", "2")
	swapLabels(blackLabels, "y", "2")

	keyMap := make(map[int]KeyStroke, highestPianoMIDINote-lowestPianoMIDINote+1)
	whiteIndex := 0
	blackIndex := 0

	for note := lowestPianoMIDINote; note <= highestPianoMIDINote; note++ {
		var label string
		if isWhiteNote(note) {
			if whiteIndex >= len(whiteLabels) {
				return nil, fmt.Errorf("roblox white key map is short at MIDI note %d", note)
			}
			label = whiteLabels[whiteIndex]
			whiteIndex++
		} else {
			if blackIndex >= len(blackLabels) {
				return nil, fmt.Errorf("roblox black key map is short at MIDI note %d", note)
			}
			label = blackLabels[blackIndex]
			blackIndex++
		}

		stroke, err := strokeFromRobloxLabel(label, layout)
		if err != nil {
			return nil, fmt.Errorf("MIDI note %d label %q: %w", note, label, err)
		}
		keyMap[note] = stroke
	}

	if whiteIndex != len(whiteLabels) {
		return nil, fmt.Errorf("roblox white key map has %d unused labels", len(whiteLabels)-whiteIndex)
	}
	if blackIndex != len(blackLabels) {
		return nil, fmt.Errorf("roblox black key map has %d unused labels", len(blackLabels)-blackIndex)
	}

	return keyMap, nil
}

func swapLabels(labels []string, a string, b string) {
	for i, label := range labels {
		switch label {
		case a:
			labels[i] = b
		case b:
			labels[i] = a
		}
	}
}

func isWhiteNote(midiNote int) bool {
	return whiteNoteClasses[midiNote%12]
}

func strokeFromRobloxLabel(label string, layout KeyboardLayout) (KeyStroke, error) {
	runes := []rune(label)
	if len(runes) != 1 {
		return KeyStroke{}, fmt.Errorf("expected one-character label")
	}

	r := runes[0]
	shift := false
	base := r

	if r >= 'A' && r <= 'Z' {
		shift = true
		base = r + ('a' - 'A')
	} else if shiftedBase, ok := shiftedUSLabels[r]; ok {
		shift = true
		base = shiftedBase
	}
	base = baseForKeyboardLayout(base, layout)

	keyCode, ok := platformKeyCodeForBase(base)
	if !ok {
		return KeyStroke{}, fmt.Errorf("no platform keycode for %q", base)
	}

	return KeyStroke{
		Label:   label,
		KeyCode: keyCode,
		Shift:   shift,
	}, nil
}

func baseForKeyboardLayout(base rune, layout KeyboardLayout) rune {
	if NormalizeKeyboardLayout(string(layout)) != LayoutGerman {
		return base
	}
	switch base {
	case 'y':
		return 'z'
	case 'z':
		return 'y'
	default:
		return base
	}
}

func platformDigitHotkeyCodes() [9]int {
	var keyCodes [9]int
	for digit := 1; digit <= 9; digit++ {
		keyCode, _ := platformKeyCodeForBase(rune('0' + digit))
		keyCodes[digit-1] = int(keyCode)
	}
	return keyCodes
}

func NoteName(note int) string {
	names := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	return fmt.Sprintf("%s%d", names[note%12], note/12-1)
}
