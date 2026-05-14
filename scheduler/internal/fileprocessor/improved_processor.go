package fileprocessor

import (
	"blackbox-scheduler/internal/database"
	"blackbox-scheduler/internal/models"
	"blackbox-scheduler/internal/storage"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const chainLengthThreshold = 30 // Вынести в настройки

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
			return nil, fmt.Errorf("failed to initialize minio client: %w", err)
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
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	normalizedContent := normalizeLineEndings(content)
	hash := ifp.calculateHash(normalizedContent)
	hostname := fileInfo.Name()

	return &models.FileInfo{
		Name:     fileInfo.Name(),
		Path:     filePath,
		Size:     fileInfo.Size(),
		ModTime:  fileInfo.ModTime(),
		Content:  normalizedContent,
		Hash:     hash,
		Hostname: hostname,
	}, nil
}

func normalizeLineEndings(content []byte) []byte {
	return bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
}

func (ifp *ImprovedFileProcessor) calculateHash(content []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(content))
}

func (ifp *ImprovedFileProcessor) SaveVersion(
	ctx context.Context,
	db *database.DB,
	fileInfo *models.FileInfo,
) (*models.ConfigVersion, error) {
	device, err := db.GetOrCreateDevice(fileInfo.Hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create device: %w", err)
	}
	fileInfo.DeviceID = device.ID

	latestVersion, err := db.GetLatestVersion(device.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}

	if latestVersion != nil && latestVersion.VersionHash == fileInfo.Hash {
		log.Printf("no changes detected for %s (hash: %s), skipping save", fileInfo.Name, fileInfo.Hash[:8])
		return latestVersion, nil
	}

	log.Printf("processing new version for device %s (hash: %s)", fileInfo.Hostname, fileInfo.Hash[:8])

	storageType, storagePath, originalSize, compressedSize, parentVersionID, chainBaseID, chainPosition, err :=
		ifp.determineStorageType(ctx, db, fileInfo, latestVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to determine storage type: %w", err)
	}

	version, err := db.SaveVersion(
		device.ID,
		fileInfo.Hash,
		storageType,
		storagePath,
		parentVersionID,
		chainBaseID,
		chainPosition,
		originalSize,
		compressedSize,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save version: %w", err)
	}

	log.Printf("successfully saved version %d for device %s (storage: %s, chain_pos: %d)",
		version.ID, fileInfo.Hostname, version.StorageType, version.ChainPosition)

	return version, nil
}

func (ifp *ImprovedFileProcessor) determineStorageType(
	ctx context.Context,
	db *database.DB,
	fileInfo *models.FileInfo,
	latestVersion *models.ConfigVersion,
) (storageType, storagePath string, originalSize, compressedSize uint32, parentVersionID, chainBaseID *int, chainPosition int, err error) {
	if latestVersion == nil {
		return ifp.saveBaseVersion(ctx, fileInfo, nil, 0)
	}

	if latestVersion.ChainPosition >= chainLengthThreshold-1 {
		log.Printf("chain position %d >= %d, starting new chain", latestVersion.ChainPosition, chainLengthThreshold)
		return ifp.saveBaseVersion(ctx, fileInfo, latestVersion, latestVersion.ID)
	}

	// Предыдущий контент = база + все диффы до latestVersion (восстановленный файл), не сырой патч
	baseContent, err := ifp.ReconstructVersion(ctx, db, latestVersion)
	if err != nil {
		return ifp.saveBaseVersion(ctx, fileInfo, latestVersion, latestVersion.ID)
	}

	stats := ifp.diffEngine.GetStats(baseContent, fileInfo.Content)
	shouldUseDiff := stats["should_use_diff"].(bool)
	savingsPercent := stats["savings_percent"].(float64)
	// Сравниваем с размером базы тот же размер, что показывается в UI и пишется в original_size — несжатый патч
	diffSizeUncompressed := stats["diff_size_uncompressed"].(int)

	// База цепочки: если размер патча (как в БД/таблице) >= размера базы — сохраняем как новую базу
	var chainBaseVersion *models.ConfigVersion
	if latestVersion.ChainBaseID != nil {
		chainBaseVersion, _ = db.GetVersionByID(*latestVersion.ChainBaseID)
	}
	if chainBaseVersion == nil {
		chainBaseVersion = latestVersion
	}
	baseOriginalSize := int(chainBaseVersion.OriginalSize)
	if baseOriginalSize == 0 {
		baseOriginalSize = len(baseContent)
	}
	if diffSizeUncompressed >= baseOriginalSize {
		log.Printf("diff size (uncompressed) %d >= base size %d, starting new chain", diffSizeUncompressed, baseOriginalSize)
		return ifp.saveBaseVersion(ctx, fileInfo, latestVersion, latestVersion.ID)
	}

	if !shouldUseDiff {
		log.Printf("diff savings %.1f%% below threshold, starting new chain", savingsPercent)
		return ifp.saveBaseVersion(ctx, fileInfo, latestVersion, latestVersion.ID)
	}

	return ifp.saveDiffVersion(ctx, fileInfo, latestVersion, baseContent)
}

