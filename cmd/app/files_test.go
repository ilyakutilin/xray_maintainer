package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TODO: Unify cleanups (e.g. RemoveAll vs Remove)
// TODO: Unify setups (e.g. create temp subdirs within the temp dir for diff tests)

func TestFileExists(t *testing.T) {
	tempFile, cleanup := createTempFile(t)
	t.Cleanup(cleanup)

	// Test when file exists
	if !fileExists(tempFile) {
		t.Errorf("fileExists shall return true for existing file")
	}

	// Test when file does not exist
	nonExistentPath := tempFile + "_nonexistent"
	if fileExists(nonExistentPath) {
		t.Errorf("fileExists shall return false for non-existing file")
	}
}

func TestExpandPath(t *testing.T) {
	usr, err := user.Current()
	if err != nil {
		t.Fatalf("Failed to get current user: %v", err)
	}

	workingDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	var tests = []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Test relative path",
			input:    "./testfile",
			expected: filepath.Join(workingDir, "testfile"),
		},
		{
			name:     "Test absolute path",
			input:    "/testfile",
			expected: "/testfile",
		},
		{
			name:     "Test tilde expansion",
			input:    "~/testfile",
			expected: filepath.Join(usr.HomeDir, "testfile"),
		},
		{
			name:     "Test redundant elements",
			input:    "./test/../testfile",
			expected: filepath.Join(workingDir, "testfile"),
		},
		{
			name:     "Test empty path",
			input:    "",
			expected: workingDir,
		},
		{
			name:     "Test current directory",
			input:    ".",
			expected: workingDir,
		},
		{
			name:     "Test parent directory",
			input:    "..",
			expected: filepath.Dir(workingDir),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := expandPath(test.input)
			if err != nil {
				t.Errorf("expandPath returned error: %v", err)
			}
			assertCorrectString(t, test.expected, got)
		})
	}
}

func TestGetStoredReleaseTag(t *testing.T) {
	// Versions file does not exist
	t.Run("Versions file does not exist", func(t *testing.T) {
		tag, err := getStoredReleaseTag("testfile", "doesnotexist.json")
		assertCorrectString(t, "", tag)
		assertNoError(t, err)
	})

	versionsFile, cleanup := createTempFile(t)

	var tests = []struct {
		name           string
		fileContents   []byte
		expectedReturn string
		// errors.As does not work with struct / loop approach, so using reflect
		expectedError reflect.Type
	}{
		{
			name:           "Versions file exists and has correct structure",
			fileContents:   []byte(`{"testfile": "1.2.3"}`),
			expectedReturn: "1.2.3",
			expectedError:  nil,
		},
		{
			name:           "Versions file exists but is empty",
			fileContents:   []byte("{}"),
			expectedReturn: "",
			expectedError:  nil,
		},
		{
			name:           "Versions file exists but is malformed",
			fileContents:   []byte(`{"testfile": `),
			expectedReturn: "",
			expectedError:  reflect.TypeOf((*json.SyntaxError)(nil)),
		},
		{
			name:           "Versions file has wrong type of value",
			fileContents:   []byte(`{"testfile": 5}`),
			expectedReturn: "",
			expectedError:  reflect.TypeOf((*json.UnmarshalTypeError)(nil)),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Cleanup(cleanup)
			err := os.WriteFile(versionsFile, test.fileContents, os.ModePerm)
			if err != nil {
				t.Fatalf("Failed to write test data: %v", err)
			}
			tag, err := getStoredReleaseTag("testfile", versionsFile)
			assertCorrectString(t, test.expectedReturn, tag)
			if test.expectedError == nil {
				assertNoError(t, err)
			} else {
				actualErrType := reflect.TypeOf(err)
				if actualErrType != test.expectedError {
					t.Errorf("Expected an error of type %v but got %v", test.expectedError, actualErrType)
				}
			}
		})
	}

}

