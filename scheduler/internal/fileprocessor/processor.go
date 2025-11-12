package fileprocessor

import (
	"blackbox-scheduler/internal/models"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileProcessor struct {
	archiveBasePath string
}

func NewFileProcessor(archiveBasePath string) *FileProcessor {
	return &FileProcessor{
		archiveBasePath: archiveBasePath,
	}
}

// ProcessFile обрабатывает файл конфига
func (fp *FileProcessor) ProcessFile(filePath string) (*models.FileInfo, error) {
	// Читаем информацию о файле
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	// Читаем содержимое файла
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Вычисляем хэш содержимого
	hash := fp.calculateHash(content)

	// Извлекаем имя устройства из имени файла (без расширения)
	deviceName := strings.TrimSuffix(fileInfo.Name(), filepath.Ext(fileInfo.Name()))

	return &models.FileInfo{
		Name:    deviceName,
		Path:    filePath,
		Size:    fileInfo.Size(),
		ModTime: fileInfo.ModTime(),
		Content: content,
		Hash:    hash,
	}, nil
}

// calculateHash вычисляет SHA256 хэш содержимого
func (fp *FileProcessor) calculateHash(content []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(content))
}

// SaveToArchive сохраняет файл в архивную структуру
func (fp *FileProcessor) SaveToArchive(fileInfo *models.FileInfo) (string, error) {
	// Создаём путь: archived_configs/{device}/{year-month}/{day}.conf
	now := time.Now()
	archivePath := filepath.Join(
		fp.archiveBasePath,
		fileInfo.Name,
		now.Format("2006-01"), // 2024-11
		now.Format("02.conf"), // 12.conf
	)

	// Создаём все необходимые директории
	if err := os.MkdirAll(filepath.Dir(archivePath), 0755); err != nil {
		return "", fmt.Errorf("failed to create archive directories: %v", err)
	}

	// Сохраняем файл в архив
	if err := os.WriteFile(archivePath, fileInfo.Content, 0644); err != nil {
		return "", fmt.Errorf("failed to write archive file: %v", err)
	}

	log.Printf("File archived: %s -> %s", fileInfo.Name, archivePath)
	return archivePath, nil
}

// GetFilesInDirectory возвращает список файлов в директории
func (fp *FileProcessor) GetFilesInDirectory(dirPath string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".conf") {
			files = append(files, filepath.Join(dirPath, entry.Name()))
		}
	}

	return files, nil
}
