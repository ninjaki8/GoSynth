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

	getAdbVersion(adbPath)

	// Device model
	deviceModel := "Quest_3"

	// Synthriders folder path
	remoteDir := "/sdcard/SynthRidersUC/CustomSongs/"

	// Start adb server
	startAdbServer()

	// Scan devices for Quest 3 model and return its serial number
	serial, err := getDeviceSerial(deviceModel)
	if err != nil {
		fmt.Println("[Error] ", err)
		return
	}

	// Check if synthriders custom folder exists
	synthFolderError := adbRemoteDirExists(serial, remoteDir)
	if synthFolderError != nil {
		fmt.Println("[Error] ", synthFolderError)
		return
	}

	// Get synth filenames from the device
	deviceSynthFilesMap := getDeviceSynthFiles(remoteDir, serial)
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
	if len(newBeatmaps) > 0 {
		fmt.Printf("[OK] Found %d beatmaps on device:\n", len(newBeatmaps))
	} else {
		fmt.Println("[OK] You are up to date!")
	}

	// Download new beatmaps and upload to device
	for _, bm := range newBeatmaps {
		// GET beatmap url
		filePath, err := downloadBeatmap(bm)
		if err != nil {
			fmt.Printf("[Error] Failed to download beatmap %s: %v\n", bm.Filename, err)
			continue
		}

		// Push file to device
		err = pushBeatmap(filePath, serial, remoteDir)
		if err != nil {
			fmt.Printf("[Error] Failed to upload to device %s: %v\n", bm.Filename, err)
		} else {
			fmt.Printf("[OK] Adb push success %s\n", bm.Filename)
		}
	}

}