func TestUpdateStoredReleaseTag(t *testing.T) {
	t.Run("Versions file gets created if it does not exist", func(t *testing.T) {
		err := updateStoredReleaseTag("testfile", "1.2.3", filepath.Join(os.TempDir(), "doesnotexist.json"))
		assertNoError(t, err)
		data, err := os.ReadFile(filepath.Join(os.TempDir(), "doesnotexist.json"))
		assertNoError(t, err)
		assertCorrectString(t, "{\n  \"testfile\": \"1.2.3\"\n}", string(data))
	})

	versionsFile, cleanup := createTempFile(t)

	var tests = []struct {
		name            string
		fileName        string
		existingContent []byte
		expectedMap     map[string]string
		errorExpected   bool
	}{
		{
			name:            "Change the existing tag",
			fileName:        "testfile",
			existingContent: []byte(`{"testfile": "1.2.3"}`),
			expectedMap:     map[string]string{"testfile": "1.2.4"},
			errorExpected:   false,
		},
		{
			name:            "Add a new tag and preserve existing ones",
			fileName:        "new_testfile",
			existingContent: []byte(`{"testfile": "1.2.3"}`),
			expectedMap:     map[string]string{"testfile": "1.2.3", "new_testfile": "1.2.4"},
			errorExpected:   false,
		},
		{
			name:            "Add a new tag to the empty JSON",
			fileName:        "testfile",
			existingContent: []byte(`{}`),
			expectedMap:     map[string]string{"testfile": "1.2.4"},
			errorExpected:   false,
		},
		{
			name:            "Updating malformed JSON fails",
			fileName:        "does_not_matter",
			existingContent: []byte(`{"something": }`),
			expectedMap:     map[string]string{},
			errorExpected:   true,
		},
		{
			name:            "Empty fileName",
			fileName:        "",
			existingContent: []byte(`{"testfile": "1.2.3"}`),
			expectedMap:     map[string]string{},
			errorExpected:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Cleanup(cleanup)
			err := os.WriteFile(versionsFile, test.existingContent, os.ModePerm)
			if err != nil {
				t.Fatalf("Failed to write test data: %v", err)
			}
			err = updateStoredReleaseTag(test.fileName, "1.2.4", versionsFile)
			if test.errorExpected {
				assertError(t, err)
				return
			} else {
				assertNoError(t, err)
			}
			data, err := os.ReadFile(versionsFile)
			assertNoError(t, err)
			var actualMap map[string]string
			err = json.Unmarshal(data, &actualMap)
			assertNoError(t, err)
			if !reflect.DeepEqual(actualMap, test.expectedMap) {
				t.Errorf("Expected map %v but got %v", test.expectedMap, actualMap)
			}
		})
	}
}

func TestGetLatestReleaseTag(t *testing.T) {
	releaseChecker := GithubReleaseChecker{}

	tests := []struct {
		name           string
		responseStatus int
		responseBody   any
		want           string
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "successful response",
			responseStatus: http.StatusOK,
			responseBody:   map[string]string{"tag_name": "v1.2.3"},
			want:           "v1.2.3",
			wantErr:        false,
		},
		{
			name:           "non-200 status code",
			responseStatus: http.StatusNotFound,
			responseBody:   map[string]string{"message": "Not Found"},
			wantErr:        true,
			errMsg:         "GitHub API request failed with status: 404",
		},
		{
			name:           "malformed JSON",
			responseStatus: http.StatusOK,
			responseBody:   "not json",
			wantErr:        true,
			errMsg:         "invalid character",
		},
		{
			name:           "empty tag name",
			responseStatus: http.StatusOK,
			responseBody:   map[string]string{"tag_name": ""},
			want:           "",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseStatus)
				switch body := tt.responseBody.(type) {
				case string:
					w.Write([]byte(body))
				default:
					json.NewEncoder(w).Encode(body)
				}
			}))
			t.Cleanup(server.Close)

			got, err := releaseChecker.GetLatestReleaseTag(server.URL)

			if (err != nil) != tt.wantErr {
				t.Errorf("getLatestReleaseTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
						t.Errorf("getLatestReleaseTag() error = %v, want error containing %q", err.Error(), tt.errMsg)
					}
				}
				return
			}

			assertCorrectString(t, tt.want, got)
		})
	}
}

