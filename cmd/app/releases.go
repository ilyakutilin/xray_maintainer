package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Returns the published time of a GitHub release
func GetPublishedTime(apiURL string) (time.Time, error) {
	resp, err := http.Get(apiURL)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("GitHub API request failed with status: %d", resp.StatusCode)
	}

	var release struct {
		PublishedAt string `json:"published_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return time.Time{}, err
	}

	parsedTime, err := time.Parse(time.RFC3339, release.PublishedAt)
	if err != nil {
		return time.Time{}, err
	}

	return parsedTime, nil
}
