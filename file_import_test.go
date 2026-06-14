package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportMIDIFileCopiesIntoImportFolder(t *testing.T) {
	workspace := t.TempDir()
	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "song.mid")
	if err := os.WriteFile(sourcePath, []byte("MThd"), 0644); err != nil {
		t.Fatal(err)
	}

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previousWD); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(workspace); err != nil {
		t.Fatal(err)
	}

	importedPath, err := ImportMIDIFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Base(importedPath) != "song.mid" {
		t.Fatalf("imported file = %q, want song.mid", filepath.Base(importedPath))
	}
	if filepath.Base(filepath.Dir(importedPath)) != importedMIDIDir {
		t.Fatalf("imported dir = %q, want %q", filepath.Base(filepath.Dir(importedPath)), importedMIDIDir)
	}
	if _, err := os.Stat(importedPath); err != nil {
		t.Fatal(err)
	}
}
