package main

import (
	"context"
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
		return "", fmt.Errorf("failed to read the versions file: %w", err)
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
// from the URL.
func (d GitHubFileDownloader) Download(filePath string, url string) error {
	dirPath := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)

	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to get the response from the url %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("file not found at %s", url)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to download file %s: HTTP %d",
			fileName, resp.StatusCode)
	}

	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create the file %s at path %s: %w",
			fileName, filePath, err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy the contents of the new file to the file "+
			"at path %s: %w", filePath, err)
	}

	return nil
}

// Checks if the version of the file by the specified fullPath (including the filename)
// can be updated to a newer version based on the latest release version from Github.
// Updates the file if necessary.
func (app *Application) updateFile(ctx context.Context, file File) error {
	fileName := file.repo.Filename
	fileDir := app.workdir
	filePath := filepath.Join(fileDir, fileName)
	versionFilePath := filepath.Join(fileDir, "versions.json")

	app.logger.Info.Printf("Starting to update the %s file...\n", fileName)

	latestReleaseTag, err := file.releaseChecker.GetLatestReleaseTag(file.repo.ReleaseInfoURL)
	if err != nil {
		app.warn(fmt.Sprintf("Failed to get the latest release tag for %s "+
			"from github: %v. The file has not been updated.", fileName, err))
		return nil
	}
	app.logger.Info.Printf("The latest release tag for %s: %s\n",
		fileName, latestReleaseTag)

	app.logger.Info.Printf("Looking for %s file in %s...\n", fileName, fileDir)
	var backup string
	if utils.FileExists(filePath) {
		app.logger.Info.Printf("%s file found in %s\n", fileName, fileDir)
		storedTag, err := getStoredReleaseTag(fileName, versionFilePath)
		if err != nil {
			app.warn(fmt.Sprintf("Error while getting the local stored release tag "+
				"for %s: %v. The file has not been updated.", fileName, err))
			return nil
		}

		if storedTag == latestReleaseTag {
			app.logger.Info.Printf("%s file is already up-to-date (%s), "+
				"no further action required\n", fileName, storedTag)
			return nil
		} else {
			app.logger.Info.Printf("%s file is out-of-date: local version is %s, "+
				"remote version is %s, updating...\n",
				fileName, storedTag, latestReleaseTag)
			app.logger.Info.Println("Creating a backup file just in case...")
			backup, err = utils.BackupFile(filePath)
			if err != nil {
				app.warn(fmt.Sprintf("Failed to back up the file %s: %v. "+
					"The file has not been updated.", fileName, err))
				return nil
			}
			defer func() {
				err = os.Remove(backup)
				if err != nil {
					app.warn(fmt.Sprintf("could not remove the backup file by path "+
						"%s: %v", backup, err))
				}
			}()
		}
	} else {
		app.logger.Info.Printf("%s file not found in %s, starting to download...\n",
			fileName, fileDir)
	}

	err = file.downloader.Download(filePath, file.repo.DownloadURL)
	if err != nil {
		app.warn(fmt.Sprintf("Failed to download the file %s: %v. "+
			"The file has not been updated.", fileName, err))
		return nil
	}
	app.logger.Info.Printf("File %s has been downloaded and is available at %s\n",
		fileName, filePath)

	fileIsZip, err := utils.IsZipFile(filePath)
	if err != nil {
		app.warn(fmt.Sprintf("Failed to check whether the file %s is a zip file"+
			"or not: %v. The file has not been updated.", fileName, err))
		app.logger.Info.Printf("Restoring file %s from backup...", fileName)
		if err := utils.RestoreFile(backup, filePath); err != nil {
			return fmt.Errorf("failed to restore file %s from backup: %w",
				fileName, err)
		}
		return nil
	}
	if fileIsZip {
		app.logger.Info.Printf("The downloaded file %s is a zip, so unzipping it...\n",
			filepath.Base(filePath))
		zipFilePath := filePath + ".zip"
		err = os.Rename(filePath, zipFilePath)
		if err != nil {
			app.warn(fmt.Sprintf("Failed to rename file %s to %s.zip: %v. "+
				"The file has not been updated.", fileName, fileName, err))
			return nil
		}

		extractedFilePath, err := utils.ExtractFileFromZip(zipFilePath, fileName)
		if err != nil {
			app.warn(fmt.Sprintf("Failed to extract the necessary file %s "+
				"from zip: %v. The file has not been updated.", fileName, err))
			return nil
		}
		app.logger.Info.Printf("File %s has been extracted from zip "+
			"and is available at %s\n", fileName, extractedFilePath)
	}

	// TODO: Test executability
	if file.repo.Executable {
		app.logger.Info.Printf("Setting executable permissions for %s\n", fileName)
		if err := utils.MakeExecutable(filePath); err != nil {
			app.warn(fmt.Sprintf("Failed to set executable permissions for %s: %v. "+
				"The file has not been updated. Restoring the file from backup...",
				fileName, err))
			if err := utils.RestoreFile(backup, filePath); err != nil {
				return fmt.Errorf("failed to restore file %s from backup: %w",
					fileName, err)
			}
		}
	}

	if !app.debug {
		app.logger.Info.Printf("Checking operability of %s after the file update...\n",
			app.xrayServiceName)
		if err = utils.CheckOperability(ctx, app.xrayServiceName, nil); err != nil {
			app.warn(fmt.Sprintf("Service %s operability check failed after the "+
				"file %s has been updated, while it was operational prior to the "+
				"update. All the changes to this file will now be reverted, "+
				"and the original file will be restored from backup. The file has not "+
				"been updated.", app.xrayServiceName, fileName))
			if err := utils.RestoreFile(backup, filePath); err != nil {
				return fmt.Errorf("failed to restore file %s from backup: %w",
					fileName, err)
			}
		}
		app.logger.Info.Printf("%s is active, updating the stored release tag...\n",
			app.xrayServiceName)
	} else {
		app.logger.Info.Println("Updating the stored release tag...")
	}

	err = updateStoredReleaseTag(fileName, latestReleaseTag, versionFilePath)
	if err != nil {
		app.warn(fmt.Sprintf("Failed to update the locally stored release tag "+
			"of %s. This will lead to the need of a repeated update of %s the next "+
			"time this app runs, and will likely fail again until the reason is "+
			"investigated. However, the %s file update was NOT interrupted.",
			fileName, fileName, fileName))
	}
	app.logger.Info.Printf("The %s file has been successfully updated to version %s\n",
		fileName, latestReleaseTag)

	return nil
}

func (app *Application) updateMultipleFiles(ctx context.Context, repos []Repo, fileCreator func(repo Repo) File) error {
	var errs utils.Errors

	for _, repo := range repos {
		file := fileCreator(repo)
		err := app.updateFile(ctx, file)
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
