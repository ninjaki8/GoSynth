package main

import (
	"fmt"
	"time"
)

func main() {

	// Ensure ADB is installed
	adbPath, err := getAdbPath()
	if err != nil {
		fmt.Println("[ERROR]", err)
		return
	}

	// Show adb verson
	getAdbVersion(adbPath)

	// Start adb server
	startAdbServerError := startAdbServer(adbPath)
	if startAdbServerError != nil {
		fmt.Println("ADB:", err)
		return
	}

	// Check usb connectivity
	if !isQuestUsbConnected() {
		fmt.Println("[USB CHECK] Please connect your Quest 3")
		return
	}

	fmt.Println("[USB CHECK] Quest 3 is plugged in via USB")

	// Device model
	deviceModel := "Quest_3"

	// Synthriders folder path
	remoteDir := "/sdcard/SynthRidersUC/CustomSongs/"

	// Scan devices for Quest 3 model and return its serial number
	serial, err := getDeviceSerial(adbPath, deviceModel)
	if err != nil {
		fmt.Println("[ADB CHECK]", err)
		return
	}

	// Check for empty adb device list
	isAdbEmptyDeviceList, err := IsAdbEmptyDeviceList(adbPath)
	if err != nil {
		fmt.Println("adb command error", err)
		return
	}

	// Check if empty list but quest is plugged in by usb
	if checkMissingDevice(isAdbEmptyDeviceList) {
		fmt.Println("Quest 3 detected via USB, but not detected by ADB.\nMake sure USB debugging is enabled and accept the prompt on your headset.")
		return
	}

	// Check if synthriders custom folder exists
	synthFolderError := adbRemoteDirExists(adbPath, serial, remoteDir)
	if synthFolderError != nil {
		fmt.Println("[Error] ", synthFolderError)
		return
	}

	// Get synth filenames from the device
	deviceSynthFilesMap := getDeviceSynthFiles(adbPath, remoteDir, serial)
	count := len(deviceSynthFilesMap)
	fmt.Printf("[OK] Found: %d beatmaps on device\n", count)

	// Get first page from synthriderz.com api that contains total pages property
	firstPage, apiErr := fetchPage(1)
	if apiErr != nil {
		fmt.Println("[Error] ", apiErr)
		return
	}

	// Get all pages concurrently from the api
	start := time.Now()
	allPages := fetchAllPagesConcurrently(firstPage.PageCount)
	fmt.Printf("[OK] GET execution time: %.2f seconds \n", time.Since(start).Seconds())

	// Loop through all beatmaps and check if each filename exists on the device
	var newBeatmaps []Beatmap
	for _, page := range allPages {
		for _, beatmap := range page.Data {
			if !deviceSynthFilesMap[beatmap.Filename] {
				newBeatmaps = append(newBeatmaps, beatmap)
			}
		}
	}

	// Display new number of beatmaps to download
	newBeatmapsTotal := len(newBeatmaps)
	if newBeatmapsTotal > 0 {
		fmt.Printf("[OK] Found %d new beatmaps to download:\n", newBeatmapsTotal)
	} else {
		fmt.Println("[OK] You are up to date!")
	}

	// Download new beatmaps and upload to device
	for i, bm := range newBeatmaps {
		fmt.Printf("[Uploading %d/%d] %s - ", i+1, newBeatmapsTotal, bm.Filename)

		// GET beatmap URL
		filePath, err := downloadBeatmap(bm)
		if err != nil {
			fmt.Println("Failed to download beatmap", err)
			continue
		}

		// Push file to device
		err = pushBeatmap(adbPath, filePath, serial, remoteDir)
		if err != nil {
			fmt.Println("Failed", err)
		} else {
			fmt.Println("Success")
		}
	}
}
