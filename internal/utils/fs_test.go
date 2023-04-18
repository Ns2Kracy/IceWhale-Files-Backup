package utils_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/IceWhaleTech/IceWhale-Files-Backup/internal/utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestSizeAndCount(t *testing.T) {
	defer goleak.VerifyNone(t)

	workerLimit := 4

	// Test case 1: Empty directory
	t.Run("EmptyDirectory", func(t *testing.T) {
		// Create a temporary directory and defer its deletion
		tempDir, err := os.MkdirTemp("", "test-empty")
		if err != nil {
			t.Fatalf("Error creating temporary directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		totalSize, fileCount, err := utils.SizeAndCount(tempDir, workerLimit)
		assert.NoError(t, err, "Error getting folder stats")
		assert.Equal(t, int64(0), totalSize, "Expected total size to be 0")
		assert.Equal(t, 0, fileCount, "Expected file count to be 0")
	})

	// Test case 2: Directory with files
	t.Run("DirectoryWithFiles", func(t *testing.T) {
		// Create a temporary directory and defer its deletion
		tempDir, err := os.MkdirTemp("", "test-files")
		if err != nil {
			t.Fatalf("Error creating temporary directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create some test files in the temporary directory
		numFiles := 3
		totalSize := int64(0)
		for i := 0; i < numFiles; i++ {
			filePath := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
			content := []byte("test content")
			err := os.WriteFile(filePath, content, 0o600)
			if err != nil {
				t.Fatalf("Error creating test file: %v", err)
			}
			totalSize += int64(len(content))
		}

		totalSize, fileCount, err := utils.SizeAndCount(tempDir, workerLimit)
		assert.NoError(t, err, "Error getting folder stats")
		assert.Equal(t, totalSize, totalSize, "Total size mismatch")
		assert.Equal(t, numFiles, fileCount, "File count mismatch")
	})
}

func createTempFileWithContent(dir, content string) (string, error) {
	file, err := os.CreateTemp(dir, "test")
	if err != nil {
		return "", err
	}

	if _, err := file.WriteString(content); err != nil {
		return "", err
	}

	if err := file.Close(); err != nil {
		return "", err
	}

	return file.Name(), nil
}
