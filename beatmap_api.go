package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// fetchPage performs an HTTP GET request for a specific page number and returns the decoded BeatmapPage
func fetchPage(page int) (BeatmapPage, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("%s?page=%d", API_ENDPOINT, page)

	resp, err := client.Get(url)
	if err != nil {
		return BeatmapPage{}, fmt.Errorf("request failed for page %d: %w", page, err)
	}
	defer resp.Body.Close()

	var apiResponse BeatmapPage
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return BeatmapPage{}, fmt.Errorf("JSON decode failed for page %d: %w", page, err)
	}

	return apiResponse, nil
}

func fetchAllPagesConcurrently(totalPages int) []BeatmapPage {
	var wg sync.WaitGroup
	var printLock sync.Mutex
	results := make(chan BeatmapPage, totalPages)

	fmt.Print("[OK] GET api pages: ")

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		wg.Add(1)
		page := pageNum

		go func(p int) {
			defer wg.Done()

			pageData, err := fetchPage(p)
			printLock.Lock()
			defer printLock.Unlock()
			if err != nil {
				fmt.Printf("x") // mark failures
				return
			}

			fmt.Print("*")
			results <- pageData
		}(page)
	}

	wg.Wait()
	close(results)

	fmt.Println() // new line after last printed page number

	var allPages []BeatmapPage
	for page := range results {
		allPages = append(allPages, page)
	}

	return allPages
}

func downloadBeatmap(b Beatmap) (string, error) {
	// Step 1: Download file
	url := BASE_URL + b.DownloadUrl
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download %s: %v", b.Filename, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed for %s: status %s", b.Filename, resp.Status)
	}

	// Step 2: Save to a temp file
	tmpPath := filepath.Join(os.TempDir(), b.Filename)
	outFile, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	return tmpPath, nil
}
