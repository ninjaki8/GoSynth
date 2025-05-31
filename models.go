package main

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