func (ifp *ImprovedFileProcessor) saveBaseVersion(
	ctx context.Context,
	fileInfo *models.FileInfo,
	latestVersion *models.ConfigVersion,
	newChainBaseID int,
) (storageType, storagePath string, originalSize, compressedSize uint32, parentVersionID, chainBaseID *int, chainPosition int, err error) {
	var objectName string

	if ifp.useMinIO {
		var origSize int64
		var compSize int64
		objectName, origSize, compSize, err = ifp.minioClient.UploadFullConfig(ctx, fileInfo.DeviceID, fileInfo.Content)
		originalSize = uint32(origSize)
		compressedSize = uint32(compSize)
		if err != nil {
			return "", "", 0, 0, nil, nil, 0, fmt.Errorf("failed to upload to MinIO: %w", err)
		}
	} else {
		archivePath := fmt.Sprintf("/app/archived_configs/%d/%s.txt", fileInfo.DeviceID, fileInfo.Hash)
		if err := os.MkdirAll(filepath.Dir(archivePath), 0755); err != nil {
			return "", "", 0, 0, nil, nil, 0, fmt.Errorf("failed to create directories: %w", err)
		}
		if err := os.WriteFile(archivePath, fileInfo.Content, 0644); err != nil {
			return "", "", 0, 0, nil, nil, 0, fmt.Errorf("failed to write file: %w", err)
		}
		objectName = archivePath
		originalSize = uint32(len(fileInfo.Content))
		compressedSize = 0
	}

	var parentID, baseID *int
	if latestVersion != nil {
		parentID = &latestVersion.ID
	}
	if newChainBaseID > 0 {
		baseID = &newChainBaseID
	}

	return "base", objectName, originalSize, compressedSize, parentID, baseID, 0, nil
}

func (ifp *ImprovedFileProcessor) saveDiffVersion(
	ctx context.Context,
	fileInfo *models.FileInfo,
	latestVersion *models.ConfigVersion,
	previousFullContent []byte,
) (storageType, storagePath string, origSize, compSize uint32, parentVersionID, chainBaseID *int, chainPosition int, err error) {
	// previousFullContent = восстановленный контент (база + все диффы до latestVersion), храним только дельту
	patchContent, err := ifp.diffEngine.CreateDiff(previousFullContent, fileInfo.Content, latestVersion.ID)
	if err != nil {
		return "", "", 0, 0, nil, nil, 0, fmt.Errorf("failed to create diff: %w", err)
	}

	var objectName string

	if ifp.useMinIO {
		var o, c int64
		objectName, o, c, err = ifp.minioClient.UploadDiff(ctx, fileInfo.DeviceID, []byte(patchContent))
		origSize = uint32(o)
		compSize = uint32(c)
		if err != nil {
			return "", "", 0, 0, nil, nil, 0, fmt.Errorf("failed to upload diff to minio: %w", err)
		}
	} else {
		archivePath := fmt.Sprintf("/app/archived_diffs/%d/%s.patch", fileInfo.DeviceID, fileInfo.Hash)
		if err := os.MkdirAll(filepath.Dir(archivePath), 0755); err != nil {
			return "", "", 0, 0, nil, nil, 0, fmt.Errorf("failed to create directories: %w", err)
		}
		if err := os.WriteFile(archivePath, []byte(patchContent), 0644); err != nil {
			return "", "", 0, 0, nil, nil, 0, fmt.Errorf("failed to write file: %w", err)
		}
		objectName = archivePath
		origSize = uint32(len(patchContent))
		compSize = 0
	}

	parentID := latestVersion.ID
	var baseID *int
	if latestVersion.ChainBaseID != nil {
		baseID = latestVersion.ChainBaseID
	} else {
		baseID = &latestVersion.ID
	}

	return "diff", objectName, origSize, compSize, &parentID, baseID, latestVersion.ChainPosition + 1, nil
}

func (ifp *ImprovedFileProcessor) getVersionContent(ctx context.Context, version *models.ConfigVersion) ([]byte, error) {
	if ifp.useMinIO && version.StoragePath != "" {
		return ifp.minioClient.DownloadConfig(ctx, version.StoragePath)
	}

	if _, err := os.Stat(version.StoragePath); err == nil {
		return os.ReadFile(version.StoragePath)
	}

	return nil, fmt.Errorf("unable to retrieve version content")
}

func (ifp *ImprovedFileProcessor) ReconstructVersion(
	ctx context.Context,
	db *database.DB,
	version *models.ConfigVersion,
) ([]byte, error) {
	if version.StorageType == "base" {
		return ifp.getVersionContent(ctx, version)
	}

	if version.StorageType == "diff" && version.ParentVersionID != nil {
		parentVersion, err := db.GetVersionByID(*version.ParentVersionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent version: %w", err)
		}

		baseContent, err := ifp.ReconstructVersion(ctx, db, parentVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to reconstruct parent version: %w", err)
		}

		patchData, err := ifp.getVersionContent(ctx, version)
		if err != nil {
			return nil, fmt.Errorf("failed to get patch data: %w", err)
		}

		patchContent := string(patchData)
		if decompressed, err := ifp.diffEngine.DecompressPatch(patchData); err == nil {
			patchContent = decompressed
		}

		return ifp.diffEngine.ApplyDiff(baseContent, patchContent)
	}

	return nil, fmt.Errorf("unknown storage type: %s", version.StorageType)
}

func (ifp *ImprovedFileProcessor) GetFilesInDirectory(dirPath string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(dirPath, entry.Name()))
		}
	}

	return files, nil
}

func (ifp *ImprovedFileProcessor) Close() error {
	return nil
}
