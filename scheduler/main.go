package main

import (
	"blackbox-scheduler/internal/database"
	"blackbox-scheduler/internal/fileprocessor"
	"fmt"
	"log"
	"time"
)

func main() {
	log.Println("Starting BlackBox Scheduler...")

	// Ждём пока MySQL запустится
	if err := waitForMySQL(); err != nil {
		log.Fatalf("Failed to wait for MySQL: %v", err)
	}

	// Подключаемся к БД
	db, err := database.NewDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Создаём процессор файлов
	processor := fileprocessor.NewFileProcessor("/app/archived_configs")

	// Сразу обрабатываем файлы при запуске
	log.Println("Performing initial file scan...")
	processFiles(db, processor)

	// Бесконечный цикл с периодической проверкой каждые 30 секунд
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Scheduler started. Monitoring for new config files every 30 seconds...")

	for range ticker.C {
		log.Println("Checking for new config files...")
		processFiles(db, processor)
		log.Println("File processing cycle completed")
	}
}

// waitForMySQL ждёт пока MySQL станет доступен
func waitForMySQL() error {
	log.Println("Waiting for MySQL to be ready...")

	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		db, err := database.NewDB()
		if err == nil {
			db.Close()
			log.Println("MySQL is ready!")
			return nil
		}

		log.Printf("Attempt %d/%d: MySQL not ready yet, retrying in 2 seconds...", i+1, maxAttempts)
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("MySQL did not become ready after %d attempts", maxAttempts)
}

// processFiles обрабатывает все файлы в директории
func processFiles(db *database.DB, processor *fileprocessor.FileProcessor) {
	files, err := processor.GetFilesInDirectory("/app/configs")
	if err != nil {
		log.Printf("Error reading source directory: %v", err)
		return
	}

	if len(files) == 0 {
		log.Println("No config files found")
		return
	}

	log.Printf("Found %d config file(s)", len(files))

	for _, filePath := range files {
		if err := processSingleFile(db, processor, filePath); err != nil {
			log.Printf("Error processing file %s: %v", filePath, err)
		}
	}
}

// processSingleFile обрабатывает один файл конфига
func processSingleFile(db *database.DB, processor *fileprocessor.FileProcessor, filePath string) error {
	log.Printf("Processing file: %s", filePath)

	fileInfo, err := processor.ProcessFile(filePath)
	if err != nil {
		return err
	}

	log.Printf("File info: %s, size: %d bytes, hash: %s",
		fileInfo.Name, fileInfo.Size, fileInfo.Hash[:8])

	device, err := db.GetOrCreateDevice(fileInfo.Name)
	if err != nil {
		return err
	}

	latestVersion, err := db.GetLatestVersion(device.ID)
	if err != nil {
		return err
	}

	if latestVersion == nil || latestVersion.FileHash != fileInfo.Hash {
		log.Printf("New or changed config detected for %s", fileInfo.Name)

		archivePath, err := processor.SaveToArchive(fileInfo, device.ID)
		if err != nil {
			return err
		}

		if err := db.SaveVersion(device.ID, archivePath, fileInfo.Hash, fileInfo.ModTime); err != nil {
			return err
		}

		log.Printf("Successfully processed new version for %s", fileInfo.Name)
	} else {
		log.Printf("No changes detected for %s", fileInfo.Name)
	}

	return nil
}
