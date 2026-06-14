package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const importedMIDIDir = "imported-midi"

func IsMIDIPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mid", ".midi":
		return true
	default:
		return false
	}
}

func ImportMIDIFile(sourcePath string) (string, error) {
	if !IsMIDIPath(sourcePath) {
		return "", fmt.Errorf("only .mid and .midi files are supported")
	}

	info, err := os.Stat(sourcePath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("dropped item is a folder")
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	destinationDir := filepath.Join(wd, importedMIDIDir)
	if err := os.MkdirAll(destinationDir, 0755); err != nil {
		return "", err
	}

	destinationPath := uniqueDestinationPath(destinationDir, filepath.Base(sourcePath))
	if samePath(sourcePath, destinationPath) {
		return sourcePath, nil
	}
	if err := copyFile(sourcePath, destinationPath); err != nil {
		return "", err
	}
	return destinationPath, nil
}

func uniqueDestinationPath(dir string, name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	candidate := filepath.Join(dir, name)
	for index := 2; ; index++ {
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
		candidate = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, index, ext))
	}
}

func samePath(left string, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	return leftErr == nil && rightErr == nil && leftAbs == rightAbs
}

func copyFile(sourcePath string, destinationPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
