package main

import (
	"blackbox-scheduler/internal/database"
	"blackbox-scheduler/internal/fileprocessor"
	"blackbox-scheduler/internal/fileserver"
	"fmt"
	"log"
	"net/http"
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
		log.Printf("Предупреждение: не удалось смонтировать некоторые файловые серверы: %v", err)
		log.Println("Продолжаем работу с работающими серверами...")
	}
	defer serverManager.Close()

	// Интервал между сканированиями из .env (секунды); отсчёт — после завершения полного сканирования
	scanIntervalSec := getEnvInt("SCAN_INTERVAL_SECONDS", 30)
	if scanIntervalSec < 5 {
		scanIntervalSec = 5
	}
	scanInterval := time.Duration(scanIntervalSec) * time.Second
	log.Printf("Интервал между сканированиями: %v (отсчёт после завершения полного сканирования)", scanInterval)

	// Канал для принудительного запуска сканирования (с дашборда)
	triggerScan := make(chan struct{}, 1)
	triggerPort := getEnv("SCHEDULER_TRIGGER_PORT", "9090")
	go runTriggerServer(":"+triggerPort, triggerScan)
	log.Printf("Сервер принудительного сканирования слушает порт %s", triggerPort)

	// Сразу обрабатываем все файлы при запуске; следующее сканирование — через scanInterval после завершения
	log.Println("Выполняем первоначальное сканирование файлов...")
	serverManager.ProcessAllServers()
	log.Println("Первоначальное сканирование завершено. Ожидание до следующего сканирования...")

	// Таймер следующего сканирования (сбрасывается после каждого полного сканирования и после принудительного)
	nextScanTimer := time.NewTimer(scanInterval)
	defer nextScanTimer.Stop()

	healthTicker := time.NewTicker(60 * time.Second)
	defer healthTicker.Stop()

	log.Println("Планировщик запущен.")

	for {
		select {
		case <-nextScanTimer.C:
			log.Println("Проверяем наличие новых конфигурационных файлов...")
			serverManager.ProcessAllServers()
			log.Println("Цикл обработки файлов завершён. Ожидание до следующего сканирования...")
			nextScanTimer.Reset(scanInterval)
		case <-triggerScan:
			log.Println("Принудительное сканирование по запросу...")
			serverManager.ProcessAllServers()
			log.Println("Принудительное сканирование завершено. Ожидание до следующего автоматического сканирования...")
			nextScanTimer.Reset(scanInterval)
		case <-healthTicker.C:
			serverManager.CheckHealth()
		}
	}
}

// runTriggerServer поднимает HTTP-сервер для принудительного запуска сканирования (POST /scan).
func runTriggerServer(addr string, triggerChan chan<- struct{}) {
	http.HandleFunc("/scan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		select {
		case triggerChan <- struct{}{}:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","message":"scan triggered"}`))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","message":"scan already queued"}`))
		}
	})
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Printf("Ошибка сервера принудительного сканирования: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// waitForMySQL ждёт пока MySQL станет доступен
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
