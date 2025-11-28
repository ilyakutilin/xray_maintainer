package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ilyakutilin/xray_maintainer/utils"
)

type ReleaseChecker interface {
	GetLatestReleaseTag(apiURL string) (string, error)
	GetStoredReleaseTag(fileName string, versionFilePath string) (string, error)
	UpdateStoredReleaseTag(fileName, newVersion, versionFilePath string) error
}

type FileDownloader interface {
	Download(filePath string, url string) error
	Backup(filePath string) (string, error)
	Restore(backup, filePath string) error
	ProcessZip(filePath, backup string) (bool, error)
	MakeExecutable(filePath string) error
}

type File struct {
	repo           Repo
	releaseChecker ReleaseChecker
	downloader     FileDownloader
}

type GithubReleaseChecker struct{}
type DryRunReleaseChecker struct{}

type GitHubFileDownloader struct{}
type DryRunFileDownloader struct{}

func NewFile(repo Repo) File {
	return File{
		repo:           repo,
		releaseChecker: GithubReleaseChecker{},
		downloader:     GitHubFileDownloader{},
	}
}

func NewFileDryRun(repo Repo) File {
	return File{
		repo:           repo,
		releaseChecker: DryRunReleaseChecker{},
		downloader:     DryRunFileDownloader{},
	}
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

func updateStoredReleaseTag(
	fileName, newVersion, versionFilePath string, write bool,
) error {
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

	if !write {
		return nil
	}
	return os.WriteFile(versionFilePath, newData, 0644)
}

// Returns the tag name of the latest GitHub release
func (rc GithubReleaseChecker) GetLatestReleaseTag(apiURL string) (string, error) {
	return getLatestReleaseTag(apiURL)
}

// Returns the tag name of the latest GitHub release
func (rc DryRunReleaseChecker) GetLatestReleaseTag(apiURL string) (string, error) {
	return getLatestReleaseTag(apiURL)
}

func (rc GithubReleaseChecker) GetStoredReleaseTag(
	fileName string, versionFilePath string,
) (string, error) {
	return getStoredReleaseTag(fileName, versionFilePath)
}

func (rc DryRunReleaseChecker) GetStoredReleaseTag(
	fileName string, versionFilePath string,
) (string, error) {
	return getStoredReleaseTag(fileName, versionFilePath)
}

func (rc GithubReleaseChecker) UpdateStoredReleaseTag(
	fileName, newVersion, versionFilePath string,
) error {
	return updateStoredReleaseTag(fileName, newVersion, versionFilePath, true)
}

func (rc DryRunReleaseChecker) UpdateStoredReleaseTag(
	fileName, newVersion, versionFilePath string,
) error {
	return updateStoredReleaseTag(fileName, newVersion, versionFilePath, false)
}

// Downloads a file from a given URL and saves it to a specified directory path.
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

// Simulates downloading a file from a given URL.
// If a filename is provided, it will be used, otherwise the filename will be extracted
// from the URL.
func (d DryRunFileDownloader) Download(filePath string, url string) error {
	fileName := filepath.Base(filePath)

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

	return nil
}

func (d GitHubFileDownloader) Backup(filePath string) (string, error) {
	return utils.BackupFile(filePath)
}

func (d DryRunFileDownloader) Backup(filePath string) (string, error) {
	return filePath + "_backup", nil
}

func (d GitHubFileDownloader) Restore(backup, filePath string) error {
	return utils.RestoreFile(backup, filePath)
}

func (d DryRunFileDownloader) Restore(backup, filePath string) error {
	return nil
}

func (d GitHubFileDownloader) ProcessZip(filePath, backup string) (bool, error) {
	fileName := filepath.Base(filePath)
	fileIsZip, err := utils.IsZipFile(filePath)
	if err != nil {
		mainErrTxt := fmt.Sprintf("Failed to check whether the file %s is a zip file "+
			"or not: %v. The file has not been updated.", fileName, err)
		if err := d.Restore(backup, filePath); err != nil {
			return false, errors.New(mainErrTxt + " Additionally, could not " +
				"restore the file from backup.")
		}
		return false, errors.New(mainErrTxt)
	}
	if fileIsZip {
		zipFilePath := filePath + ".zip"
		err = os.Rename(filePath, zipFilePath)
		if err != nil {
			return true, fmt.Errorf("failed to rename file %s to %s.zip: %v. "+
				"The file has not been updated", fileName, fileName, err)
		}
		defer os.Remove(zipFilePath)

		extractedFilePath, err := utils.ExtractFileFromZip(zipFilePath, fileName)
		if extractedFilePath != filePath {
			panic(fmt.Sprintf("Original file path is %s, file path extracted from "+
				"zip is %s, while they should be identical.",
				filePath, extractedFilePath))
		}
		if err != nil {
			return true, fmt.Errorf("failed to extract the necessary file %s "+
				"from zip: %v. The file has not been updated", fileName, err)
		}
		return true, nil
	}

	return false, nil
}

func (d DryRunFileDownloader) ProcessZip(filePath, backup string) (bool, error) {
	fileName := filepath.Base(filePath)
	fileIsZip, err := utils.IsZipFile(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to check whether the file %s is a zip file "+
			"or not: %v. The file has not been updated", fileName, err)
	}
	if fileIsZip {
		return true, nil
	}

	return false, nil
}

