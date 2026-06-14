package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

type rawEventKind int

const (
	rawNoteOn rawEventKind = iota
	rawNoteOff
	rawTempo
	rawControlChange
)

type rawMIDIEvent struct {
	Tick              int64
	Order             int
	Kind              rawEventKind
	Note              int
	Velocity          int
	Controller        int
	Value             int
	TempoUSPerQuarter int
}

type midiFile struct {
	Format          int
	Tracks          int
	TicksPerQuarter int
	Events          []rawMIDIEvent
	TempoChanges    int
	ControlChanges  int
	SustainDowns    int
	SustainUps      int
	NoteOns         int
	NoteOffs        int
}

func parseMIDIFile(path string) (*midiFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pos := 0

	chunkID, chunkData, err := readChunk(data, &pos)
	if err != nil {
		return nil, err
	}
	if chunkID != "MThd" {
		return nil, fmt.Errorf("expected MThd header, got %q", chunkID)
	}
	if len(chunkData) < 6 {
		return nil, fmt.Errorf("MThd header is too short")
	}

	format := int(binary.BigEndian.Uint16(chunkData[0:2]))
	tracks := int(binary.BigEndian.Uint16(chunkData[2:4]))
	divisionRaw := binary.BigEndian.Uint16(chunkData[4:6])
	if divisionRaw&0x8000 != 0 {
		return nil, fmt.Errorf("SMPTE MIDI timing is not supported")
	}
	ticksPerQuarter := int(divisionRaw)
	if ticksPerQuarter <= 0 {
		return nil, fmt.Errorf("invalid ticks-per-quarter value %d", ticksPerQuarter)
	}

	parsed := &midiFile{
		Format:          format,
		Tracks:          tracks,
		TicksPerQuarter: ticksPerQuarter,
	}

	order := 0
	for trackIndex := 0; trackIndex < tracks; trackIndex++ {
		chunkID, chunkData, err = readChunk(data, &pos)
		if err != nil {
			return nil, fmt.Errorf("track %d: %w", trackIndex+1, err)
		}
		if chunkID != "MTrk" {
			return nil, fmt.Errorf("track %d: expected MTrk chunk, got %q", trackIndex+1, chunkID)
		}

		events, err := parseTrack(chunkData, &order)
		if err != nil {
			return nil, fmt.Errorf("track %d: %w", trackIndex+1, err)
		}
		for _, event := range events {
			switch event.Kind {
			case rawTempo:
				parsed.TempoChanges++
			case rawControlChange:
				parsed.ControlChanges++
				if event.Controller == 64 {
					if event.Value >= 64 {
						parsed.SustainDowns++
					} else {
						parsed.SustainUps++
					}
				}
			case rawNoteOn:
				parsed.NoteOns++
			case rawNoteOff:
				parsed.NoteOffs++
			}
			parsed.Events = append(parsed.Events, event)
		}
	}

	sort.SliceStable(parsed.Events, func(i, j int) bool {
		if parsed.Events[i].Tick != parsed.Events[j].Tick {
			return parsed.Events[i].Tick < parsed.Events[j].Tick
		}
		return parsed.Events[i].Order < parsed.Events[j].Order
	})

	return parsed, nil
}

func readChunk(data []byte, pos *int) (string, []byte, error) {
	if *pos+8 > len(data) {
		return "", nil, fmt.Errorf("unexpected end of file while reading chunk header")
	}
	id := string(data[*pos : *pos+4])
	length := int(binary.BigEndian.Uint32(data[*pos+4 : *pos+8]))
	*pos += 8
	if length < 0 || *pos+length > len(data) {
		return "", nil, fmt.Errorf("chunk %q length exceeds file size", id)
	}
	chunkData := data[*pos : *pos+length]
	*pos += length
	return id, chunkData, nil
}

func parseTrack(data []byte, order *int) ([]rawMIDIEvent, error) {
	var events []rawMIDIEvent
	pos := 0
	var tick int64
	var runningStatus byte

	for pos < len(data) {
		delta, err := readVariableLength(data, &pos)
		if err != nil {
			return nil, err
		}
		tick += int64(delta)

		if pos >= len(data) {
			return nil, fmt.Errorf("unexpected end of track after delta time")
		}

		status := data[pos]
		if status < 0x80 {
			if runningStatus == 0 {
				return nil, fmt.Errorf("running status used before a status byte")
			}
			status = runningStatus
		} else {
			pos++
			if status < 0xF0 {
				runningStatus = status
			}
		}

		switch {
		case status >= 0x80 && status <= 0xEF:
			kind := status & 0xF0
			dataBytes := 2
			if kind == 0xC0 || kind == 0xD0 {
				dataBytes = 1
			}
			if pos+dataBytes > len(data) {
				return nil, fmt.Errorf("unexpected end of track in MIDI event 0x%X", status)
			}

			first := data[pos]
			second := byte(0)
			if dataBytes == 2 {
				second = data[pos+1]
			}
			pos += dataBytes

			switch kind {
			case 0xB0:
				events = append(events, rawMIDIEvent{
					Tick:       tick,
					Order:      nextOrder(order),
					Kind:       rawControlChange,
					Controller: int(first),
					Value:      int(second),
				})
			case 0x80:
				events = append(events, rawMIDIEvent{
					Tick:     tick,
					Order:    nextOrder(order),
					Kind:     rawNoteOff,
					Note:     int(first),
					Velocity: int(second),
				})
			case 0x90:
				eventKind := rawNoteOn
				if second == 0 {
					eventKind = rawNoteOff
				}
				events = append(events, rawMIDIEvent{
					Tick:     tick,
					Order:    nextOrder(order),
					Kind:     eventKind,
					Note:     int(first),
					Velocity: int(second),
				})
			}

		case status == 0xFF:
			if pos >= len(data) {
				return nil, fmt.Errorf("unexpected end of track in meta event")
			}
			metaType := data[pos]
			pos++
			length, err := readVariableLength(data, &pos)
			if err != nil {
				return nil, err
			}
			if pos+int(length) > len(data) {
				return nil, fmt.Errorf("meta event length exceeds track size")
			}
			payload := data[pos : pos+int(length)]
			pos += int(length)

			if metaType == 0x2F {
				return events, nil
			}
			if metaType == 0x51 && len(payload) == 3 {
				tempo := int(payload[0])<<16 | int(payload[1])<<8 | int(payload[2])
				events = append(events, rawMIDIEvent{
					Tick:              tick,
					Order:             nextOrder(order),
					Kind:              rawTempo,
					TempoUSPerQuarter: tempo,
				})
			}

		case status == 0xF0 || status == 0xF7:
			length, err := readVariableLength(data, &pos)
			if err != nil {
				return nil, err
			}
			if pos+int(length) > len(data) {
				return nil, fmt.Errorf("sysex event length exceeds track size")
			}
			pos += int(length)

		default:
			return nil, fmt.Errorf("unsupported MIDI status byte 0x%X", status)
		}
	}

	return events, nil
}

func readVariableLength(data []byte, pos *int) (uint32, error) {
	var value uint32
	for i := 0; i < 4; i++ {
		if *pos >= len(data) {
			return 0, fmt.Errorf("unexpected end of data in variable-length value")
		}
		b := data[*pos]
		*pos++
		value = (value << 7) | uint32(b&0x7F)
		if b&0x80 == 0 {
			return value, nil
		}
	}
	return 0, fmt.Errorf("variable-length value is longer than four bytes")
}

func nextOrder(order *int) int {
	current := *order
	*order++
	return current
}
