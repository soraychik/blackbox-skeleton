package main

import (
	"blackbox-scheduler/internal/database"
	"blackbox-scheduler/internal/fileprocessor"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

func main() {
	log.Println("Starting BlackBox Scheduler with Improved Storage Architecture...")

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

	// Определяем настройки хранения
	useMinIO := os.Getenv("USE_MINIO") == "true"
	diffThreshold := getEnvFloat("DIFF_THRESHOLD", 0.1)

	log.Printf("Storage configuration: MinIO=%t, DiffThreshold=%.2f", useMinIO, diffThreshold)

	// Создаём улучшенный процессор файлов
	processor, err := fileprocessor.NewImprovedFileProcessor(useMinIO, diffThreshold)
	if err != nil {
		log.Fatalf("Failed to create improved file processor: %v", err)
	}
	defer processor.Close()

	// Сразу обрабатываем файлы при запуске
	log.Println("Performing initial file scan...")
	processFilesImproved(db, processor)

	// Бесконечный цикл с периодической проверкой каждые 30 секунд
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Scheduler started. Monitoring for new config files every 30 seconds...")

	for range ticker.C {
		log.Println("Checking for new config files...")
		processFilesImproved(db, processor)
		log.Println("File processing cycle completed")
	}
}

// waitForMySQL ждёт пока MySQL станет доступен
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

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

// processFilesImproved обрабатывает все файлы с улучшенной архитектурой
func processFilesImproved(db *database.DB, processor *fileprocessor.ImprovedFileProcessor) {
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
		if err := processSingleFileImproved(db, processor, filePath); err != nil {
			log.Printf("Error processing file %s: %v", filePath, err)
		}
	}
}

// processSingleFileImproved обрабатывает один файл с улучшенной архитектурой
func processSingleFileImproved(db *database.DB, processor *fileprocessor.ImprovedFileProcessor, filePath string) error {
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

	_, err = processor.SaveVersion(context.Background(), db, fileInfo, device.ID)
	if err != nil {
		return err
	}

	log.Printf("Successfully processed version for %s", fileInfo.Name)
	return nil
}
