package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type guiRuntime struct {
	mu     sync.Mutex
	cfg    AppConfig
	song   *Song
	player *Player
}

type guiHotkeyEvent struct {
	toggle bool
	slot   int
}

const emptySongLabel = "(empty)"

func RunGUI(initial AppConfig, keyMap map[int]KeyStroke) {
	guiApp := app.NewWithID("dev.codex.roblox-midi-piano")
	window := guiApp.NewWindow("Roblox MIDI Piano")
	window.Resize(fyne.NewSize(820, 680))

	state := &guiRuntime{cfg: initial}
	settings := LoadUserSettings()
	hotkeyEvents := make(chan guiHotkeyEvent, 8)

	pathEntry := widget.NewEntry()
	pathEntry.SetText(initial.MIDIPath)

	statusLabel := widget.NewLabel("Idle")
	statusLabel.Wrapping = fyne.TextWrapWord

	summaryLabel := widget.NewLabel("")
	summaryLabel.TextStyle = fyne.TextStyle{Monospace: true}
	summaryLabel.Wrapping = fyne.TextWrapWord

	speedLabel := widget.NewLabel(formatSpeed(initial.Speed))
	speedSlider := widget.NewSlider(0.5, 2.0)
	speedSlider.Step = 0.05
	speedSlider.Value = initial.Speed
	speedSlider.OnChanged = func(value float64) {
		speedLabel.SetText(formatSpeed(value))
	}

	transposeLabel := widget.NewLabel(formatTranspose(initial.Transpose))
	transposeSlider := widget.NewSlider(-24, 24)
	transposeSlider.Step = 1
	transposeSlider.Value = float64(initial.Transpose)
	transposeSlider.OnChanged = func(value float64) {
		transposeLabel.SetText(formatTranspose(int(value)))
	}

	autoSustainLabel := widget.NewLabel(formatMilliseconds(initial.AutoSustain))
	autoSustainSlider := widget.NewSlider(0, 2000)
	autoSustainSlider.Step = 100
	autoSustainSlider.Value = float64(initial.AutoSustain / time.Millisecond)
	autoSustainSlider.OnChanged = func(value float64) {
		autoSustainLabel.SetText(formatMilliseconds(time.Duration(value) * time.Millisecond))
	}

	leadInLabel := widget.NewLabel(formatMilliseconds(initial.LeadIn))
	leadInSlider := widget.NewSlider(0, 3000)
	leadInSlider.Step = 250
	leadInSlider.Value = float64(initial.LeadIn / time.Millisecond)
	leadInSlider.OnChanged = func(value float64) {
		leadInLabel.SetText(formatMilliseconds(time.Duration(value) * time.Millisecond))
	}

	tapDurationLabel := widget.NewLabel(formatMilliseconds(initial.TapDuration))
	tapDurationSlider := widget.NewSlider(5, 120)
	tapDurationSlider.Step = 1
	tapDurationSlider.Value = float64(initial.TapDuration / time.Millisecond)
	tapDurationSlider.OnChanged = func(value float64) {
		tapDurationLabel.SetText(formatMilliseconds(time.Duration(value) * time.Millisecond))
	}

	interKeyGapLabel := widget.NewLabel(formatMilliseconds(initial.InterKeyGap))
	interKeyGapSlider := widget.NewSlider(0, 15)
	interKeyGapSlider.Step = 1
	interKeyGapSlider.Value = float64(initial.InterKeyGap / time.Millisecond)
	interKeyGapSlider.OnChanged = func(value float64) {
		interKeyGapLabel.SetText(formatMilliseconds(time.Duration(value) * time.Millisecond))
	}

	modeRadio := widget.NewRadioGroup([]string{string(ModeHold), string(ModeTap)}, nil)
	modeRadio.Horizontal = true
	modeRadio.SetSelected(string(initial.Mode))

	disableSustainCheck := widget.NewCheck("Disable MIDI sustain", nil)
	disableSustainCheck.SetChecked(initial.DisableSustain)

	consumeHotkeyCheck := widget.NewCheck("Consume start/stop", nil)
	consumeHotkeyCheck.SetChecked(initial.ConsumeHotkey)

	hotkeyEntry := widget.NewEntry()
	hotkeyEntry.SetText(strconv.Itoa(initial.HotkeyCode))
	hotkeyEntry.SetPlaceHolder(platformHotkeyHelp())

	var browseButton *widget.Button
	var loadButton *widget.Button
	var quickButton *widget.Button
	var settingsButton *widget.Button
	var startButton *widget.Button
	var settingsWindow fyne.Window
	var refreshSlotButtons func()
	var refreshSettingsSelects func()
	var configureGlobalHotkeys func(AppConfig)
	var loadSong func(bool) bool
	var selectSongPath func(string, bool, bool) bool
	var triggerSlot func(int)

	slotButtons := make([]*widget.Button, 9)

	readConfig := func() (AppConfig, error) {
		cfg := initial
		cfg.MIDIPath = strings.TrimSpace(pathEntry.Text)
		cfg.Mode = PlaybackMode(modeRadio.Selected)
		cfg.Speed = roundSlider(speedSlider.Value, 2)
		cfg.Transpose = int(transposeSlider.Value)
		cfg.DisableSustain = disableSustainCheck.Checked
		cfg.AutoSustain = time.Duration(autoSustainSlider.Value) * time.Millisecond
		cfg.LeadIn = time.Duration(leadInSlider.Value) * time.Millisecond
		cfg.TapDuration = time.Duration(tapDurationSlider.Value) * time.Millisecond
		cfg.InterKeyGap = time.Duration(interKeyGapSlider.Value) * time.Millisecond
		cfg.ConsumeHotkey = consumeHotkeyCheck.Checked

		if cfg.Mode != ModeHold && cfg.Mode != ModeTap {
			return cfg, fmt.Errorf("choose hold or tap mode")
		}
		if cfg.MIDIPath == "" {
			return cfg, fmt.Errorf("choose a MIDI file")
		}
		hotkeyCode, err := parseKeyCode(strings.TrimSpace(hotkeyEntry.Text))
		if err != nil {
			return cfg, fmt.Errorf("start/stop keycode must be a number")
		}
		cfg.HotkeyCode = hotkeyCode
		return cfg, nil
	}

	configureGlobalHotkeys = func(cfg AppConfig) {
		bindings := []HotkeyBinding{{
			KeyCode: cfg.HotkeyCode,
			Consume: cfg.ConsumeHotkey,
			OnPress: func() {
				select {
				case hotkeyEvents <- guiHotkeyEvent{toggle: true}:
				default:
				}
			},
		}}

		if settings.EnableSlotHotkeys {
			digitKeyCodes := platformDigitHotkeyCodes()
			for index, path := range settings.SlotPaths {
				if strings.TrimSpace(path) == "" {
					continue
				}
				slot := index
				bindings = append(bindings, HotkeyBinding{
					KeyCode: digitKeyCodes[index],
					Consume: settings.ConsumeSlotHotkeys,
					OnPress: func() {
						select {
						case hotkeyEvents <- guiHotkeyEvent{slot: slot + 1}:
						default:
						}
					},
				})
			}
		}

		ConfigureHotkeyBindings(bindings, 250*time.Millisecond)
	}

	saveSettings := func() {
		if err := SaveUserSettings(settings); err != nil {
			statusLabel.SetText("Settings save failed")
			if settingsWindow != nil {
				dialog.ShowError(err, settingsWindow)
			}
			return
		}
		state.mu.Lock()
		cfg := state.cfg
		state.mu.Unlock()
		configureGlobalHotkeys(cfg)
		if refreshSlotButtons != nil {
			refreshSlotButtons()
		}
	}

	isPlaying := func() bool {
		state.mu.Lock()
		player := state.player
		state.mu.Unlock()
		return player != nil && player.IsPlaying()
	}

	updateButtons := func() {
		playing := isPlaying()
		if playing {
			startButton.SetText("Stop")
			startButton.SetIcon(theme.MediaStopIcon())
			statusLabel.SetText("Playing")
			return
		}
		startButton.SetText("Start")
		startButton.SetIcon(theme.MediaPlayIcon())
		if statusLabel.Text == "Playing" || statusLabel.Text == "Starting" {
			statusLabel.SetText("Idle")
		}
	}

	loadSong = func(showErrors bool) bool {
		cfg, err := readConfig()
		if err != nil {
			if showErrors {
				dialog.ShowError(err, window)
			}
			statusLabel.SetText(err.Error())
			return false
		}

		song, err := LoadSong(cfg.MIDIPath, keyMap, cfg.SongOptions())
		if err != nil {
			if showErrors {
				dialog.ShowError(err, window)
			}
			statusLabel.SetText("Load failed")
			return false
		}
		if len(song.Actions) == 0 {
			err := fmt.Errorf("nothing to play after mapping")
			if showErrors {
				dialog.ShowError(err, window)
			}
			statusLabel.SetText(err.Error())
			return false
		}

		state.mu.Lock()
		if state.player != nil && state.player.IsPlaying() {
			state.player.Stop()
		}
		state.cfg = cfg
		state.song = song
		state.player = NewPlayer(song, cfg.PlayerOptions())
		state.mu.Unlock()

		configureGlobalHotkeys(cfg)
		summaryLabel.SetText(formatSongSummary(song, cfg.Mode, cfg.Speed, cfg.Transpose, !cfg.DisableSustain, cfg.AutoSustain))
		statusLabel.SetText("Loaded")
		updateButtons()
		return true
	}

	selectSongPath = func(path string, showErrors bool, restartIfPlaying bool) bool {
		path = resolveSongPath(path)
		if strings.TrimSpace(path) == "" {
			statusLabel.SetText("Slot is empty")
			return false
		}

		wasPlaying := restartIfPlaying && isPlaying()
		if wasPlaying {
			state.mu.Lock()
			player := state.player
			state.mu.Unlock()
			if player != nil {
				player.Stop()
			}
		}

		pathEntry.SetText(path)
		if !loadSong(showErrors) {
			return false
		}

		if wasPlaying {
			state.mu.Lock()
			player := state.player
			state.mu.Unlock()
			if player != nil {
				player.Start()
				statusLabel.SetText("Starting")
			}
		}
		updateButtons()
		return true
	}

	triggerSlot = func(slot int) {
		if slot < 1 || slot > len(settings.SlotPaths) {
			return
		}
		path := settings.SlotPaths[slot-1]
		if strings.TrimSpace(path) == "" {
			statusLabel.SetText(fmt.Sprintf("Slot %d empty", slot))
			return
		}
		if selectSongPath(path, true, true) {
			statusLabel.SetText(fmt.Sprintf("Slot %d loaded", slot))
		}
	}

	browseButton = widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), func() {
		browseButton.Disable()
		go func() {
			path, ok, err := OpenNativeMIDIFile()
			fyne.Do(func() {
				browseButton.Enable()
				if err != nil {
					dialog.ShowError(err, window)
					statusLabel.SetText("Browse failed")
					return
				}
				if !ok {
					return
				}
				if !IsMIDIPath(path) {
					err := fmt.Errorf("choose a .mid or .midi file")
					dialog.ShowError(err, window)
					statusLabel.SetText(err.Error())
					return
				}
				selectSongPath(path, false, false)
				if refreshSettingsSelects != nil {
					refreshSettingsSelects()
				}
			})
		}()
	})

	quickButton = widget.NewButtonWithIcon("Quick Select", theme.MenuDropDownIcon(), func() {
		extras := []string{pathEntry.Text}
		extras = append(extras, settings.SlotPaths[:]...)
		choices := DiscoverMIDIFiles(extras...)
		if len(choices) == 0 {
			statusLabel.SetText("No MIDI files found")
			return
		}

		items := make([]*fyne.MenuItem, 0, len(choices))
		for _, choice := range choices {
			choice := choice
			items = append(items, fyne.NewMenuItem(choice.Label, func() {
				selectSongPath(choice.Path, true, false)
			}))
		}
		menu := fyne.NewMenu("", items...)
		popup := widget.NewPopUpMenu(menu, window.Canvas())
		popup.ShowAtPosition(quickButton.Position().Add(fyne.NewPos(0, quickButton.Size().Height)))
	})

	loadButton = widget.NewButtonWithIcon("Load", theme.ViewRefreshIcon(), func() {
		loadSong(true)
		if refreshSettingsSelects != nil {
			refreshSettingsSelects()
		}
	})

	startOrStop := func() {
		state.mu.Lock()
		player := state.player
		state.mu.Unlock()

		if player != nil && player.IsPlaying() {
			player.Stop()
			statusLabel.SetText("Stopped")
			updateButtons()
			return
		}

		if !loadSong(true) {
			return
		}

		state.mu.Lock()
		player = state.player
		state.mu.Unlock()
		player.Start()
		statusLabel.SetText("Starting")
		updateButtons()
	}

	startButton = widget.NewButtonWithIcon("Start", theme.MediaPlayIcon(), startOrStop)

	refreshSlotButtons = func() {
		for index, button := range slotButtons {
			button.SetText(strconv.Itoa(index + 1))
			if strings.TrimSpace(settings.SlotPaths[index]) == "" {
				button.Disable()
			} else {
				button.Enable()
			}
		}
	}

	for index := range slotButtons {
		slot := index + 1
		slotButtons[index] = widget.NewButton(strconv.Itoa(slot), func() {
			triggerSlot(slot)
		})
	}
	refreshSlotButtons()

	openSettings := func() {
		if settingsWindow != nil {
			settingsWindow.Show()
			return
		}

		settingsWindow = guiApp.NewWindow("Song Slots")
		settingsWindow.Resize(fyne.NewSize(620, 520))
		settingsStatus := widget.NewLabel("")
		settingsStatus.Wrapping = fyne.TextWrapWord

		enableSlotHotkeys := widget.NewCheck("Enable 1-9 hotkeys", func(value bool) {
			settings.EnableSlotHotkeys = value
			saveSettings()
			settingsStatus.SetText("Saved")
		})
		enableSlotHotkeys.SetChecked(settings.EnableSlotHotkeys)

		consumeSlotHotkeys := widget.NewCheck("Consume 1-9 hotkeys", func(value bool) {
			settings.ConsumeSlotHotkeys = value
			saveSettings()
			settingsStatus.SetText("Saved")
		})
		consumeSlotHotkeys.SetChecked(settings.ConsumeSlotHotkeys)

		slotSelects := make([]*widget.Select, len(settings.SlotPaths))
		var labelToPath map[string]string
		refreshing := false

		buildOptions := func() []string {
			extras := []string{pathEntry.Text}
			extras = append(extras, settings.SlotPaths[:]...)
			choices := DiscoverMIDIFiles(extras...)
			labelToPath = map[string]string{emptySongLabel: ""}
			labels := []string{emptySongLabel}
			for _, choice := range choices {
				labelToPath[choice.Label] = choice.Path
				labels = append(labels, choice.Label)
			}
			return labels
		}

		labelForPath := func(path string) string {
			if strings.TrimSpace(path) == "" {
				return emptySongLabel
			}
			resolved := resolveSongPath(path)
			for label, candidate := range labelToPath {
				if candidate != "" && samePath(candidate, resolved) {
					return label
				}
			}
			return emptySongLabel
		}

		options := buildOptions()
		formItems := make([]*widget.FormItem, 0, len(settings.SlotPaths))
		for index := range settings.SlotPaths {
			slot := index
			selectWidget := widget.NewSelect(options, func(label string) {
				if refreshing {
					return
				}
				path := labelToPath[label]
				settings.SlotPaths[slot] = storeSongPath(path)
				saveSettings()
				settingsStatus.SetText("Saved")
			})
			selectWidget.SetSelected(labelForPath(settings.SlotPaths[index]))
			slotSelects[index] = selectWidget
			formItems = append(formItems, widget.NewFormItem(strconv.Itoa(index+1), selectWidget))
		}

		refreshSettingsSelects = func() {
			refreshing = true
			options := buildOptions()
			for index, selectWidget := range slotSelects {
				selectWidget.Options = options
				selectWidget.SetSelected(labelForPath(settings.SlotPaths[index]))
				selectWidget.Refresh()
			}
			refreshing = false
		}

		rescanButton := widget.NewButtonWithIcon("Rescan", theme.ViewRefreshIcon(), func() {
			refreshSettingsSelects()
			settingsStatus.SetText("Rescanned")
		})

		settingsContent := container.NewVBox(
			widget.NewLabelWithStyle("Song Slots", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewGridWithColumns(2, enableSlotHotkeys, consumeSlotHotkeys),
			widget.NewSeparator(),
			widget.NewForm(formItems...),
			container.NewHBox(rescanButton, settingsStatus),
		)

		settingsWindow.SetOnClosed(func() {
			settingsWindow = nil
			refreshSettingsSelects = nil
		})
		settingsWindow.SetContent(container.NewPadded(settingsContent))
		settingsWindow.Show()
	}

	settingsButton = widget.NewButtonWithIcon("", theme.SettingsIcon(), openSettings)

	window.SetOnDropped(func(_ fyne.Position, uris []fyne.URI) {
		go func() {
			sourcePath := ""
			for _, uri := range uris {
				if IsMIDIPath(uri.Path()) {
					sourcePath = uri.Path()
					break
				}
			}
			if sourcePath == "" {
				fyne.Do(func() {
					err := fmt.Errorf("drop a .mid or .midi file")
					dialog.ShowError(err, window)
					statusLabel.SetText(err.Error())
				})
				return
			}

			importedPath, err := ImportMIDIFile(sourcePath)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, window)
					statusLabel.SetText("Import failed")
					return
				}
				pathEntry.SetText(importedPath)
				statusLabel.SetText("Imported")
				loadSong(true)
				if refreshSettingsSelects != nil {
					refreshSettingsSelects()
				}
			})
		}()
	})

	configureGlobalHotkeys(initial)
	go func() {
		if err := RunGlobalHotkeyListener(); err != nil {
			fyne.Do(func() {
				statusLabel.SetText(err.Error())
				dialog.ShowError(err, window)
			})
		}
	}()

	closed := make(chan struct{})
	var closeOnce sync.Once
	stopAll := func() {
		closeOnce.Do(func() {
			state.mu.Lock()
			player := state.player
			state.mu.Unlock()
			if player != nil {
				player.Stop()
			}
			close(closed)
		})
	}

	go func() {
		for {
			select {
			case event := <-hotkeyEvents:
				if event.toggle {
					fyne.Do(startOrStop)
					continue
				}
				if event.slot > 0 {
					slot := event.slot
					fyne.Do(func() {
						triggerSlot(slot)
					})
				}
			case <-closed:
				return
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fyne.Do(updateButtons)
			case <-closed:
				return
			}
		}
	}()

	window.SetCloseIntercept(func() {
		stopAll()
		window.Close()
	})

	songRow := container.NewBorder(nil, nil, nil, container.NewHBox(quickButton, browseButton, loadButton), pathEntry)
	playRow := container.NewGridWithColumns(2, startButton, statusLabel)
	hotbarRow := container.NewGridWithColumns(9, slotButtons[0], slotButtons[1], slotButtons[2], slotButtons[3], slotButtons[4], slotButtons[5], slotButtons[6], slotButtons[7], slotButtons[8])

	playbackForm := widget.NewForm(
		widget.NewFormItem("Speed", container.NewBorder(nil, nil, nil, speedLabel, speedSlider)),
		widget.NewFormItem("Transpose", container.NewBorder(nil, nil, nil, transposeLabel, transposeSlider)),
		widget.NewFormItem("Mode", modeRadio),
		widget.NewFormItem("Auto Sustain", container.NewBorder(nil, nil, nil, autoSustainLabel, autoSustainSlider)),
		widget.NewFormItem("Lead In", container.NewBorder(nil, nil, nil, leadInLabel, leadInSlider)),
	)

	advancedForm := widget.NewForm(
		widget.NewFormItem("Tap Duration", container.NewBorder(nil, nil, nil, tapDurationLabel, tapDurationSlider)),
		widget.NewFormItem("Inter-Key Gap", container.NewBorder(nil, nil, nil, interKeyGapLabel, interKeyGapSlider)),
		widget.NewFormItem("Start/Stop Keycode", hotkeyEntry),
		widget.NewFormItem("", disableSustainCheck),
		widget.NewFormItem("", consumeHotkeyCheck),
	)

	advanced := widget.NewAccordion(
		widget.NewAccordionItem("Advanced", advancedForm),
	)

	header := container.NewBorder(nil, nil, nil, settingsButton,
		widget.NewLabelWithStyle("Roblox MIDI Piano", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		widget.NewLabel("Song"),
		songRow,
		hotbarRow,
		playRow,
		widget.NewSeparator(),
		widget.NewLabel("Playback"),
		playbackForm,
		advanced,
		widget.NewSeparator(),
		widget.NewLabel("Summary"),
		container.NewVScroll(summaryLabel),
	)

	window.SetContent(container.NewPadded(content))
	loadSong(false)
	window.ShowAndRun()
}

func formatSpeed(speed float64) string {
	return fmt.Sprintf("%.2fx", speed)
}

func formatTranspose(transpose int) string {
	return fmt.Sprintf("%+d", transpose)
}

func formatMilliseconds(duration time.Duration) string {
	return fmt.Sprintf("%dms", duration/time.Millisecond)
}

func roundSlider(value float64, precision int) float64 {
	scale := 1.0
	for range precision {
		scale *= 10
	}
	return float64(int(value*scale+0.5)) / scale
}

func parseKeyCode(value string) (int, error) {
	keyCode, err := strconv.ParseInt(value, 0, 32)
	if err != nil {
		return 0, err
	}
	if keyCode <= 0 {
		return 0, fmt.Errorf("keycode must be greater than zero")
	}
	return int(keyCode), nil
}