func (d GitHubFileDownloader) MakeExecutable(filePath string) error {
	return utils.MakeExecutable(filePath)
}

func (d DryRunFileDownloader) MakeExecutable(filePath string) error {
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
	if latestReleaseTag == "" {
		app.logger.Info.Printf("The release tag for %s is not recorded in the "+
			"versions file or there is no versions file at all\n", fileName)
	} else {
		app.logger.Info.Printf("The latest release tag for %s: %s\n",
			fileName, latestReleaseTag)
	}

	app.logger.Info.Printf("Looking for %s file in %s...\n", fileName, fileDir)
	var backup string
	if utils.FileExists(filePath) {
		app.logger.Info.Printf("%s file found in %s\n", fileName, fileDir)
		storedTag, err := file.releaseChecker.GetStoredReleaseTag(
			fileName, versionFilePath,
		)
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
			backup, err = file.downloader.Backup(filePath)
			if err != nil {
				app.warn(fmt.Sprintf("Failed to back up the file %s: %v. "+
					"The file has not been updated.", fileName, err))
				return nil
			}
			if !app.dryRun {
				defer func() {
					err = os.Remove(backup)
					if err != nil {
						app.warn(fmt.Sprintf("could not remove the backup file by path "+
							"%s: %v", backup, err))
					}
				}()
			}
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

	fileIsZip, err := file.downloader.ProcessZip(filePath, backup)
	switch {
	case fileIsZip && err == nil:
		app.logger.Info.Printf("File %s is zip, and it was successfully extracted.",
			fileName)
	case fileIsZip && err != nil:
		app.warn(fmt.Sprintf("File %s is zip, and there was a problem with its "+
			"extracting: %v", fileName, err))
		return nil
	case !fileIsZip && err != nil:
		app.warn(fmt.Sprintf("failed to determine whether file %s is zip or not: "+
			"%v. To be on the safe side, it will not be updated.", fileName, err))
		return nil
	}

	// TODO: Test executability
	if file.repo.Executable {
		app.logger.Info.Printf("Setting executable permissions for %s\n", fileName)
		if err := file.downloader.MakeExecutable(filePath); err != nil {
			app.warn(fmt.Sprintf("Failed to set executable permissions for %s: %v. "+
				"The file has not been updated. Restoring the file from backup...",
				fileName, err))
			if err := file.downloader.Restore(backup, filePath); err != nil {
				return fmt.Errorf("failed to restore file %s from backup: %w",
					fileName, err)
			}
		}
	}

	// TODO: Instead of checking operability after updating each file, do this only once
	app.logger.Info.Printf("Checking operability of %s after the file update...\n",
		app.xrayServiceName)
	if err = utils.CheckOperability(ctx, app.xrayServiceName, nil); err != nil {
		app.warn(fmt.Sprintf("Service %s operability check failed after the "+
			"file %s has been updated, while it was operational prior to the "+
			"update. All the changes to this file will now be reverted, "+
			"and the original file will be restored from backup. The file has not "+
			"been updated.", app.xrayServiceName, fileName))
		if err := file.downloader.Restore(backup, filePath); err != nil {
			return fmt.Errorf("failed to restore file %s from backup: %w",
				fileName, err)
		}
	}
	app.logger.Info.Printf("%s is active, updating the stored release tag...\n",
		app.xrayServiceName)

	err = file.releaseChecker.UpdateStoredReleaseTag(
		fileName, latestReleaseTag, versionFilePath,
	)
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

func (app *Application) updateMultipleFiles(
	ctx context.Context, repos []Repo, dryRun bool,
) error {
	var errs utils.Errors

	var fileCreator func(repo Repo) File
	if dryRun {
		fileCreator = NewFileDryRun
	} else {
		fileCreator = NewFile
	}

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