func TestGetLatestReleaseTag_NetworkError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network-dependent tests in short mode")
	}

	releaseChecker := GithubReleaseChecker{}

	t.Run("invalid URL", func(t *testing.T) {
		_, err := releaseChecker.GetLatestReleaseTag("http://invalid-url")
		if err == nil {
			t.Error("Expected error for invalid URL, got nil")
		}
	})

	t.Run("connection refused", func(t *testing.T) {
		_, err := releaseChecker.GetLatestReleaseTag("http://localhost:19999")
		if err == nil {
			t.Error("Expected error for connection refused, got nil")
		}
	})
}

func TestDownload(t *testing.T) {
	// Setup test server with various responses
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/success":
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, "file content")
		case "/notfound":
			w.WriteHeader(http.StatusNotFound)
		case "/servererror":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(mockServer.Close)

	tempDir := t.TempDir()
	restrictedDir := filepath.Join(tempDir, "restricted")
	err := os.MkdirAll(restrictedDir, 0555)
	if err != nil {
		t.Fatalf("Failed to create restricted dir: %v", err)
	}

	filePath := filepath.Join(tempDir, "testfile.txt")
	downloader := GitHubFileDownloader{}

	tests := []struct {
		name        string
		filePath    string
		url         string
		wantErr     bool
		errContains string
	}{
		{
			name:     "successful download",
			filePath: filePath,
			url:      mockServer.URL + "/success",
			wantErr:  false,
		},
		{
			name:        "not found error",
			filePath:    filePath,
			url:         mockServer.URL + "/notfound",
			wantErr:     true,
			errContains: "file not found",
		},
		{
			name:        "server error",
			filePath:    filePath,
			url:         mockServer.URL + "/servererror",
			wantErr:     true,
			errContains: "failed to download file: HTTP 500",
		},
		{
			name:        "directory without permissions",
			filePath:    filepath.Join(restrictedDir, "permissions.txt"),
			url:         mockServer.URL + "/success",
			wantErr:     true,
			errContains: "permission denied",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := downloader.Download(test.filePath, test.url)
			t.Cleanup(func() {
				os.Remove(test.filePath)
			})

			if (err != nil) != test.wantErr {
				t.Errorf("Download() error = %v, wantErr %v", err, test.wantErr)
				return
			}

			if test.wantErr {
				if test.errContains != "" && !strings.Contains(err.Error(), test.errContains) {
					t.Errorf("Download() error = %v, want error containing %q", err, test.errContains)
				}
				return
			}

			// Verify file was created with correct content
			if _, err := os.Stat(test.filePath); os.IsNotExist(err) {
				t.Errorf("Download() file %q was not created", test.filePath)
			}
		})
	}
}

func TestDownload_NetworkErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network error tests in short mode")
	}

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "testfile.txt")
	downloader := GitHubFileDownloader{}

	t.Run("invalid URL", func(t *testing.T) {
		err := downloader.Download(filePath, "http://invalid-url")
		if err == nil {
			t.Error("Expected error for invalid URL, got nil")
		}
	})

	t.Run("connection refused", func(t *testing.T) {
		err := downloader.Download(filePath, "http://localhost:19999")
		if err == nil {
			t.Error("Expected error for connection refused, got nil")
		}
	})
}

func TestDownload_ExistingPath(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("file already exists", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "new content")
		}))
		defer server.Close()

		// Create file first
		filePath := filepath.Join(tempDir, "existing.txt")
		err := os.WriteFile(filePath, []byte("old content"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Try to download to same path
		downloader := GitHubFileDownloader{}
		err = downloader.Download(filePath, server.URL)
		if err != nil {
			t.Errorf("Download() error = %v, expected success", err)
		}

		// Verify file was overwritten
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != "new content" {
			t.Errorf("Download() file content = %q, want %q", string(content), "new content")
		}
	})
}

