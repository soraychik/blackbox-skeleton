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

	// Извлекаем имя устройства из имени файла
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

// calculateHash вычисляет SHA256 хэш содержимого
func (fp *FileProcessor) calculateHash(content []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(content))
}

// SaveToArchive сохраняет файл в архивную структуру /configs/{device_id}/{yyyy}/{mm}/{dd}/{hash}.txt
func (fp *FileProcessor) SaveToArchive(fileInfo *models.FileInfo, deviceID int) (string, error) {
	// Создаём путь: /configs/{device_id}/{yyyy}/{mm}/{dd}/{hash}.txt
	now := time.Now()
	archivePath := filepath.Join(
		fp.archiveBasePath,
		fmt.Sprintf("%d", deviceID),          // device_id
		now.Format("2006"),                   // yyyy
		now.Format("01"),                     // mm
		now.Format("02"),                     // dd
		fmt.Sprintf("%s.txt", fileInfo.Hash), // hash.txt
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

// GetFilesInDirectory возвращает список всех файлов в директории
func (fp *FileProcessor) GetFilesInDirectory(dirPath string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			fileName := entry.Name()

			// Skip temporary and backup files
			if fp.shouldSkipFile(fileName) {
				log.Printf("Skipping temporary file: %s", fileName)
				continue
			}

			files = append(files, filepath.Join(dirPath, fileName))
		}
	}

	return files, nil
}

func (fp *FileProcessor) shouldSkipFile(fileName string) bool {
	// Skip vim swap files
	if strings.HasSuffix(fileName, ".swp") || strings.HasSuffix(fileName, ".swo") {
		return true
	}

	// Skip other temporary and backup files
	tempSuffixes := []string{".tmp", ".temp", ".bak", "~", ".lock"}
	for _, suffix := range tempSuffixes {
		if strings.HasSuffix(fileName, suffix) {
			return true
		}
	}

	// Skip hidden files (starting with dot)
	if strings.HasPrefix(fileName, ".") {
		return true
	}

	// Skip system files
	systemFiles := []string{"Thumbs.db", "desktop.ini", ".DS_Store"}
	for _, sysFile := range systemFiles {
		if fileName == sysFile {
			return true
		}
	}

	return false
}
