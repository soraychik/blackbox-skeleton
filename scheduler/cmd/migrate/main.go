package main

import (
	"blackbox-scheduler/internal/database"
	"blackbox-scheduler/internal/fileprocessor"
	"blackbox-scheduler/internal/storage"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	var dryRun = flag.Bool("dry-run", true, "Perform a dry run without making changes")
	var batchSize = flag.Int("batch-size", 10, "Number of versions to process in each batch")
	flag.Parse()

	log.Println("Starting migration to improved storage architecture...")
	log.Printf("Dry run: %t", *dryRun)

	// Подключаемся к БД
	db, err := database.NewDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Создаём MinIO клиент
	minioClient, err := storage.NewMinIOImprovedClient()
	if err != nil {
		log.Fatalf("Failed to create MinIO client: %v", err)
	}

	// Создаём процессор файлов (нужен только для совместимости)
	_ = fileprocessor.NewFileProcessor("/app/archived_configs")

	// Получаем все существующие версии
	versions, err := db.GetAllVersionsForMigration()
	if err != nil {
		log.Fatalf("Failed to get versions for migration: %v", err)
	}

	log.Printf("Found %d versions to migrate", len(versions))

	ctx := context.Background()
	migrated := 0
	errors := 0

	for i, version := range versions {
		log.Printf("Processing version %d/%d (ID: %d, Device: %d)", i+1, len(versions), version.ID, version.DeviceID)

		// Пропускаем если уже мигрировано
		if version.StorageType != "" && version.StorageType != "full" {
			log.Printf("Version %d already migrated, skipping", version.ID)
			continue
		}

		// Читаем файл
		content, err := readConfigFile(version.FilePath)
		if err != nil {
			log.Printf("Error reading file %s: %v", version.FilePath, err)
			errors++
			continue
		}

		// Загружаем в MinIO
		if !*dryRun {
			objectName, originalSize, compressedSize, err := minioClient.UploadFullConfig(ctx, version.DeviceID, version.ID, content)
			if err != nil {
				log.Printf("Error uploading to MinIO: %v", err)
				errors++
				continue
			}

			// Обновляем запись в БД
			err = db.UpdateVersionForMigration(version.ID, objectName, "full", int64(originalSize), int64(compressedSize))
			if err != nil {
				log.Printf("Error updating version in DB: %v", err)
				errors++
				continue
			}

			// Создаем запись в storage_snapshots
			err = db.CreateStorageSnapshot(version.DeviceID, version.ID, "full", objectName, int64(originalSize), int64(compressedSize))
			if err != nil {
				log.Printf("Error creating storage snapshot: %v", err)
				errors++
				continue
			}
		}

		migrated++
		log.Printf("Successfully migrated version %d", version.ID)

		// Пауза каждые batchSize записей
		if (i+1)%*batchSize == 0 {
			log.Printf("Processed %d versions, pausing for 2 seconds...", i+1)
			time.Sleep(2 * time.Second)
		}
	}

	log.Printf("Migration completed. Migrated: %d, Errors: %d", migrated, errors)
	if *dryRun {
		log.Println("This was a dry run. No changes were made.")
	}
}

func readConfigFile(filePath string) ([]byte, error) {
	// Проверяем если путь абсолютный
	if filepath.IsAbs(filePath) {
		if _, err := os.Stat(filePath); err == nil {
			return os.ReadFile(filePath)
		}
	}

	// Пробуем найти в archived_configs
	archivePath := filepath.Join("/app/archived_configs", filePath)
	if _, err := os.Stat(archivePath); err == nil {
		return os.ReadFile(archivePath)
	}

	// Пробуем извлечь хэш из имени файла и найти по структуре
	if len(filePath) > 4 && filepath.Ext(filePath) == ".txt" {
		hash := filepath.Base(filePath[:len(filePath)-4])
		searchPath := "/app/archived_configs"

		// Рекурсивный поиск
		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Name() == hash+".txt" {
				filePath = path
				return filepath.SkipAll
			}
			return nil
		})

		if err == nil && filePath != searchPath {
			return os.ReadFile(filePath)
		}
	}

	return nil, fmt.Errorf("file not found: %s", filePath)
}