func TestIsZipFile(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected bool
		setup    func(t testing.TB) string // returns file path
	}{
		{
			name:     "valid zip file",
			content:  []byte{0x50, 0x4B, 0x03, 0x04, 0x0A, 0x00, 0x00, 0x00},
			expected: true,
			setup:    createTempFilePath,
		},
		{
			name:     "empty zip file",
			content:  []byte{0x50, 0x4B, 0x05, 0x06, 0x00, 0x00, 0x00, 0x00},
			expected: true,
			setup:    createTempFilePath,
		},
		{
			name:     "not a zip file",
			content:  []byte("This is not a ZIP file"),
			expected: false,
			setup:    createTempFilePath,
		},
		{
			name:     "empty file",
			content:  []byte{},
			expected: false,
			setup:    createTempFilePath,
		},
		{
			name:     "partial zip signature",
			content:  []byte{0x50},
			expected: false,
			setup:    createTempFilePath,
		},
		{
			name:     "non-existent file",
			content:  nil,
			expected: false,
			setup: func(t testing.TB) string {
				return "nonexistent_file_123456789"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filePath := test.setup(t)

			// Only write content if this isn't the non-existent file test
			if test.content != nil {
				err := os.WriteFile(filePath, test.content, 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				defer os.Remove(filePath)
			}

			result, err := isZipFile(filePath)
			if test.content == nil {
				// For non-existent file, we expect an error
				assertError(t, err)
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if result != test.expected {
				t.Errorf("Expected %v, got %v", test.expected, result)
			}
		})
	}
}

func TestExtractFileFromZip(t *testing.T) {
	createZipFile := func(contents []byte, perm os.FileMode) string {
		tempDir := filepath.Join(os.TempDir(), "test-zip")
		if perm == 0 {
			perm = 0755
		}
		err := os.MkdirAll(tempDir, perm)
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}

		zipPath := filepath.Join(tempDir, "test.zip")

		// Create a zip file for no permission test case
		if perm&0200 == 0 {
			return zipPath
		}

		zipFile, err := os.Create(zipPath)
		if err != nil {
			t.Fatalf("Failed to create zip file: %v", err)
		}
		defer zipFile.Close()

		// Create a zip file for invalid zip file test case
		if contents != nil {
			err := os.WriteFile(zipPath, contents, 0644)
			if err != nil {
				t.Fatalf("Failed to create invalid zip: %v", err)
			}
			return zipPath
		}

		zipWriter := zip.NewWriter(zipFile)
		defer zipWriter.Close()

		// Add a file "testfile" with content "test content" to the zip file
		w, err := zipWriter.Create("testfile.txt")
		if err != nil {
			t.Fatalf("Failed to create file in zip: %v", err)
		}
		_, err = io.WriteString(w, "test content")
		if err != nil {
			t.Fatalf("Failed to write to zip file: %v", err)
		}

		return zipPath
	}

	tests := []struct {
		name        string
		content     []byte
		perm        os.FileMode
		targetFile  string
		wantContent string
		wantErr     bool
		errContains string
	}{
		{
			name:        "successful extraction",
			targetFile:  "testfile.txt",
			wantContent: "test content",
		},
		{
			name:        "file not found in zip",
			targetFile:  "nonexistent.txt",
			wantErr:     true,
			errContains: "not found in zip archive",
		},
		{
			name:        "invalid zip file",
			content:     []byte("not a zip file"),
			targetFile:  "testfile.txt",
			wantErr:     true,
			errContains: "not a valid zip file",
		},
		{
			name:        "permission error on extraction",
			perm:        0444,
			targetFile:  "testfile.txt",
			wantErr:     true,
			errContains: "permission denied",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			zipPath := createZipFile(test.content, test.perm)
			t.Cleanup(func() {
				err := os.RemoveAll(filepath.Dir(zipPath))
				if err != nil {
					t.Fatalf("Failed to remove temp dir: %v", err)
				}
			})

			gotPath, err := extractFileFromZip(zipPath, test.targetFile)

			if (err != nil) != test.wantErr {
				t.Errorf("extractFileFromZip() error = %v, wantErr %v", err, test.wantErr)
				return
			}

			if test.wantErr {
				if test.errContains != "" && err != nil {
					if err.Error() != "" && !strings.Contains(err.Error(), test.errContains) {
						t.Errorf("extractFileFromZip() error = %v, should contain %v", err.Error(), test.errContains)
					}
				}
				return
			}

			content, err := os.ReadFile(gotPath)
			if err != nil {
				t.Errorf("Failed to read extracted file: %v", err)
			}
			if string(content) != test.wantContent {
				t.Errorf("Extracted file content = %v, want %v", string(content), test.wantContent)
			}

			if _, err := os.Stat(zipPath); !os.IsNotExist(err) {
				err = os.Remove(zipPath)
				if err != nil {
					t.Fatalf("Failed to delete zip file: %v", err)
				}
				t.Errorf("Zip file was not deleted: %v", zipPath)
			}
		})
	}
}

