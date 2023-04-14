package service

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/IceWhaleTech/IceWhale-Files-Backup/codegen"
	"github.com/cespare/xxhash/v2"
)

func Proceed(backup codegen.FolderBackup) (*codegen.FolderBackup, error) {
	// TODO - convert Windows path to Unix path
	backupFolderPath := filepath.Join("Backup", *backup.ClientId, *backup.ClientFolderPath)

	nonBackupFiles, err := FilterBackupFiles(backupFolderPath)
	if err != nil {
		return nil, err
	}

	if backup.ClientFolderFileHashes == nil {
		return nil, fmt.Errorf("client folder file hashes is nil")
	}

	for _, file := range nonBackupFiles {

		// get the size of the file
		fileInfo, err := os.Stat(file)
		if err != nil {
			return nil, err
		}

		// if file has been deleted, or its size/hash has changed, then backup it
		if size, ok := (*backup.ClientFolderFileSizes)[file]; !ok || size != fileInfo.Size() {
			// TODO - backup the file with different size by copying.

			continue
		}

		fileHash, err := FileHash(file)
		if err != nil {
			return nil, err
		}

		if hash, ok := (*backup.ClientFolderFileHashes)[file]; !ok || hash != fileHash {
			// TODO - backup the file with the same size but different hash by renaming.

			// WebDAV doesn't support modification time and checksum, so Rclone will be looking
			// at the file size only. Even the hashes are different, Rclone will still think
			// the file is the same if the sizes are identical. So we need to rename the file so
			// Rclone would proceed to transfer the newer file from client.

			continue
		}
	}

	folderBackup := codegen.FolderBackup{
		BackupFolderPath: &backupFolderPath,
		// TODO
	}

	return &folderBackup, nil
}

func isBackupFile(filename string) bool {
	backupFilePattern := regexp.MustCompile(`-backup-\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2}-\d{3}`)
	return backupFilePattern.MatchString(filename)
}

func FilterBackupFiles(root string) ([]string, error) {
	var nonBackupFiles []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !isBackupFile(d.Name()) {
			nonBackupFiles = append(nonBackupFiles, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return nonBackupFiles, nil
}

// TODO - implement a scheduled job to calculate the hash of all files in the backup folder

func FileHash(path string) (string, error) {
	// TODO - read from in-memory cache if it exists
	// TODO - read from the checksum file if it exists

	// TODO - calculate the hash of the file if it doesn't exist
	hash, err := XXHash(path)
	if err != nil {
		return "", err
	}

	// TODO - write the hash to the checksum file
	// TODO - write to the in-memory cache
	return hash, nil
}

func XXHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hash := xxhash.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func Backup(path string) error {
	// TODO - backup the file to the backup folder
	return nil
}
