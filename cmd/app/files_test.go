package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ilyakutilin/xray_maintainer/utils"
)

// TODO: Unify cleanups (e.g. RemoveAll vs Remove)
// TODO: Unify setups (e.g. create temp subdirs within the temp dir for diff tests)

func TestGetStoredReleaseTag(t *testing.T) {
	// Versions file does not exist
	t.Run("Versions file does not exist", func(t *testing.T) {
		tag, err := getStoredReleaseTag("testfile", "doesnotexist.json")
		utils.AssertCorrectString(t, "", tag)
		utils.AssertNoError(t, err)
	})

	versionsFile, cleanup := utils.CreateTempFile(t)

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
			utils.AssertCorrectString(t, test.expectedReturn, tag)
			if test.expectedError == nil {
				utils.AssertNoError(t, err)
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
		utils.AssertNoError(t, err)
		data, err := os.ReadFile(filepath.Join(os.TempDir(), "doesnotexist.json"))
		utils.AssertNoError(t, err)
		utils.AssertCorrectString(t, "{\n  \"testfile\": \"1.2.3\"\n}", string(data))
	})

	versionsFile, cleanup := utils.CreateTempFile(t)

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
				utils.AssertError(t, err)
				return
			} else {
				utils.AssertNoError(t, err)
			}
			data, err := os.ReadFile(versionsFile)
			utils.AssertNoError(t, err)
			var actualMap map[string]string
			err = json.Unmarshal(data, &actualMap)
			utils.AssertNoError(t, err)
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

			utils.AssertCorrectString(t, tt.want, got)
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
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		t.Run(test.name, func(t *testing.T) {
			tempFile := utils.CreateTempFilePath(t)

			testApp := &Application{
				debug:   true,
				logger:  GetLogger(false),
				workdir: filepath.Dir(tempFile),
			}

			file := File{
				repo: Repo{Filename: filepath.Base(tempFile)},
			}

			if test.oldContent != "" {
				err := os.WriteFile(tempFile, []byte(test.oldContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
			}

			file.releaseChecker = test.releaseChecker
			file.downloader = test.downloader

			err := testApp.updateFile(ctx, file)

			if test.errorExpected {
				utils.AssertError(t, err)
				return
			} else {
				utils.AssertNoError(t, err)
			}

			content, err := os.ReadFile(tempFile)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			utils.AssertCorrectString(t, "mock content", string(content))

			// Check that the versions file is updated
			versionsFilePath := filepath.Join(filepath.Dir(tempFile), "versions.json")
			versionsContent, err := os.ReadFile(versionsFilePath)
			if err != nil {
				t.Fatalf("Failed to read versions file: %v", err)
			}

			var versions map[string]string
			err = json.Unmarshal(versionsContent, &versions)
			if err != nil {
				t.Fatalf("Failed to unmarshal versions file: %v", err)
			}

			utils.AssertCorrectString(t, "1.2.3", versions[filepath.Base(tempFile)])

			// Check that there are no zip files in the folder
			files, err := os.ReadDir(filepath.Dir(tempFile))
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

func TestUpdateMultipleFiles(t *testing.T) {
	tests := []struct {
		name           string
		ReleaseChecker ReleaseChecker
		downloader     FileDownloader
		errorExpected  bool
	}{
		{
			name:           "Successful update",
			ReleaseChecker: MockReleaseChecker{},
			downloader:     OrdinaryFileDownloader{},
			errorExpected:  false,
		},
		{
			name:           "Failed update",
			ReleaseChecker: MockReleaseChecker{},
			downloader:     FailFileDownloader{},
			errorExpected:  true,
		},
	}

	for _, test := range tests {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		t.Run(test.name, func(t *testing.T) {
			tempFileOne := utils.CreateTempFilePath(t)
			tempFileTwo := utils.CreateTempFilePath(t)

			testApp := &Application{
				debug:   true,
				logger:  GetLogger(false),
				workdir: filepath.Dir(tempFileOne),
			}

			fn := func(repo Repo) File {
				return File{
					repo:           repo,
					releaseChecker: test.ReleaseChecker,
					downloader:     test.downloader,
				}
			}

			filenameOne := filepath.Base(tempFileOne)
			filenameTwo := filepath.Base(tempFileTwo)

			repos := []Repo{
				{Name: filenameOne, Filename: filenameOne},
				{Name: filenameTwo, Filename: filenameTwo},
			}

			err := testApp.updateMultipleFiles(ctx, repos, fn)

			if test.errorExpected {
				utils.AssertError(t, err)
				utils.AssertErrorContains(t, err, "failed to download file\n")
				return
			} else {
				utils.AssertNoError(t, err)
			}
		})
	}
}