func TestBackupFile(t *testing.T) {
	t.Run("successful backup", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalPath := filepath.Join(tmpDir, "testfile.txt")
		backupPath := originalPath + ".backup"

		// Create a test file with content
		err := os.WriteFile(originalPath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Test backup
		gotBackupPath, err := backupFile(originalPath)
		if err != nil {
			t.Fatalf("backupFile failed: %v", err)
		}

		// Verify backup path matches expected
		if gotBackupPath != backupPath {
			t.Errorf("Expected backup path %q, got %q", backupPath, gotBackupPath)
		}

		// Verify original file no longer exists
		if _, err := os.Stat(originalPath); !os.IsNotExist(err) {
			t.Errorf("Original file still exists after backup")
		}

		// Verify backup file exists and has correct content
		content, err := os.ReadFile(backupPath)
		if err != nil {
			t.Fatalf("Failed to read backup file: %v", err)
		}
		assertCorrectString(t, "test content", string(content))
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := backupFile("nonexistentfile.txt")
		assertError(t, err)
	})

	t.Run("permission denied", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		tmpDir := t.TempDir()
		restrictedFile := filepath.Join(tmpDir, "restricted.txt")

		// Create file but make parent directory read-only
		err := os.WriteFile(restrictedFile, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		err = os.Chmod(tmpDir, 0555) // Read-only directory
		if err != nil {
			t.Fatalf("Failed to change directory permissions: %v", err)
		}
		defer os.Chmod(tmpDir, 0755) // Clean up

		_, err = backupFile(restrictedFile)
		assertError(t, err)
	})
}

func TestRestoreFile(t *testing.T) {
	t.Run("successful restore", func(t *testing.T) {
		// Create a temporary directory for testing
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "source.txt")
		dstPath := filepath.Join(tmpDir, "destination.txt")

		// Create source file with content
		err := os.WriteFile(srcPath, []byte("restore test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		// Test restore
		err = restoreFile(srcPath, dstPath)
		if err != nil {
			t.Fatalf("restoreFile failed: %v", err)
		}

		// Verify destination file exists and has correct content
		content, err := os.ReadFile(dstPath)
		if err != nil {
			t.Fatalf("Failed to read destination file: %v", err)
		}
		assertCorrectString(t, "restore test content", string(content))

		// Verify source file still exists (restore shouldn't delete it)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			t.Errorf("Source file was deleted during restore")
		}
	})

	t.Run("non-existent source file", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := restoreFile("nonexistentfile.txt", filepath.Join(tmpDir, "dest.txt"))
		assertError(t, err)
	})

	t.Run("unwritable destination", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "source.txt")
		err := os.WriteFile(srcPath, []byte("content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		// Create unwritable directory
		unwritableDir := filepath.Join(tmpDir, "unwritable")
		if err := os.Mkdir(unwritableDir, 0000); err != nil {
			t.Fatalf("Failed to create unwritable directory: %v", err)
		}
		defer os.Chmod(unwritableDir, 0755) // Clean up permissions after test

		err = restoreFile(srcPath, filepath.Join(unwritableDir, "dest.txt"))
		assertError(t, err)
	})

	t.Run("destination directory doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "source.txt")
		err := os.WriteFile(srcPath, []byte("content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		nonexistentDir := filepath.Join(tmpDir, "nonexistent")
		err = restoreFile(srcPath, filepath.Join(nonexistentDir, "dest.txt"))
		assertError(t, err)
	})
}

