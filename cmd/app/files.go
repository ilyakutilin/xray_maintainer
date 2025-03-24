package main

import (
	"archive/zip"
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
// from the URL. The permissions are set based on umask. Returns the full path
// to the downloaded file.
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

// extractFileFromZip checks if a zip archive by the provided zipFilePath contains
// a file with the fileName. If there is no such file, returns an error.
// If the file is found, it is extracted to the same directory as the zip archive,
// and the full path to the extracted file is returned.
func extractFileFromZip(zipFilePath string, fileName string) (string, error) {
	// Open the zip archive
	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// Loop through the files in the zip archive to find the file by the fileName
	var foundFile *zip.File
	for _, f := range r.File {
		if f.Name == fileName {
			foundFile = f
			break
		}
	}

	zipFileName := filepath.Base(zipFilePath)
	if foundFile == nil {
		return "", fmt.Errorf("file %s not found in zip archive %s", fileName, zipFileName)
	}

	// Extract the file to the same directory as the zip archive
	destDir := filepath.Dir(zipFilePath)
	destPath := filepath.Join(destDir, fileName)
	outFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file %s: %w", fileName, err)
	}
	defer outFile.Close()

	zipFileReader, err := foundFile.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file %s inside zip %s: %w", fileName, zipFileName, err)
	}
	defer zipFileReader.Close()

	_, err = io.Copy(outFile, zipFileReader)
	if err != nil {
		return "", fmt.Errorf("failed to extract file %s from zip %s: %w", fileName, zipFileName, err)
	}

	// Delete the ZIP archive
	if err := os.Remove(zipFilePath); err != nil {
		return "", fmt.Errorf("failed to delete zip file %s: %w", zipFileName, err)
	}

	return destPath, nil
}

func backupFile(filePath string) (string, error) {
	backupFilePath := filePath + ".backup"
	if err := os.Rename(filePath, backupFilePath); err != nil {
		return "", err
	}
	return backupFilePath, nil
}

func restoreFile(filePath, backupFilePath string) error {
	if err := os.Rename(backupFilePath, filePath); err != nil {
		return err
	}
	return nil
}

// Checks if the version of the file by the specified fullPath (including the filename)
// can be updated to a newer version based on the latest release version from Github.
// Updates the file if necessary.
func updateFile(file File) error {
	logger := GetLogger()

	fileName := filepath.Base(file.filePath)
	fileDir := filepath.Dir(file.filePath)
	versionFilePath := filepath.Join(fileDir, "versions.json")

	logger.Info.Printf("Starting to update the %s file...\n", fileName)

	latestReleaseTag, err := getLatestReleaseTag(file.releaseURL)
	if err != nil {
		return err
	}
	logger.Info.Printf("The latest release tag for %s: %s\n", fileName, latestReleaseTag)

	logger.Info.Printf("Looking for %s file in %s...\n", fileName, fileDir)
	var backup string
	if fileExists(file.filePath) {
		logger.Info.Printf("%s file found in %s\n", fileName, fileDir)
		storedTag, err := getStoredReleaseTag(fileName, versionFilePath)
		if err != nil {
			return err
		}

		if storedTag == latestReleaseTag {
			logger.Info.Printf("%s file is already up-to-date, no further action required\n", fileName)
			return nil
		} else {
			logger.Info.Printf("%s file is out-of-date, updating...\n", fileName)
			logger.Info.Println("Creating a backup file just in case...")
			backup, err = backupFile(file.filePath)
			if err != nil {
				return err
			}
		}
	} else {
		logger.Info.Printf("%s file not found in %s, starting to download...\n", fileName, fileDir)
	}

	downloadedFilePath, err := downloadFile(file.downloadURL, fileDir, "")
	if err != nil {
		return err
	}
	logger.Info.Printf("File downloaded: %s\n", downloadedFilePath)

	if filepath.Ext(downloadedFilePath) == ".zip" {
		logger.Info.Printf("The downloaded file %s is a zip, so unzipping it...\n", filepath.Base(downloadedFilePath))

		extractedFilePath, err := extractFileFromZip(downloadedFilePath, fileName)
		if err != nil {
			return err
		}
		logger.Info.Printf("File extracted and is available at %s\n", extractedFilePath)
		logger.Info.Printf("Removing the zip file %s\n", downloadedFilePath)
		if err = os.Remove(downloadedFilePath); err != nil {
			return err
		}
	} else {
		logger.Info.Printf("Renaming %s to %s\n", filepath.Base(downloadedFilePath), fileName)
		err = os.Rename(downloadedFilePath, file.filePath)
		if err != nil {
			return err
		}
	}

	logger.Info.Println("Checking operability of xray after the file update...")
	if err = checkOperability("xray"); err != nil {
		logger.Error.Printf("Something went wrong with the %s file update, restoring the backup file...\n", fileName)
		err = restoreFile(file.filePath, backup)
		return err
	}

	logger.Info.Println("Xray is active, updating the stored release tag...")
	err = updateStoredReleaseTag(fileName, latestReleaseTag, versionFilePath)
	if err != nil {
		return err
	}
	logger.Info.Printf("The %s file has been updated to version %s\n", fileName, latestReleaseTag)

	return nil
}
