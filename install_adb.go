package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// getAdbPath returns the path to adb and sets it up if needed
func getAdbPath() (string, error) {

	os := getOperatingSystem()

	switch os {
	case "windows":
		return installAdbWindows()
	case "linux":
		return installAdbLinux()
	default:
		return "", fmt.Errorf("unsupported operating system: %s", os)
	}
}

func getOperatingSystem() string {
	return runtime.GOOS
}

// installAdbLinux checks if adb is installed via system package manager
func installAdbLinux() (string, error) {
	path, err := exec.LookPath("adb")
	if err != nil {
		return "", fmt.Errorf("adb not found: please install it with `sudo pacman -S android-tools` or equivalent")
	}
	return path, nil
}

// installAdbWindows downloads and extracts adb if not found, and returns its path
func installAdbWindows() (string, error) {
	adbURL := "https://dl.google.com/android/repository/platform-tools-latest-windows.zip"
	localAppData := os.Getenv("LOCALAPPDATA")
	destDir := filepath.Join(localAppData, "Android", "platform-tools")
	adbPath := filepath.Join(destDir, "adb.exe")

	if _, err := os.Stat(adbPath); err == nil {
		return adbPath, nil
	}

	fmt.Println("[INFO] ADB not found, downloading...")

	zipFile := filepath.Join(os.TempDir(), "platform-tools.zip")
	resp, err := http.Get(adbURL)
	if err != nil {
		return "", fmt.Errorf("failed to download ADB: %v", err)
	}
	defer resp.Body.Close()

	out, err := os.Create(zipFile)
	if err != nil {
		return "", fmt.Errorf("failed to create temp zip: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save zip: %v", err)
	}

	fmt.Println("[INFO] Extracting ADB...")
	if err := unzip(zipFile, filepath.Dir(destDir)); err != nil {
		return "", fmt.Errorf("failed to unzip: %v", err)
	}

	os.Remove(zipFile)
	return adbPath, nil
}

// unzip extracts a zip archive to a destination folder
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fPath := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fPath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fPath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.Create(fPath)
		if err != nil {
			return err
		}
		defer outFile.Close()

		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		_, err = io.Copy(outFile, rc)
		if err != nil {
			return err
		}
	}
	return nil
}
