package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

// Checks if a file exists
func CheckFileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// Returns the modification time of a file
func FileModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// Handles ~, relative paths, and normalizes them
func ExpandPath(path string) (string, error) {
	// Expand tilde (~) to the user's home directory
	if strings.HasPrefix(path, "~") {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		path = filepath.Join(usr.HomeDir, path[1:])
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return filepath.Clean(absPath), nil
}

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

// Downloads a file from a given URL and saves it to a specified path
func DownloadFile(url, path, filename string, executable bool) (string, error) {
	if filename == "" {
		parts := strings.Split(url, "/")
		filename = parts[len(parts)-1]
	}

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return "", err
	}

	filePath := filepath.Join(path, filename)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("file not found: %s", url)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	if executable {
		err := os.Chmod(filePath, 0755)
		if err != nil {
			return "", err
		}
	}

	return filePath, nil
}
