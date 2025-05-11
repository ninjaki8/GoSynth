package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Device struct {
	Serial string
	Model  string
}

// Beatmap represents a single beatmap entry in the API response
type Beatmap struct {
	Filename    string `json:"filename"`
	DownloadUrl string `json:"download_url"`
}

// BeatmapPage represents a single paginated response from the API
type BeatmapPage struct {
	Data      []Beatmap `json:"data"`
	Count     int       `json:"count"`
	Total     int       `json:"total"`
	Page      int       `json:"page"`
	PageCount int       `json:"pageCount"`
}

const apiEndpoint = "https://synthriderz.com/api/beatmaps"

// Reusable HTTP client with timeout
var client = &http.Client{Timeout: 10 * time.Second}

// fetchPage performs an HTTP GET request for a specific page number and returns the decoded BeatmapPage
func fetchPage(page int) BeatmapPage {
	url := fmt.Sprintf("%s?page=%d", apiEndpoint, page)

	resp, err := client.Get(url)
	if err != nil {
		log.Fatalf("Request failed for page %d: %v", page, err)
	}
	defer resp.Body.Close()

	var apiResponse BeatmapPage
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		log.Fatalf("JSON decode failed for page %d: %v", page, err)
	}

	return apiResponse
}

func fetchAllPagesConcurrently(totalPages int) []BeatmapPage {
	var wg sync.WaitGroup
	results := make(chan BeatmapPage, totalPages)

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		wg.Add(1)
		page := pageNum // capture loop variable safely

		go func() {
			defer wg.Done()
			results <- fetchPage(page)
		}()
	}

	wg.Wait()
	close(results)

	var allPages []BeatmapPage
	for page := range results {
		allPages = append(allPages, page)
	}

	return allPages
}

func isAdbServerRunning() bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:5037", 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func startAdbServer() {
	cmd := exec.Command("adb", "start-server")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to start ADB server: %v\n", err)
	}

	fmt.Printf("adb start-server output:\n%s\n", output)
}

// listConnectedDevices lists all connected devices and returns a slice of Device structs.
func listConnectedDevices() ([]Device, error) {
	cmd := exec.Command("adb", "devices", "-l")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var devices []Device
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "List of devices") || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 || fields[1] != "device" {
			continue
		}

		serial := fields[0]
		model := "(unknown)"
		for _, field := range fields {
			if strings.HasPrefix(field, "model:") {
				model = strings.TrimPrefix(field, "model:")
				break
			}
		}

		devices = append(devices, Device{Serial: serial, Model: model})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return devices, nil
}

// selectDevice lets the user choose a device from the list and returns the serial number.
func selectDevice(devices []Device) (string, error) {
	if len(devices) == 0 {
		return "", fmt.Errorf("no devices found")
	}

	// Display devices
	fmt.Println("Available devices:")
	for i, device := range devices {
		fmt.Printf("%d. Serial: %s, Model: %s\n", i+1, device.Serial, device.Model)
	}

	// Prompt user for selection
	fmt.Print("Enter the number of the device you want to select: ")
	var choice int
	_, err := fmt.Scanf("%d", &choice)
	if err != nil || choice < 1 || choice > len(devices) {
		return "", fmt.Errorf("invalid choice")
	}

	// Return the serial of the selected device
	return devices[choice-1].Serial, nil
}

// listDeviceFolder lists the contents of a specified folder on the connected device.
func listDeviceFolder(folderPath string, serial string) []string {
	cmd := exec.Command("adb", "-s", serial, "shell", "ls", folderPath)

	// Get the output of the adb command
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error listing folder %s: %v\n", folderPath, err)
		return nil
	}

	// Split the output into lines and store them in a slice
	lines := strings.Split(string(output), "\n")

	// Remove any empty lines at the end of the output
	var nonEmptyLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	// Return the slice of lines
	return nonEmptyLines
}

func downloadAndPushBeatmap(b Beatmap, serial string, remoteDir string) error {
	// Step 1: Download the file
	fullURL := "https://synthriderz.com" + b.DownloadUrl
	resp, err := http.Get(fullURL)
	if err != nil {
		return fmt.Errorf("failed to download %s: %v", b.Filename, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed for %s: status %s", b.Filename, resp.Status)
	}

	// Step 2: Save to a temp file
	tmpPath := filepath.Join(os.TempDir(), b.Filename)
	outFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	// Step 3: Push to device
	var cmd *exec.Cmd
	if serial != "" {
		cmd = exec.Command("adb", "-s", serial, "push", tmpPath, remoteDir)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("adb push failed: %v\nOutput: %s", err, string(output))
	}

	fmt.Printf("✅ Pushed %s to device at %s\n", b.Filename, remoteDir)

	// Step 4: Clean up
	err = os.Remove(tmpPath)
	if err != nil {
		fmt.Printf("⚠️ Warning: failed to delete temp file %s: %v\n", tmpPath, err)
	}

	return nil
}

func main() {
	// Start adb server
	if isAdbServerRunning() {
		fmt.Println("ADB server is already running.")
	} else {
		startAdbServer()
	}

	// List connected devices
	devices, err := listConnectedDevices()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Let the user select a device
	serial, err := selectDevice(devices)
	if err != nil {
		fmt.Printf("Error selecting device: %v\n", err)
		return
	}

	// Print the selected device's serial
	fmt.Printf("You selected device with Serial: %s\n", serial)

	// Get synth filenames from the device
	files := listDeviceFolder("/sdcard/SynthRidersUC/CustomSongs/", serial)

	count := len(files)
	fmt.Printf("The number of items in the slice is: %d\n", count)

	// Fetch beatmaps from synthriderz.com api
	firstPage := fetchPage(1)
	start := time.Now()

	allPages := fetchAllPagesConcurrently(firstPage.PageCount)

	fmt.Printf("Execution time: %v\n", time.Since(start))
	for _, page := range allPages {
		fmt.Printf("Processed page %d with %d beatmaps\n", page.Page, len(page.Data))
	}

	// Step 1: Convert device files to a map for fast lookup
	deviceFilesMap := make(map[string]bool)
	for _, file := range files {
		deviceFilesMap[file] = true
	}

	// Step 2: Loop through all beatmaps and check if each filename exists on the device
	var missing []Beatmap

	for _, page := range allPages {
		for _, beatmap := range page.Data {
			if !deviceFilesMap[beatmap.Filename] {
				missing = append(missing, beatmap)
			}
		}
	}

	// Step 3: Report missing beatmaps
	if len(missing) > 0 {
		fmt.Printf("\nMissing %d beatmaps on device:\n", len(missing))
		for _, bm := range missing {
			fmt.Printf("Filename: %s\nDownload URL: %s\n\n", bm.Filename, bm.DownloadUrl)
		}
	} else {
		fmt.Println("\nAll beatmaps are present on the device.")
	}

	// Download missing beatmaps and upload to device
	remoteDir := "/sdcard/SynthRidersUC/CustomSongs/"

	for _, bm := range missing {
		err := downloadAndPushBeatmap(bm, serial, remoteDir)
		if err != nil {
			fmt.Printf("❌ Error processing %s: %v\n", bm.Filename, err)
		}
	}

}
