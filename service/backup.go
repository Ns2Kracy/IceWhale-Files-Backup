package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/codegen"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/common"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/internal/utils"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/pkg/config"
	"github.com/cespare/xxhash/v2"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

type BackupService struct {
	backupRoot string
	// clientID -> folder path -> cancel function
	BackupsInProgress map[string]map[string]context.CancelFunc // TODO: add ongoing backup to BackupsInProgress
}

func (b *BackupService) GetAllBackups(ctx context.Context, full bool) (map[string][]codegen.FolderBackup, error) {
	allBackups := map[string][]codegen.FolderBackup{}
	// for each child folder under backupRoot, call GetBackupsByPath
	err := filepath.WalkDir(b.backupRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		// if the path is the backupRoot, skip it
		if path == b.backupRoot {
			return nil
		}

		// get the clientID
		clientID := filepath.Base(path)

		// get the backups
		backupsByClient, err := GetBackupsByPath(path, full)
		if err != nil {
			return err
		}

		allBackups[clientID] = backupsByClient

		return fs.SkipDir
	})
	if err != nil {
		return nil, err
	}

	return allBackups, nil
}

func (b *BackupService) GetBackupsByClientID(ctx context.Context, clientID string, full bool) ([]codegen.FolderBackup, error) {
	// traverse the backup folder and get all the backups
	backupRootByClient := filepath.Join(b.backupRoot, clientID)
	backups, err := GetBackupsByPath(backupRootByClient, full)
	if err != nil {
		return nil, err
	}

	return backups, nil
}

