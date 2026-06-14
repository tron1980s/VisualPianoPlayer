package main

import "testing"

func TestKeyboardLayoutSwapsYAndZ(t *testing.T) {
	englishY, err := strokeFromRobloxLabel("y", LayoutEnglish)
	if err != nil {
		t.Fatal(err)
	}
	germanY, err := strokeFromRobloxLabel("y", LayoutGerman)
	if err != nil {
		t.Fatal(err)
	}
	englishZ, err := strokeFromRobloxLabel("z", LayoutEnglish)
	if err != nil {
		t.Fatal(err)
	}

	if germanY.KeyCode != englishZ.KeyCode {
		t.Fatalf("German y keycode = %d, want English z keycode %d", germanY.KeyCode, englishZ.KeyCode)
	}
	if englishY.KeyCode == germanY.KeyCode {
		t.Fatalf("German y keycode should differ from English y keycode %d", englishY.KeyCode)
	}
}
