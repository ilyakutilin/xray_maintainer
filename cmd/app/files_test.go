package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"testing"
)

// contains checks if string s contains substring substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

func assertCorrectString(t testing.TB, want, got string) {
	t.Helper()
	if got != want {
		t.Errorf("Expected %q, got %q", want, got)
	}
}

func assertError(t testing.TB, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("Wanted an error but didn't get one")
	}
}

func assertNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Wanted no error but got: %v", err)
	}
}

func createTempFile(t testing.TB) (string, func()) {
	t.Helper()
	tempFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	return tempFile.Name(), func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}
}

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
	t.Cleanup(cleanup)

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
	t.Cleanup(cleanup)

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
	// Setup test cases
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

			got, err := getLatestReleaseTag(server.URL)

			if (err != nil) != tt.wantErr {
				t.Errorf("getLatestReleaseTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
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

	t.Run("invalid URL", func(t *testing.T) {
		_, err := getLatestReleaseTag("http://invalid-url")
		if err == nil {
			t.Error("Expected error for invalid URL, got nil")
		}
	})

	t.Run("connection refused", func(t *testing.T) {
		_, err := getLatestReleaseTag("http://localhost:19999")
		if err == nil {
			t.Error("Expected error for connection refused, got nil")
		}
	})
}
