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
	log.Println("launching blackbox scheduler with support for multiple file servers...")

	// Ждём пока MySQL запустится
	if err := waitForMySQL(); err != nil {
		log.Fatalf("unable to wait for mysql to start: %v", err)
	}

	// Подключаемся к БД
	db, err := database.NewDB()
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
	}
	defer db.Close()

	// Определяем настройки хранения
	useMinIO := os.Getenv("USE_MINIO") == "true"
	diffThreshold := getEnvFloat("DIFF_THRESHOLD", 0.1)

	log.Printf("storage configuration: minio=%t, thresholdDiff=%.2f", useMinIO, diffThreshold)

	// Создаём улучшенный процессор файлов
	processor, err := fileprocessor.NewImprovedFileProcessor(useMinIO, diffThreshold)
	if err != nil {
		log.Fatalf("failed to create enhanced file processor: %v", err)
	}
	defer processor.Close()

	// Создаем менеджер файловых серверов
	serverManager := fileserver.NewFileServerManager(processor, db)

	// Загружаем конфигурации серверов
	if err := serverManager.LoadServers(); err != nil {
		log.Fatalf("failed to load file server configurations: %v", err)
	}

	// Монтируем все серверы
	if err := serverManager.MountAllServers(); err != nil {
		log.Printf("warning: failed to mount some file servers: %v", err)
		log.Println("continuing to work with running servers...")
	}
	defer serverManager.Close()

	// Интервал между сканированиями из .env (секунды); отсчёт — после завершения полного сканирования
	scanIntervalSec := getEnvInt("SCAN_INTERVAL_SECONDS", 30)
	if scanIntervalSec < 5 {
		scanIntervalSec = 5
	}
	scanInterval := time.Duration(scanIntervalSec) * time.Second
	log.Printf("interval between scans: %v (count after full scan completes)", scanInterval)

	// Канал для принудительного запуска сканирования (с дашборда)
	triggerScan := make(chan struct{}, 1)
	triggerPort := getEnv("SCHEDULER_TRIGGER_PORT", "9090")
	go runTriggerServer(":"+triggerPort, triggerScan)
	log.Printf("force scan server is listening on port %s", triggerPort)

	// Сразу обрабатываем все файлы при запуске; следующее сканирование — через scanInterval после завершения
	log.Println("performing an initial file scan...")
	serverManager.ProcessAllServers()
	log.Println("initial scan complete, waiting for next scan...")

	// Таймер следующего сканирования (сбрасывается после каждого полного сканирования и после принудительного)
	nextScanTimer := time.NewTimer(scanInterval)
	defer nextScanTimer.Stop()

	healthTicker := time.NewTicker(60 * time.Second)
	defer healthTicker.Stop()

	log.Println("scheduler started")

	for {
		select {
		case <-nextScanTimer.C:
			log.Println("checking for new configuration files...")
			serverManager.ProcessAllServers()
			log.Println("file processing cycle completed, waiting until next scan...")
			nextScanTimer.Reset(scanInterval)
		case <-triggerScan:
			log.Println("forced scanning on demand")
			serverManager.ProcessAllServers()
			log.Println("forced scan complete, waiting until next automatic scan...")
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
		log.Printf("forced scan server error: %v", err)
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
	log.Println("waiting for mysql to be ready...")

	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		db, err := database.NewDB()
		if err == nil {
			db.Close()
			log.Println("mysql is ready")
			return nil
		}

		log.Printf("attempt %d/%d: mysql is not ready yet, retrying in 2 seconds...", i+1, maxAttempts)
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("mysql was not ready after %d attempts", maxAttempts)
}