func (b *BackupService) IsClientIDExists(clientID string) (bool, error) {
	backupRootByClient := filepath.Join(b.backupRoot, clientID)
	if _, err := os.Stat(backupRootByClient); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (b *BackupService) IsBackupExists(clientID, clientFolderPath string) (bool, error) {
	backupRootByClient := filepath.Join(b.backupRoot, clientID)

	// convert Windows path to Unix path
	clientFolderPath = Normalize(clientFolderPath)

	backupFolderPath := filepath.Join(backupRootByClient, clientFolderPath)
	if _, err := os.Stat(backupFolderPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (b *BackupService) Proceed(backup codegen.FolderBackup) (*codegen.FolderBackup, error) {
	if backup.ClientFolderFileHashes == nil {
		return nil, fmt.Errorf("client folder file hashes is nil")
	}

	// convert Windows path to Unix path
	clientFolderPathNormalized := Normalize(*backup.ClientFolderPath)
	backupFolderPath := filepath.Join(common.BackupRootFolder, *backup.ClientID, clientFolderPathNormalized)
	if err := os.MkdirAll(backupFolderPath, 0o755); err != nil {
		return nil, err
	}
	backup.BackupFolderPath = &backupFolderPath

	// checkpoint
	if err := SaveMetadata(&backup); err != nil {
		return nil, err
	}

	backupFolderFullpath := filepath.Join(config.AppInfo.DataRootPath, backupFolderPath)

	nonBackupFiles, err := FilterBackupFiles(backupFolderFullpath)
	if err != nil {
		return nil, err
	}
	backup.InProgress = lo.ToPtr(true)

	// checkpoint
	if err := SaveMetadata(&backup); err != nil {
		return nil, err
	}

	for _, file := range nonBackupFiles {

		shouldBackup := false
		shouldMove := false

		// check by comparing the sizes
		{
			fileInfo, err := os.Stat(file)
			if err != nil {
				return nil, err
			}

			// if file has been deleted, or its size/hash has changed, then backup it
			if size, ok := (*backup.ClientFolderFileSizes)[file]; !ok {

				// file doesn't exist in the client folder, so consider it has been deleted.
				// thus the file should be moved instead of copied.
				shouldMove = true
				shouldBackup = true
			} else if size != fileInfo.Size() {
				// file size has changed, so backup it by copying.
				shouldBackup = true
			}
		}

		// check again by comparing the hashes if the sizes are identical
		if !shouldBackup {
			fileHash, err := FileHash(file)
			if err != nil {
				return nil, err
			}

			if hash, ok := (*backup.ClientFolderFileHashes)[file]; !ok || hash != fileHash {
				// backup the file with the same size but different hash by renaming.

				// WebDAV doesn't support modification time and checksum, so Rclone will be looking
				// at the file size only. Even the hashes are different, Rclone will still think
				// the file is the same if the sizes are identical. So we need to rename the file so
				// Rclone would proceed to transfer the newer file from client.
				shouldMove = true
				shouldBackup = true
			}
		}

		if !shouldBackup {
			logger.Info("file is up to date, no backup needed.", zap.String("file", file))
			continue
		}

		backupFilePath, err := BackupFile(file, shouldMove)
		if err != nil {
			logger.Error("failed to backup file", zap.String("file", file), zap.Error(err))
			return nil, err
		}

		logger.Info("file has been backed up", zap.String("file", file), zap.String("backup", backupFilePath))
	}

	backup.LastBackupTime = lo.ToPtr(time.Now().Unix())

	// checkpoint
	backup.ClientFolderFileHashes = nil
	backup.ClientFolderFileSizes = nil

	if err := SaveMetadata(&backup); err != nil {
		return nil, err
	}

	return &backup, nil
}

func (b *BackupService) DeleteBackupsByClientID(ctx context.Context, clientID, clientFolderPath string) error {
	// convert Windows path to Unix path
	clientFolderPathNormalized := Normalize(clientFolderPath)

	backupRoot := filepath.Join(config.AppInfo.DataRootPath, common.BackupRootFolder)
	backupRootByClient := filepath.Join(backupRoot, clientID)

	backupFolderPath := filepath.Join(backupRootByClient, clientFolderPathNormalized)

	// check if the backup folder exists
	if _, err := os.Stat(backupFolderPath); err != nil {
		if os.IsNotExist(err) {
			logger.Error("backup folder doesn't exist", zap.String("path", backupFolderPath))
		} else {
			logger.Error("failed to check if backup folder exists", zap.String("path", backupFolderPath), zap.Error(err))
		}
		return err
	}

	// delete the backup folder
	currentPath := backupFolderPath

	for {
		if err := os.RemoveAll(currentPath); err != nil {
			logger.Error("failed to delete backup folder", zap.String("path", backupFolderPath), zap.Error(err))
			return err
		}

		currentPath = filepath.Dir(currentPath)

		if currentPath == backupRoot || currentPath == config.AppInfo.DataRootPath {
			break
		}

		// check if the current path is empty
		entries, err := os.ReadDir(currentPath)
		if err != nil {
			logger.Error("failed to read backup folder", zap.String("path", currentPath), zap.Error(err))
			return err
		}

		if len(entries) > 0 {
			break
		}
	}

	return nil
}

func NewBackupService() *BackupService {
	backupRoot := filepath.Join(config.AppInfo.DataRootPath, common.BackupRootFolder)

	if _, err := os.Stat(backupRoot); err != nil {
		if os.IsNotExist(err) {
			logger.Info("backup root folder doesn't exist, creating one", zap.String("path", backupRoot))
			if err := os.MkdirAll(backupRoot, 0o755); err != nil {
				logger.Error("failed to create backup root folder", zap.String("path", backupRoot), zap.Error(err))
			}
		} else {
			logger.Error("failed to check if backup root folder exists", zap.String("path", backupRoot), zap.Error(err))
		}
	}

	return &BackupService{
		backupRoot: backupRoot,

		BackupsInProgress: map[string]map[string]context.CancelFunc{},
	}
}

func GetBackupsByPath(root string, full bool) ([]codegen.FolderBackup, error) {
	var backups []codegen.FolderBackup

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		metadataFilePath := filepath.Join(path, common.MetadataFileName)

		if _, err := os.Stat(metadataFilePath); err == nil {
			backup, err := LoadMetadata(path)
			if err != nil {
				logger.Error("failed to load metadata", zap.String("path", metadataFilePath), zap.Error(err))
				return fs.SkipDir
			}

			if full {
				size, count, err := utils.SizeAndCount(path, common.Throttling)
				if err != nil {
					logger.Info("failed to calculate the size and count", zap.String("path", path), zap.Error(err))
				}

				backup.BackupFolderCount = &count
				backup.BackupFolderSize = &size
			}

			backup.ClientFolderFileHashes = nil
			backup.ClientFolderFileSizes = nil

			backups = append(backups, *backup)
			return fs.SkipDir
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return backups, nil
}

func isBackupFile(filename string) bool {
	if filename == common.MetadataFileName {
		return true
	}

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
	// TODO - read from in-memory cache if it exists and motification time matches
	// TODO - read from the checksum file if it exists and motification time matches

	// TODO - calculate the hash of the file if it doesn't exist
	hash, err := XXHash(path)
	if err != nil {
		return "", err
	}

	// TODO - write the hash to the checksum file and touch it with the same modification time as the file
	// TODO - write to the in-memory cache along with the modification time

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

func BackupFile(path string, move bool) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("error accessing file: %w", err)
	}

	dir := filepath.Dir(path)
	filename := filepath.Base(path)
	filenameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	backupName := fmt.Sprintf(
		"%s-backup-%s%s",
		filenameWithoutExt,
		time.Now().Format("2006-01-02-15-04-05-000"),
		filepath.Ext(filename),
	)

	backupPath := filepath.Join(dir, backupName)

	if move {
		if err := os.Rename(path, backupPath); err != nil {
			return "", fmt.Errorf("error renaming file: %w", err)
		}
	} else {
		if err := copyFile(path, backupPath); err != nil {
			return "", fmt.Errorf("error copying file: %w", err)
		}
	}

	return backupPath, nil
}

func copyFile(src, dst string) error {
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

	return nil
}

func Normalize(path string) string {
	// Check for a drive letter (e.g., "C:")
	if len(path) > 2 && path[1] == ':' && ('a' <= path[0] && path[0] <= 'z' || 'A' <= path[0] && path[0] <= 'Z') {
		driveLetter := string(path[0])
		remainingPath := path[2:]
		unixPath := strings.ReplaceAll(remainingPath, `\`, `/`)
		return driveLetter + unixPath // Combine driveLetter and unixPath with a forward slash
	}
	return strings.ReplaceAll(path, `\`, `/`)
}

func SaveMetadata(backup *codegen.FolderBackup) error {
	if backup.BackupFolderPath == nil {
		return fmt.Errorf("backup folder path is not set")
	}

	backupFolderFullPath := filepath.Join(config.AppInfo.DataRootPath, *backup.BackupFolderPath)

	if err := os.MkdirAll(backupFolderFullPath, 0o755); err != nil {
		return err
	}

	metadataFilePath := filepath.Join(backupFolderFullPath, common.MetadataFileName)
	metadataFile, err := os.Create(metadataFilePath)
	if err != nil {
		return err
	}
	defer metadataFile.Close()

	encoder := json.NewEncoder(metadataFile)
	encoder.SetIndent("", "  ")

	return encoder.Encode(backup)
}

func LoadMetadata(path string) (*codegen.FolderBackup, error) {
	metadataFilePath := filepath.Join(path, common.MetadataFileName)

	metadataFile, err := os.Open(metadataFilePath)
	if err != nil {
		return nil, err
	}
	defer metadataFile.Close()

	decoder := json.NewDecoder(metadataFile)
	var backup codegen.FolderBackup
	if err := decoder.Decode(&backup); err != nil {
		return nil, err
	}

	return &backup, nil
}
