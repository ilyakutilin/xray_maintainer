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
)

// Checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// Handles ~, relative paths, and normalizes them
func expandPath(path string) (string, error) {
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

func getStoredReleaseTag(fileName string, versionFilePath string) (string, error) {
	data, err := os.ReadFile(versionFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No stored version yet
		}
		return "", err
	}

	var versions map[string]string
	if err := json.Unmarshal(data, &versions); err != nil {
		return "", err
	}

	version, exists := versions[fileName]
	if !exists {
		return "", nil // File version not found
	}

	return version, nil
}

func updateStoredReleaseTag(fileName, newVersion, versionFilePath string) error {
	data, err := os.ReadFile(versionFilePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var versions map[string]string
	if len(data) > 0 {
		if err := json.Unmarshal(data, &versions); err != nil {
			return err
		}
	} else {
		versions = make(map[string]string)
	}

	versions[fileName] = newVersion

	newData, err := json.MarshalIndent(versions, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(versionFilePath, newData, 0644)
}

// Returns the tag name of the latest GitHub release
func getLatestReleaseTag(apiURL string) (string, error) {
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API request failed with status: %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

// Downloads a file from a given URL and saves it to a specified directory path.
// If a filename is provided, it will be used, otherwise the filename will be extracted
// from the URL. The permissions are set based on umask.
func downloadFile(url, dirPath, filename string) (string, error) {
	if filename == "" {
		parts := strings.Split(url, "/")
		filename = parts[len(parts)-1]
	}

	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return "", err
	}

	filePath := filepath.Join(dirPath, filename)

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

	return filePath, nil
}

// Checks if the version of the file by the specified fullPath (including the filename)
// can be updated to a newer version based on the latest release version from Github.
// Updates the file if necessary.
func updateFile(file File) error {
	fileName := filepath.Base(file.filePath)
	versionFilePath := filepath.Join(filepath.Dir(file.filePath), "versions.json")

	latestReleaseTag, err := getLatestReleaseTag(file.releaseURL)
	if err != nil {
		return err
	}

	if fileExists(file.filePath) {
		storedTag, err := getStoredReleaseTag(fileName, versionFilePath)
		if err != nil {
			return err
		}

		if storedTag == latestReleaseTag {
			return nil
		}
	}

	_, err = downloadFile(file.downloadURL, filepath.Dir(file.filePath), "")
	if err != nil {
		return err
	}

	return nil
}
