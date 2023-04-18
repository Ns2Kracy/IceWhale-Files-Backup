package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"

	"github.com/IceWhaleTech/IceWhale-Files-Backup/codegen"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/common"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/pkg/config"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/service"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func createFileWithContent(dir, name, content string) error {
	filePath := filepath.Join(dir, name)
	err := os.WriteFile(filePath, []byte(content), 0o600)
	return err
}

func TestGetAllBackups(t *testing.T) {
	defer goleak.VerifyNone(t)

	// Create temporary directory for test
	tmpDataRootDir, err := os.MkdirTemp("", "test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDataRootDir)

	config.AppInfo.DataRootPath = tmpDataRootDir

	tmpDir := filepath.Join(tmpDataRootDir, common.BackupRootFolder)

	// Create test folders
	backupFolder1 := "client1/folder1"
	backupFolder2 := "client1/folder2"
	backupFolder3 := "client2/folder1"

	client1Folder1 := filepath.Join(tmpDir, backupFolder1)
	client1Folder2 := filepath.Join(tmpDir, backupFolder2)
	client2Folder1 := filepath.Join(tmpDir, backupFolder3)
	assert.NoError(t, os.MkdirAll(client1Folder1, 0o755))
	assert.NoError(t, os.MkdirAll(client1Folder2, 0o755))
	assert.NoError(t, os.MkdirAll(client2Folder1, 0o755))

	// Create test metadata files
	client1Folder1Metadata := filepath.Join(client1Folder1, common.MetadataFileName)
	client1Folder2Metadata := filepath.Join(client1Folder2, common.MetadataFileName)
	client2Folder1Metadata := filepath.Join(client2Folder1, common.MetadataFileName)

	backup1 := codegen.FolderBackup{BackupFolderPath: &backupFolder1}
	backup2 := codegen.FolderBackup{BackupFolderPath: &backupFolder2}
	backup3 := codegen.FolderBackup{BackupFolderPath: &backupFolder3}

	buf1, err := json.Marshal(backup1)
	assert.NoError(t, err)

	buf2, err := json.Marshal(backup2)
	assert.NoError(t, err)

	buf3, err := json.Marshal(backup3)
	assert.NoError(t, err)

	assert.NoError(t, os.WriteFile(client1Folder1Metadata, buf1, 0o600))
	assert.NoError(t, os.WriteFile(client1Folder2Metadata, buf2, 0o600))
	assert.NoError(t, os.WriteFile(client2Folder1Metadata, buf3, 0o600))

	// Run test
	ctx := context.Background()
	allBackups, err := service.NewBackupService().GetAllBackups(ctx, false)
	assert.NoError(t, err)

	// Check results
	assert.Equal(t, 2, len(allBackups))

	client1Backups, ok := allBackups["client1"]
	assert.True(t, ok)
	assert.Equal(t, 2, len(client1Backups))

	client2Backups, ok := allBackups["client2"]
	assert.True(t, ok)
	assert.Equal(t, 1, len(client2Backups))

	// Check backup folder paths are set
	for _, backups := range allBackups {
		for _, backup := range backups {
			assert.NotNil(t, backup.BackupFolderPath)
		}
	}
}

func TestNormalizeWindowsPath(t *testing.T) {
	defer goleak.VerifyNone(t)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Windows path with drive letter",
			input:    `C:\Users\username\Documents\file.txt`,
			expected: `C/Users/username/Documents/file.txt`,
		},
		{
			name:     "Windows path without drive letter",
			input:    `\Users\username\Documents\file.txt`,
			expected: `/Users/username/Documents/file.txt`,
		},
		{
			name:     "Unix path",
			input:    "/home/username/Documents/file.txt",
			expected: "/home/username/Documents/file.txt",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := service.Normalize(test.input)
			if output != test.expected {
				t.Errorf("Expected: %s, got: %s", test.expected, output)
			}
		})
	}
}

func TestGetBackupsByPath(t *testing.T) {
	// create a temporary directory for testing
	dir, err := os.MkdirTemp("", "DATA")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	config.AppInfo.DataRootPath = dir

	// create a metadata file for a backup
	backupFolderPath := "backup"
	backup := &codegen.FolderBackup{
		BackupFolderPath: &backupFolderPath,
	}
	err = service.SaveMetadata(backup)
	assert.NoError(t, err)

	// test GetBackupsByPath with full set to false
	backupFolderFullpath := filepath.Join(dir, backupFolderPath)
	backups, err := service.GetBackupsByPath(backupFolderFullpath, false)
	assert.NoError(t, err)
	assert.Len(t, backups, 1)
	assert.Equal(t, *backup.BackupFolderPath, *backups[0].BackupFolderPath)

	// test GetBackupsByPath with full set to true
	backups, err = service.GetBackupsByPath(backupFolderFullpath, true)
	assert.NoError(t, err)
	assert.Len(t, backups, 1)
	assert.Equal(t, *backup.BackupFolderPath, *backups[0].BackupFolderPath)
	assert.NotNil(t, backups[0].BackupFolderCount)
	assert.Equal(t, 1, *backups[0].BackupFolderCount)
	assert.NotNil(t, backups[0].BackupFolderSize)
	assert.Equal(t, int64(37), *backups[0].BackupFolderSize)
}

