package main

import (
	"blackbox-scheduler/internal/database"
	"blackbox-scheduler/internal/fileprocessor"
	"blackbox-scheduler/internal/fileserver"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

func main() {
	log.Println("Запуск BlackBox Scheduler с поддержкой множественных файловых серверов...")

	// Ждём пока MySQL запустится
	if err := waitForMySQL(); err != nil {
		log.Fatalf("Не удалось дождаться запуска MySQL: %v", err)
	}

	// Подключаемся к БД
	db, err := database.NewDB()
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	defer db.Close()

	// Определяем настройки хранения
	useMinIO := os.Getenv("USE_MINIO") == "true"
	diffThreshold := getEnvFloat("DIFF_THRESHOLD", 0.1)

	log.Printf("Конфигурация хранилища: MinIO=%t, ПорогDiff=%.2f", useMinIO, diffThreshold)

	// Создаём улучшенный процессор файлов
	processor, err := fileprocessor.NewImprovedFileProcessor(useMinIO, diffThreshold)
	if err != nil {
		log.Fatalf("Не удалось создать улучшенный файловый процессор: %v", err)
	}
	defer processor.Close()

	// Создаем менеджер файловых серверов
	serverManager := fileserver.NewFileServerManager(processor, db)

	// Загружаем конфигурации серверов
	if err := serverManager.LoadServers(); err != nil {
		log.Fatalf("Не удалось загрузить конфигурации файловых серверов: %v", err)
	}

	// Монтируем все серверы
	if err := serverManager.MountAllServers(); err != nil {
		log.Fatalf("Не удалось смонтировать файловые серверы: %v", err)
	}
	defer serverManager.Close()

	// Сразу обрабатываем файлы при запуске
	log.Println("Выполняем первоначальное сканирование файлов...")
	serverManager.ProcessAllServers()

	// Бесконечный цикл с периодической проверкой каждые 30 секунд
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Тикер для проверки состояния
	healthTicker := time.NewTicker(60 * time.Second)
	defer healthTicker.Stop()

	log.Println("Планировщик запущен. Мониторинг файловых серверов каждые 30 секунд...")

	for {
		select {
		case <-ticker.C:
			log.Println("Проверяем наличие новых конфигурационных файлов...")
			serverManager.ProcessAllServers()
			log.Println("Цикл обработки файлов завершен")
		case <-healthTicker.C:
			serverManager.CheckHealth()
		}
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
	log.Println("Ожидание готовности MySQL...")

	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		db, err := database.NewDB()
		if err == nil {
			db.Close()
			log.Println("MySQL готов к работе!")
			return nil
		}

		log.Printf("Попытка %d/%d: MySQL еще не готов, повторная попытка через 2 секунды...", i+1, maxAttempts)
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("MySQL не стал готов после %d попыток", maxAttempts)
}
