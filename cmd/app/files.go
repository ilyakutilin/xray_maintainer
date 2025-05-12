package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ilyakutilin/xray_maintainer/utils"
)

type ReleaseChecker interface {
	GetLatestReleaseTag(apiURL string) (string, error)
}

type FileDownloader interface {
	Download(filePath string, url string) error
}

type File struct {
	repo           Repo
	releaseChecker ReleaseChecker
	downloader     FileDownloader
}

type GithubReleaseChecker struct{}

type GitHubFileDownloader struct{}

func NewFile(repo Repo) File {
	return File{
		repo:           repo,
		releaseChecker: GithubReleaseChecker{},
		downloader:     GitHubFileDownloader{},
	}
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
	if fileName == "" {
		return fmt.Errorf("file name cannot be empty")
	}

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
func (rc GithubReleaseChecker) GetLatestReleaseTag(apiURL string) (string, error) {
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
func (d GitHubFileDownloader) Download(filePath string, url string) error {
	dirPath := filepath.Dir(filePath)

	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return err
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("file not found: %s", url)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// Checks if the version of the file by the specified fullPath (including the filename)
// can be updated to a newer version based on the latest release version from Github.
// Updates the file if necessary.
func (app *Application) updateFile(file File) error {
	fileName := file.repo.Filename
	fileDir := app.workdir
	filePath := filepath.Join(fileDir, fileName)
	versionFilePath := filepath.Join(fileDir, "versions.json")

	app.logger.Info.Printf("Starting to update the %s file...\n", fileName)

	latestReleaseTag, err := file.releaseChecker.GetLatestReleaseTag(file.repo.ReleaseInfoURL)
	if err != nil {
		return err
	}
	app.logger.Info.Printf("The latest release tag for %s: %s\n", fileName, latestReleaseTag)

	app.logger.Info.Printf("Looking for %s file in %s...\n", fileName, fileDir)
	var backup string
	if utils.FileExists(filePath) {
		app.logger.Info.Printf("%s file found in %s\n", fileName, fileDir)
		storedTag, err := getStoredReleaseTag(fileName, versionFilePath)
		if err != nil {
			return err
		}

		if storedTag == latestReleaseTag {
			app.logger.Info.Printf("%s file is already up-to-date, no further action required\n", fileName)
			return nil
		} else {
			app.logger.Info.Printf("%s file is out-of-date, updating...\n", fileName)
			app.logger.Info.Println("Creating a backup file just in case...")
			backup, err = utils.BackupFile(filePath)
			if err != nil {
				return err
			}
			defer func() {
				err = os.Remove(backup)
				if err != nil {
					app.logger.Info.Printf("could not remove the backup file by path %s: %v", backup, err)
				}
			}()
		}
	} else {
		app.logger.Info.Printf("%s file not found in %s, starting to download...\n", fileName, fileDir)
	}

	err = file.downloader.Download(filePath, file.repo.DownloadURL)
	if err != nil {
		return err
	}
	app.logger.Info.Printf("File downloaded and is available at %s\n", filePath)

	fileIsZip, err := utils.IsZipFile(filePath)
	if err != nil {
		return err
	}
	if fileIsZip {
		app.logger.Info.Printf("The downloaded file %s is a zip, so unzipping it...\n", filepath.Base(filePath))
		zipFilePath := filePath + ".zip"
		err = os.Rename(filePath, zipFilePath)
		if err != nil {
			return err
		}

		extractedFilePath, err := utils.ExtractFileFromZip(zipFilePath, fileName)
		if err != nil {
			return err
		}
		app.logger.Info.Printf("File extracted and is available at %s\n", extractedFilePath)
		app.logger.Info.Printf("Removing the zip file %s\n", zipFilePath)
	}

	if file.repo.Executable {
		app.logger.Info.Printf("Setting executable permissions for %s\n", fileName)
		if err := utils.MakeExecutable(filePath); err != nil {
			app.logger.Error.Printf("Failed to set executable permissions for %s: %v\n", fileName, err)
			app.logger.Error.Println("Restoring the backup file...")
			err = utils.RestoreFile(backup, filePath)
			return err
		}
	}

	if !app.debug {
		app.logger.Info.Println("Checking operability of xray after the file update...")
		if err = utils.CheckOperability("xray", nil); err != nil {
			app.logger.Error.Printf("Something went wrong with the %s file update, restoring the backup file...\n", fileName)
			err = utils.RestoreFile(backup, filePath)
			return err
		}

	}

	app.logger.Info.Println("Xray is active, updating the stored release tag...")
	err = updateStoredReleaseTag(fileName, latestReleaseTag, versionFilePath)
	if err != nil {
		return err
	}
	app.logger.Info.Printf("The %s file has been updated to version %s\n", fileName, latestReleaseTag)

	return nil
}

func (app *Application) updateMultipleFiles(repos []Repo, fileCreator func(repo Repo) File) error {
	var errs utils.Errors

	for _, repo := range repos {
		file := fileCreator(repo)
		err := app.updateFile(file)
		if err != nil {
			errs.Append(err)
			app.logger.Error.Printf("Error updating %s: %v\n", repo.Name, err)
		}
	}

	if len(errs) > 0 {
		return errs
	}

	app.logger.Info.Println("All files have been updated successfully")
	return nil
}
