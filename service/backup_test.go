package service_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/IceWhaleTech/IceWhale-Files-Backup/service"
	"github.com/stretchr/testify/assert"
)

func createFileWithContent(dir, name, content string) error {
	filePath := filepath.Join(dir, name)
	err := os.WriteFile(filePath, []byte(content), 0o600)
	return err
}

func TestFilterBackupFiles(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create non-backup files
	assert.NoError(t, createFileWithContent(tempDir, "file1.txt", "test content"))
	assert.NoError(t, createFileWithContent(tempDir, "file2.txt", "test content"))

	// Create backup files
	assert.NoError(t, createFileWithContent(tempDir, "file1-backup-2022-08-19-15-30-45-123.txt", "test content"))
	assert.NoError(t, createFileWithContent(tempDir, "file2-backup-2022-08-19-15-30-45-124.txt", "test content"))

	// Create nested folder
	nestedFolder := filepath.Join(tempDir, "nested")
	assert.NoError(t, os.Mkdir(nestedFolder, 0o755))

	// Create non-backup file in nested folder
	assert.NoError(t, createFileWithContent(nestedFolder, "nested_file1.txt", "test content"))

	// Create backup file in nested folder
	assert.NoError(t, createFileWithContent(nestedFolder, "nested_file1-backup-2022-08-19-15-30-45-125.txt", "test content"))

	nonBackupFiles, err := service.FilterBackupFiles(tempDir)
	assert.NoError(t, err)

	expectedFiles := []string{
		filepath.Join(tempDir, "file1.txt"),
		filepath.Join(tempDir, "file2.txt"),
		filepath.Join(nestedFolder, "nested_file1.txt"),
	}

	assert.ElementsMatch(t, expectedFiles, nonBackupFiles)
}
