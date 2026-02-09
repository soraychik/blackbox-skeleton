package fileprocessor

import (
	"blackbox-scheduler/internal/models"
	"blackbox-scheduler/internal/storage"
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"blackbox-scheduler/internal/database"
)

type ImprovedFileProcessor struct {
	diffEngine    *storage.DiffEngine
	minioClient   *storage.MinIOImprovedClient
	useMinIO      bool
	diffThreshold float64
}

func NewImprovedFileProcessor(useMinIO bool, diffThreshold float64) (*ImprovedFileProcessor, error) {
	var minioClient *storage.MinIOImprovedClient
	var err error

	if useMinIO {
		minioClient, err = storage.NewMinIOImprovedClient()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
		}
	}

	return &ImprovedFileProcessor{
		diffEngine:    storage.NewDiffEngine(),
		minioClient:   minioClient,
		useMinIO:      useMinIO,
		diffThreshold: diffThreshold,
	}, nil
}

func (ifp *ImprovedFileProcessor) ProcessFile(filePath string) (*models.FileInfo, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	hash := ifp.calculateHash(content)
	deviceName := fileInfo.Name()

	return &models.FileInfo{
		Name:    deviceName,
		Path:    filePath,
		Size:    fileInfo.Size(),
		ModTime: fileInfo.ModTime(),
		Content: content,
		Hash:    hash,
	}, nil
}

func (ifp *ImprovedFileProcessor) calculateHash(content []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(content))
}

func (ifp *ImprovedFileProcessor) SaveVersion(
	ctx context.Context,
	db *database.DB,
	fileInfo *models.FileInfo,
	deviceID int,
) (*models.ConfigVersion, error) {
	latestVersion, err := db.GetLatestVersion(deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}

	var version *models.ConfigVersion

	if latestVersion == nil || latestVersion.FileHash != fileInfo.Hash {
		log.Printf("Processing new version for device %s (hash: %s)", fileInfo.Name, fileInfo.Hash[:8])

		if latestVersion != nil && ifp.useMinIO {
			// Пробуем использовать diff storage
			version, err = ifp.saveDiffVersion(ctx, db, fileInfo, deviceID, latestVersion)
			if err != nil {
				log.Printf("Failed to save diff version, falling back to full storage: %v", err)
				version, err = ifp.saveFullVersion(ctx, db, fileInfo, deviceID, "full")
				if err != nil {
					return nil, fmt.Errorf("failed to save full version as fallback: %w", err)
				}
			}
		} else {
			// Первый версии или MinIO отключен - сохраняем полный файл
			version, err = ifp.saveFullVersion(ctx, db, fileInfo, deviceID, "base")
			if err != nil {
				return nil, fmt.Errorf("failed to save full version: %w", err)
			}
		}

		log.Printf("Successfully saved version %d for device %s (storage: %s)",
			version.ID, fileInfo.Name, version.StorageType)
	} else {
		log.Printf("No changes detected for %s, skipping save", fileInfo.Name)
		version = latestVersion
	}

	return version, nil
}

