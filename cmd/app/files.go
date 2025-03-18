package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Returns the modification time of a file
func FileModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// Downloads a file from a given URL and saves it to a specified path
func DownloadFile(url, path, filename string, executable bool) error {
	if filename == "" {
		parts := strings.Split(url, "/")
		filename = parts[len(parts)-1]
	}

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	filePath := filepath.Join(path, filename)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	if executable {
		return os.Chmod(filePath, 0755)
	}

	return nil
}
