package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	defaults := DefaultConfig()

	midiPath := flag.String("midi", defaults.MIDIPath, "path to the .mid/.midi file to play")
	modeFlag := flag.String("mode", string(defaults.Mode), "playback mode: tap or hold")
	speed := flag.Float64("speed", defaults.Speed, "playback speed multiplier; 2 is twice as fast, 0.5 is half speed")
	transpose := flag.Int("transpose", defaults.Transpose, "transpose MIDI notes by this many semitones before mapping to the Roblox piano")
	disableSustain := flag.Bool("disable-sustain", defaults.DisableSustain, "ignore MIDI sustain pedal CC64 events")
	autoSustain := flag.Duration("auto-sustain", defaults.AutoSustain, "extra sustain tail for dry MIDI files without CC64 pedal events, for example 900ms")
	leadIn := flag.Duration("lead-in", defaults.LeadIn, "delay after pressing the hotkey before playback starts")
	tapDuration := flag.Duration("tap-duration", defaults.TapDuration, "how long each note is held in tap mode")
	interKeyGap := flag.Duration("inter-key-gap", defaults.InterKeyGap, "tiny delay between keys inside a chord")
	hotkeyCode := flag.Int("hotkey-keycode", defaults.HotkeyCode, platformHotkeyHelp())
	consumeHotkey := flag.Bool("consume-hotkey", defaults.ConsumeHotkey, "prevent the hotkey press from reaching the focused app")
	keyboardLayout := flag.String("keyboard-layout", string(defaults.KeyboardLayout), "keyboard layout for y/z mapping: german or english")
	cli := flag.Bool("cli", false, "run the headless command-line hotkey player instead of the GUI")
	dryRun := flag.Bool("dry-run", false, "parse the MIDI and print what would be played without installing the hotkey")
	listMap := flag.Bool("list-map", false, "print the built-in Roblox 88-key MIDI map")
	flag.Parse()

	if flag.NArg() > 0 {
		*midiPath = flag.Arg(0)
	}

	mode := PlaybackMode(strings.ToLower(strings.TrimSpace(*modeFlag)))
	if mode != ModeTap && mode != ModeHold {
		exitf("unknown mode %q; use tap or hold", *modeFlag)
	}

	cfg := AppConfig{
		MIDIPath:       *midiPath,
		Mode:           mode,
		Speed:          *speed,
		Transpose:      *transpose,
		DisableSustain: *disableSustain,
		AutoSustain:    *autoSustain,
		LeadIn:         *leadIn,
		TapDuration:    *tapDuration,
		InterKeyGap:    *interKeyGap,
		HotkeyCode:     *hotkeyCode,
		ConsumeHotkey:  *consumeHotkey,
		KeyboardLayout: NormalizeKeyboardLayout(strings.ToLower(strings.TrimSpace(*keyboardLayout))),
	}

	keyMap, err := Roblox88KeyMap(cfg.KeyboardLayout)
	if err != nil {
		exitf("building Roblox key map: %v", err)
	}

	if *listMap {
		printMap(keyMap)
		return
	}

	if *dryRun || *cli {
		runCLI(cfg, keyMap, *dryRun)
		return
	}

	RunGUI(cfg)
}

func runCLI(cfg AppConfig, keyMap map[int]KeyStroke, dryRun bool) {
	if strings.TrimSpace(cfg.MIDIPath) == "" {
		exitf("no MIDI file found; pass one with -midi path/to/song.mid")
	}

	song, err := LoadSong(cfg.MIDIPath, keyMap, cfg.SongOptions())
	if err != nil {
		exitf("loading MIDI: %v", err)
	}

	printSongSummary(song, cfg.Mode, cfg.Speed, cfg.Transpose, !cfg.DisableSustain, cfg.AutoSustain)
	if dryRun {
		return
	}
	if len(song.Actions) == 0 {
		exitf("nothing to play after mapping; try a different transpose value")
	}

	player := NewPlayer(song, cfg.PlayerOptions())

	toggle := make(chan struct{}, 1)
	ConfigureHotkey(HotkeyConfig{
		KeyCode: cfg.HotkeyCode,
		Consume: cfg.ConsumeHotkey,
		OnPress: func() {
			select {
			case toggle <- struct{}{}:
			default:
			}
		},
	})

	go func() {
		for range toggle {
			player.Toggle()
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		player.Stop()
		fmt.Println("Bye.")
		os.Exit(0)
	}()

	fmt.Println("Press the start/stop hotkey to play or stop. Leave this terminal open; focus Roblox before starting.")
	fmt.Printf("%s hotkey keycode: %d. If it does not toggle, rerun with -hotkey-keycode <code>.\n", platformName(), cfg.HotkeyCode)

	if err := RunGlobalHotkeyListener(); err != nil {
		exitf("%v", err)
	}
}

func defaultMIDIPath() string {
	preferred := []string{
		"026-2004-Chopin - Etude Op-10 No-4 (SHYBAY).mid",
		"Philip Glass - Opening (better merged).mid",
		"Philip Glass - Opening.mid",
	}
	for _, path := range preferred {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	matches, err := filepath.Glob("*.mid")
	if err == nil && len(matches) > 0 {
		return matches[0]
	}
	matches, err = filepath.Glob("*.midi")
	if err == nil && len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func printSongSummary(song *Song, mode PlaybackMode, speed float64, transpose int, sustain bool, autoSustain time.Duration) {
	fmt.Print(formatSongSummary(song, mode, speed, transpose, sustain, autoSustain))
}

func formatSongSummary(song *Song, mode PlaybackMode, speed float64, transpose int, sustain bool, autoSustain time.Duration) string {
	absPath, err := filepath.Abs(song.Path)
	if err != nil {
		absPath = song.Path
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Loaded: %s\n", absPath)
	fmt.Fprintf(&b, "MIDI: format %d, %d track(s), %d ticks/quarter, %d tempo change(s)\n",
		song.Stats.Format, song.Stats.Tracks, song.Stats.TicksPerQuarter, song.Stats.TempoChanges)
	fmt.Fprintf(&b, "Playback: %s mode, speed %.2fx, transpose %+d, duration %s\n",
		mode, speed, transpose, song.Duration.Round(time.Millisecond))
	fmt.Fprintf(&b, "Notes: %d note-on events, %d mapped action(s), %d out of range, %d unmapped\n",
		song.Stats.NoteOns, song.Stats.MappedActions, song.Stats.SkippedOutOfRange, song.Stats.SkippedUnmapped)
	fmt.Fprintf(&b, "Pedal: %d sustain-down, %d sustain-up event(s), simulation %s\n",
		song.Stats.SustainDowns, song.Stats.SustainUps, enabledLabel(sustain && mode == ModeHold))
	if autoSustain > 0 {
		active := song.Stats.SustainDowns == 0 && song.Stats.SustainUps == 0 && mode == ModeHold
		fmt.Fprintf(&b, "Auto sustain: %s tail, %s\n", autoSustain, enabledLabel(active))
	}
	if song.Stats.HasNoteRange() {
		fmt.Fprintf(&b, "Mapped range: %s-%s\n", NoteName(song.Stats.LowestNote), NoteName(song.Stats.HighestNote))
	}
	return b.String()
}

func enabledLabel(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}

func printMap(keyMap map[int]KeyStroke) {
	for note := lowestPianoMIDINote; note <= highestPianoMIDINote; note++ {
		stroke := keyMap[note]
		shift := ""
		if stroke.Shift {
			shift = " + Shift"
		}
		fmt.Printf("%3d %-4s -> %-2s keycode %d%s\n", note, NoteName(note), stroke.Label, stroke.KeyCode, shift)
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