func (ifp *ImprovedFileProcessor) saveFullVersion(
	ctx context.Context,
	db *database.DB,
	fileInfo *models.FileInfo,
	deviceID int,
	storageType string,
) (*models.ConfigVersion, error) {
	var minioObjectName string
	var originalSize, compressedSize int64

	if ifp.useMinIO {
		// Сохраняем в MinIO
		versionID := time.Now().UnixNano() // временный ID
		objectName, origSize, compSize, err := ifp.minioClient.UploadFullConfig(ctx, deviceID, int(versionID), fileInfo.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to upload full config to MinIO: %w", err)
		}
		minioObjectName = objectName
		originalSize = origSize
		compressedSize = compSize
	} else {
		// Сохраняем в файловую систему (legacy)
		archiveBasePath := "/app/archived_configs"
		now := time.Now()
		archivePath := filepath.Join(
			archiveBasePath,
			fmt.Sprintf("%d", deviceID),
			now.Format("2006"),
			now.Format("01"),
			now.Format("02"),
			fmt.Sprintf("%s.txt", fileInfo.Hash),
		)

		if err := os.MkdirAll(filepath.Dir(archivePath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create archive directories: %w", err)
		}

		if err := os.WriteFile(archivePath, fileInfo.Content, 0644); err != nil {
			return nil, fmt.Errorf("failed to write archive file: %w", err)
		}
		minioObjectName = archivePath
		originalSize = int64(len(fileInfo.Content))
		compressedSize = 0
	}

	// Сохраняем в базу данных
	versionDate := fileInfo.ModTime
	if versionDate.IsZero() {
		versionDate = time.Now()
	}

	version, err := db.SaveFullVersion(deviceID, minioObjectName, fileInfo.Hash, versionDate, storageType, originalSize, compressedSize)
	if err != nil {
		return nil, fmt.Errorf("failed to save version to database: %w", err)
	}

	if ifp.useMinIO {
		log.Printf("Full version stored in MinIO: %s (compression: %d -> %d bytes, %.1f%% reduction)",
			minioObjectName, originalSize, compressedSize,
			float64(originalSize-compressedSize)/float64(originalSize)*100)
	} else {
		log.Printf("Full version stored in filesystem: %s", minioObjectName)
	}

	return version, nil
}

func (ifp *ImprovedFileProcessor) saveDiffVersion(
	ctx context.Context,
	db *database.DB,
	fileInfo *models.FileInfo,
	deviceID int,
	latestVersion *models.ConfigVersion,
) (*models.ConfigVersion, error) {
	// Получаем контент базовой версии
	baseContent, err := ifp.getVersionContent(ctx, latestVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get base version content: %w", err)
	}

	// Проверяем, стоит ли использовать diff
	stats := ifp.diffEngine.GetStats(baseContent, fileInfo.Content)
	useDiff := stats["should_use_diff"].(bool)

	if !useDiff {
		return nil, fmt.Errorf("diff not efficient (savings: %.1f%%)", stats["savings_percent"])
	}

	// Создаем diff
	patch, err := ifp.diffEngine.CreateDiff(baseContent, fileInfo.Content, latestVersion.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create diff: %w", err)
	}

	// Сохраняем diff в MinIO
	compressedPatch, err := ifp.diffEngine.CompressPatch(patch)
	if err != nil {
		return nil, fmt.Errorf("failed to compress patch: %w", err)
	}

	diffObjectName, err := ifp.minioClient.UploadDiff(ctx, deviceID, 0, latestVersion.ID, compressedPatch)
	if err != nil {
		return nil, fmt.Errorf("failed to upload diff to MinIO: %w", err)
	}

	// Сохраняем информацию о версии в БД
	versionDate := fileInfo.ModTime
	if versionDate.IsZero() {
		versionDate = time.Now()
	}

	version, err := db.SaveDiffVersion(deviceID, diffObjectName, fileInfo.Hash, versionDate,
		latestVersion.ID, int64(len(compressedPatch)))
	if err != nil {
		return nil, fmt.Errorf("failed to save diff version to database: %w", err)
	}

	log.Printf("Diff version stored: %s -> %s (savings: %.1f%%, diff size: %d bytes)",
		latestVersion.FilePath, diffObjectName, stats["savings_percent"], len(compressedPatch))

	return version, nil
}

func (ifp *ImprovedFileProcessor) getVersionContent(ctx context.Context, version *models.ConfigVersion) ([]byte, error) {
	if ifp.useMinIO && version.MinioObjectName != "" {
		return ifp.minioClient.DownloadConfig(ctx, version.MinioObjectName)
	}

	// Fallback to filesystem
	if _, err := os.Stat(version.FilePath); err == nil {
		return os.ReadFile(version.FilePath)
	}

	return nil, fmt.Errorf("unable to retrieve version content")
}

func (ifp *ImprovedFileProcessor) ReconstructVersion(
	ctx context.Context,
	db *database.DB,
	version *models.ConfigVersion,
) ([]byte, error) {
	if version.StorageType == "full" || version.StorageType == "base" {
		return ifp.getVersionContent(ctx, version)
	}

	if version.StorageType == "diff" && version.ParentVersionID != nil {
		// Рекурсивно восстанавливаем базовую версию
		parentVersion, err := db.GetVersionByID(*version.ParentVersionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent version: %w", err)
		}

		baseContent, err := ifp.ReconstructVersion(ctx, db, parentVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to reconstruct parent version: %w", err)
		}

		// Получаем patch
		patchData, err := ifp.getVersionContent(ctx, version)
		if err != nil {
			return nil, fmt.Errorf("failed to get patch data: %w", err)
		}

		patch, err := ifp.diffEngine.DecompressPatch(patchData)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress patch: %w", err)
		}

		// Применяем patch
		return ifp.diffEngine.ApplyDiff(baseContent, patch)
	}

	return nil, fmt.Errorf("unknown storage type: %s", version.StorageType)
}

func (ifp *ImprovedFileProcessor) GetFilesInDirectory(dirPath string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(dirPath, entry.Name()))
		}
	}

	return files, nil
}

func (ifp *ImprovedFileProcessor) Close() error {
	// Закрываем соединения если нужно
	return nil
}