func TestBackup(t *testing.T) {
	defer goleak.VerifyNone(t)

	// Create a temporary directory to store the test files
	tmpDir, err := os.MkdirTemp("", "test")
	assert.Nil(t, err)
	defer os.RemoveAll(tmpDir)

	testCases := []struct {
		name    string
		move    bool
		content string
	}{
		{
			name:    "copy",
			move:    false,
			content: "hello world",
		},
		{
			name:    "move",
			move:    true,
			content: "hello world",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a source file with some content
			srcFileName := "testfile.txt"
			err = createFileWithContent(tmpDir, srcFileName, tc.content)
			assert.Nil(t, err)
			srcFilePath := filepath.Join(tmpDir, srcFileName)

			// Backup the file
			backupPath, err := service.BackupFile(srcFilePath, tc.move)
			assert.Nil(t, err)

			// Check if the backup file exists and has the right naming pattern
			backupFilePattern := regexp.MustCompile(`-backup-\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2}-\d{3}`)
			assert.True(t, backupFilePattern.MatchString(filepath.Base(backupPath)))

			// Check if the backup file has the same content as the source file (if not moved)
			if !tc.move {
				backupContent, err := os.ReadFile(backupPath)
				assert.Nil(t, err)
				assert.Equal(t, tc.content, string(backupContent))
			}

			// Check if the source file is removed (if moved)
			if tc.move {
				_, err := os.Stat(srcFilePath)
				assert.NotNil(t, err)
				assert.True(t, errors.Is(err, os.ErrNotExist))
			}
		})
	}
}

func TestFilterBackupFiles(t *testing.T) {
	defer goleak.VerifyNone(t)

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "test")
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

func TestBackupWithFilterBackupFiles(t *testing.T) {
	defer goleak.VerifyNone(t)

	// Create a temporary directory to store the test files
	tmpDir, err := os.MkdirTemp("", "test")
	assert.Nil(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a source file with some content
	srcFileName := "testfile.txt"
	err = createFileWithContent(tmpDir, srcFileName, "hello world")
	assert.Nil(t, err)
	srcFilePath := filepath.Join(tmpDir, srcFileName)

	// Create some additional non-backup files
	err = createFileWithContent(tmpDir, "non-backup-1.txt", "hello world 1")
	assert.Nil(t, err)
	err = createFileWithContent(tmpDir, "non-backup-2.txt", "hello world 2")
	assert.Nil(t, err)

	// Backup the file with 'move' set to true (rename)
	_, err = service.BackupFile(srcFilePath, true)
	assert.Nil(t, err)

	// Filter backup files
	nonBackupFiles, err := service.FilterBackupFiles(tmpDir)
	assert.Nil(t, err)

	// Check if there are exactly 2 non-backup files
	assert.Equal(t, 2, len(nonBackupFiles))

	// Check if the non-backup files have the expected names
	expectedNonBackupFiles := map[string]bool{
		filepath.Join(tmpDir, "non-backup-1.txt"): false,
		filepath.Join(tmpDir, "non-backup-2.txt"): false,
	}
	for _, path := range nonBackupFiles {
		_, ok := expectedNonBackupFiles[path]
		assert.True(t, ok, "unexpected non-backup file: %s", path)
		expectedNonBackupFiles[path] = true
	}

	// Check if all the expected non-backup files are found
	for path, found := range expectedNonBackupFiles {
		assert.True(t, found, "non-backup file not found: %s", path)
	}
}

func TestSaveAndLoadMetadata(t *testing.T) {
	defer goleak.VerifyNone(t)

	// create a temporary directory for testing
	dir, err := os.MkdirTemp("", "DATA")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	config.AppInfo.DataRootPath = dir

	// Create a temporary directory to store the test files
	backupFolderPath := "backup"

	// Set up test data
	testBackup := &codegen.FolderBackup{
		BackupFolderCount: lo.ToPtr(5),
		BackupFolderPath:  &backupFolderPath,
		BackupFolderSize:  lo.ToPtr(int64(1024)),
		ClientFolderFileHashes: lo.ToPtr(map[string]string{
			"file1.txt": "hash1",
			"file2.txt": "hash2",
		}),
		ClientFolderFileSizes: lo.ToPtr(map[string]int64{
			"file1.txt": 100,
			"file2.txt": 200,
		}),
		ClientFolderPath:    lo.ToPtr("/files"),
		ClientID:            lo.ToPtr("client1"),
		ClientName:          lo.ToPtr("Client One"),
		ClientType:          lo.ToPtr("desktop"),
		InProgress:          lo.ToPtr(false),
		KeepHistoryCopy:     lo.ToPtr(true),
		LastBackupSucceeded: lo.ToPtr(true),
		LastBackupTime:      lo.ToPtr(int64(1619977711000)),
		RemainingCount:      lo.ToPtr(12),
	}

	// Save metadata
	err = service.SaveMetadata(testBackup)
	assert.NoError(t, err)

	// Load metadata
	backupFolderFullpath := filepath.Join(dir, backupFolderPath)
	loadedBackup, err := service.LoadMetadata(backupFolderFullpath)
	assert.NoError(t, err)

	// Check that loaded backup matches the original backup
	if !reflect.DeepEqual(testBackup, loadedBackup) {
		t.Errorf("loaded backup does not match original backup: expected %+v, got %+v", testBackup, loadedBackup)
	}
}
