package utils

import (
	"archive/zip"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
)

// TODO: Unify cleanups (e.g. RemoveAll vs Remove)
// TODO: Unify setups (e.g. create temp subdirs within the temp dir for diff tests)

func TestFileExists(t *testing.T) {
	tempFile, cleanup := CreateTempFile(t)
	t.Cleanup(cleanup)

	// Test when file exists
	if !FileExists(tempFile) {
		t.Errorf("fileExists shall return true for existing file")
	}

	// Test when file does not exist
	nonExistentPath := tempFile + "_nonexistent"
	if FileExists(nonExistentPath) {
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
			got, err := ExpandPath(test.input)
			if err != nil {
				t.Errorf("expandPath returned error: %v", err)
			}
			AssertCorrectString(t, test.expected, got)
		})
	}
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
			setup:    CreateTempFilePath,
		},
		{
			name:     "empty zip file",
			content:  []byte{0x50, 0x4B, 0x05, 0x06, 0x00, 0x00, 0x00, 0x00},
			expected: true,
			setup:    CreateTempFilePath,
		},
		{
			name:     "not a zip file",
			content:  []byte("This is not a ZIP file"),
			expected: false,
			setup:    CreateTempFilePath,
		},
		{
			name:     "empty file",
			content:  []byte{},
			expected: false,
			setup:    CreateTempFilePath,
		},
		{
			name:     "partial zip signature",
			content:  []byte{0x50},
			expected: false,
			setup:    CreateTempFilePath,
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

			result, err := IsZipFile(filePath)
			if test.content == nil {
				// For non-existent file, we expect an error
				AssertError(t, err)
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

			gotPath, err := ExtractFileFromZip(zipPath, test.targetFile)

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
		gotBackupPath, err := BackupFile(originalPath)
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
		AssertCorrectString(t, "test content", string(content))
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := BackupFile("nonexistentfile.txt")
		AssertError(t, err)
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

		_, err = BackupFile(restrictedFile)
		AssertError(t, err)
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
		err = RestoreFile(srcPath, dstPath)
		if err != nil {
			t.Fatalf("restoreFile failed: %v", err)
		}

		// Verify destination file exists and has correct content
		content, err := os.ReadFile(dstPath)
		if err != nil {
			t.Fatalf("Failed to read destination file: %v", err)
		}
		AssertCorrectString(t, "restore test content", string(content))

		// Verify source file still exists (restore shouldn't delete it)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			t.Errorf("Source file was deleted during restore")
		}
	})

	t.Run("non-existent source file", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := RestoreFile("nonexistentfile.txt", filepath.Join(tmpDir, "dest.txt"))
		AssertError(t, err)
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

		err = RestoreFile(srcPath, filepath.Join(unwritableDir, "dest.txt"))
		AssertError(t, err)
	})

	t.Run("destination directory doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "source.txt")
		err := os.WriteFile(srcPath, []byte("content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		nonexistentDir := filepath.Join(tmpDir, "nonexistent")
		err = RestoreFile(srcPath, filepath.Join(nonexistentDir, "dest.txt"))
		AssertError(t, err)
	})
}
