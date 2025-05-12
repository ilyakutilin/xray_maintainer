package utils

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// EnsureDir checks if a directory exists at the specified path,
// and creates it if it doesn't exist.
// Returns error if creation fails or if path exists but is not a directory.
func EnsureDir(dirPath string) error {
	// Check if the path exists
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		// Directory doesn't exist, create it
		err = os.MkdirAll(dirPath, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
		return nil
	}
	if err != nil {
		// Some other error occurred (like permission issues)
		return fmt.Errorf("error checking directory: %v", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path exists but is not a directory: %s", dirPath)
	}
	// Directory already exists
	return nil
}

// ExpandPath handles ~, relative paths, and normalizes them
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

// MakeExecutable makes a file executable
func MakeExecutable(filePath string) error {
	return os.Chmod(filePath, 0755)
}

// IsZipFile checks if a file is a zip archive
func IsZipFile(filePath string) (bool, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Get file info to check size
	fileInfo, err := file.Stat()
	if err != nil {
		return false, err
	}

	// Empty file cannot be a zip
	if fileInfo.Size() == 0 {
		return false, nil
	}

	// Read the first 4 bytes to check the signature
	buf := make([]byte, 4)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}

	// If we couldn't read at least 2 bytes, it's not a zip
	if n < 2 {
		return false, nil
	}

	// ZIP file signature is "PK" (0x50 0x4B)
	return bytes.Equal(buf[:2], []byte{0x50, 0x4B}), nil
}

// ExtractFileFromZip checks if a zip archive by the provided zipFilePath contains
// a file with the fileName. If there is no such file, returns an error.
// If the file is found, it is extracted to the same directory as the zip archive,
// and the full path to the extracted file is returned.
func ExtractFileFromZip(zipFilePath string, fileName string) (string, error) {
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

// BackupFile renames the specified file to create a backup with a ".backup" extension.
// It takes the original file path as input and returns the new backup file path
// or an error if the renaming operation fails.
func BackupFile(filePath string) (string, error) {
	backupFilePath := filePath + ".backup"
	if err := os.Rename(filePath, backupFilePath); err != nil {
		return "", err
	}
	return backupFilePath, nil
}

// RestoreFile copies the contents of the source file specified by `src`
// to the destination file specified by `dst`. If the destination file
// does not exist, it will be created. The function ensures that all data
// is written to disk before returning.
func RestoreFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Ensure data is written to disk
	return dstFile.Sync()
}
