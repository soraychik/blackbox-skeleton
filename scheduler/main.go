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
	"sync"
	"time"
)

var (
	scanStatusMu       sync.RWMutex
	scanInProgress     bool
	scanLastStartedAt  time.Time
	scanLastFinishedAt time.Time
)

func setScanStarted() {
	scanStatusMu.Lock()
	defer scanStatusMu.Unlock()
	scanInProgress = true
	scanLastStartedAt = time.Now()
}

func setScanFinished() {
	scanStatusMu.Lock()
	defer scanStatusMu.Unlock()
	scanInProgress = false
	scanLastFinishedAt = time.Now()
}

func getScanStatus() (bool, time.Time, time.Time) {
	scanStatusMu.RLock()
	defer scanStatusMu.RUnlock()
	return scanInProgress, scanLastStartedAt, scanLastFinishedAt
}

func runScan(serverManager *fileserver.FileServerManager) {
	setScanStarted()
	serverManager.ProcessAllServers()
	setScanFinished()
}

func loadScanInterval(db *database.DB, fallbackSec int) time.Duration {
	if cfg, err := db.GetConfigSourceSettings(); err == nil && cfg.ScanIntervalSeconds >= 5 {
		return time.Duration(cfg.ScanIntervalSeconds) * time.Second
	}
	if fallbackSec < 5 {
		fallbackSec = 5
	}
	return time.Duration(fallbackSec) * time.Second
}

func main() {
	log.Println("launching blackbox scheduler...")

	if err := waitForMySQL(); err != nil {
		log.Fatalf("unable to wait for mysql to start: %v", err)
	}

	db, err := database.NewDB()
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
	}
	defer db.Close()

	useMinIO := os.Getenv("USE_MINIO") == "true"
	diffThreshold := getEnvFloat("DIFF_THRESHOLD", 0.1)
	if cfg, err := db.GetConfigSourceSettings(); err == nil && cfg.DiffThreshold > 0 {
		diffThreshold = cfg.DiffThreshold
	}
	log.Printf("storage: minio=%t, thresholdDiff=%.2f", useMinIO, diffThreshold)

	processor, err := fileprocessor.NewImprovedFileProcessor(useMinIO, diffThreshold)
	if err != nil {
		log.Fatalf("failed to create file processor: %v", err)
	}
	defer processor.Close()

	serverManager := fileserver.NewFileServerManager(processor, db)
	if err := serverManager.LoadServers(); err != nil {
		log.Fatalf("failed to load file server configurations: %v", err)
	}
	defer serverManager.Close()

	triggerScan := make(chan struct{}, 1)
	triggerReload := make(chan struct{}, 1)
	triggerPort := getEnv("SCHEDULER_TRIGGER_PORT", "9090")
	go runTriggerServer(":"+triggerPort, triggerScan, triggerReload)
	log.Printf("trigger server listening on port %s", triggerPort)

	// Применяем настройки источника из DB при старте (без сканирования)
	if err := serverManager.ReloadConfigSource(db); err != nil {
		log.Printf("warning: failed to load config source from DB: %v", err)
	}

	scanInterval := loadScanInterval(db, getEnvInt("SCAN_INTERVAL_SECONDS", 300))
	log.Printf("scan interval: %v", scanInterval)

	healthTicker := time.NewTicker(60 * time.Second)
	defer healthTicker.Stop()

	// timerChan == nil до первого ручного запуска сканирования.
	// nil-канал в select блокируется навсегда.
	var timerChan <-chan time.Time

	log.Println("scheduler ready, waiting for first scan trigger")

	for {
		select {
		case <-timerChan:
			log.Println("periodic scan started")
			runScan(serverManager)
			log.Println("periodic scan complete")
			timerChan = time.After(scanInterval)
		case <-triggerScan:
			log.Println("manual scan triggered")
			runScan(serverManager)
			log.Println("scan complete, periodic scanning active")
			timerChan = time.After(scanInterval)
		case <-triggerReload:
			log.Println("reloading config source...")
			if err := serverManager.ReloadConfigSource(db); err != nil {
				log.Printf("reload failed: %v", err)
			} else {
				log.Println("config source reloaded")
			}
			scanInterval = loadScanInterval(db, getEnvInt("SCAN_INTERVAL_SECONDS", 300))
			log.Printf("scan interval: %v", scanInterval)
		case <-healthTicker.C:
			serverManager.CheckHealth()
		}
	}
}

// runTriggerServer поднимает HTTP-сервер для принудительного запуска сканирования и перезагрузки настроек.
func runTriggerServer(addr string, triggerChan chan<- struct{}, reloadChan chan<- struct{}) {
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
	http.HandleFunc("/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		select {
		case reloadChan <- struct{}{}:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","message":"reload triggered"}`))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","message":"reload already queued"}`))
		}
	})
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		inProgress, lastStartedAt, lastFinishedAt := getScanStatus()
		lastStarted := ""
		if !lastStartedAt.IsZero() {
			lastStarted = lastStartedAt.UTC().Format(time.RFC3339)
		}
		lastFinished := ""
		if !lastFinishedAt.IsZero() {
			lastFinished = lastFinishedAt.UTC().Format(time.RFC3339)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"in_progress":%t,"last_started_at":"%s","last_finished_at":"%s"}`, inProgress, lastStarted, lastFinished)))
	})
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Printf("trigger server error: %v", err)
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
		log.Printf("attempt %d/%d: mysql not ready, retrying in 2s...", i+1, maxAttempts)
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("mysql was not ready after %d attempts", maxAttempts)
}