type MockReleaseChecker struct{}

func (rc MockReleaseChecker) GetLatestReleaseTag(apiURL string) (string, error) {
	return "1.2.3", nil
}

type FailReleaseChecker struct{}

func (rc FailReleaseChecker) GetLatestReleaseTag(apiURL string) (string, error) {
	return "", errors.New("failed to get release tag")
}

type OrdinaryFileDownloader struct{}

func (d OrdinaryFileDownloader) Download(filePath string, url string) error {
	return os.WriteFile(filePath, []byte("mock content"), 0644)
}

type ZipFileDownloader struct{}

func (d ZipFileDownloader) Download(filePath string, url string) error {
	zipFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	w, err := zipWriter.Create(filepath.Base(filePath))
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, "mock content")
	return err
}

type FailFileDownloader struct{}

func (d FailFileDownloader) Download(filePath string, url string) error {
	return errors.New("failed to download file")
}

func TestUpdateFile(t *testing.T) {
	tests := []struct {
		name           string
		oldContent     string
		releaseChecker ReleaseChecker
		downloader     FileDownloader
		errorExpected  bool
	}{
		{
			name:           "Update nonexistent file",
			oldContent:     "",
			releaseChecker: MockReleaseChecker{},
			downloader:     OrdinaryFileDownloader{},
			errorExpected:  false,
		},
		{
			name:           "Update existing file",
			oldContent:     "old content",
			releaseChecker: MockReleaseChecker{},
			downloader:     OrdinaryFileDownloader{},
			errorExpected:  false,
		},
		{
			name:           "Update existing zip file",
			oldContent:     "old content",
			releaseChecker: MockReleaseChecker{},
			downloader:     ZipFileDownloader{},
			errorExpected:  false,
		},
		{
			name:           "Fail to get release tag",
			oldContent:     "old content",
			releaseChecker: FailReleaseChecker{},
			downloader:     OrdinaryFileDownloader{},
			errorExpected:  true,
		},
		{
			name:           "Fail to download file",
			oldContent:     "old content",
			releaseChecker: MockReleaseChecker{},
			downloader:     FailFileDownloader{},
			errorExpected:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tempFile := createTempFilePath(t)

			file := File{
				filePath: tempFile,
			}

			if test.oldContent != "" {
				err := os.WriteFile(file.filePath, []byte(test.oldContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
			}

			file.releaseChecker = test.releaseChecker
			file.downloader = test.downloader

			err := updateFile(file, true)

			if test.errorExpected {
				assertError(t, err)
				return
			} else {
				assertNoError(t, err)
			}

			content, err := os.ReadFile(file.filePath)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			assertCorrectString(t, "mock content", string(content))

			// Check that the versions file is updated
			versionsFilePath := filepath.Join(filepath.Dir(file.filePath), "versions.json")
			versionsContent, err := os.ReadFile(versionsFilePath)
			if err != nil {
				t.Fatalf("Failed to read versions file: %v", err)
			}

			var versions map[string]string
			err = json.Unmarshal(versionsContent, &versions)
			if err != nil {
				t.Fatalf("Failed to unmarshal versions file: %v", err)
			}

			assertCorrectString(t, "1.2.3", versions[filepath.Base(file.filePath)])

			// Check that there are no zip files in the folder
			files, err := os.ReadDir(filepath.Dir(file.filePath))
			if err != nil {
				t.Fatalf("Failed to read directory: %v", err)
			}

			for _, f := range files {
				if strings.HasSuffix(f.Name(), ".zip") {
					t.Errorf("Found zip file %s in directory", f.Name())
				}
			}
		})
	}

}
