package fileserver

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"blackbox-scheduler/internal/database"
	"blackbox-scheduler/internal/fileprocessor"
	"blackbox-scheduler/internal/remotefs"
)

const fileWorkers = 20

type FileServerConfig struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Server     string `json:"server"`
	SharePath  string `json:"sharePath"`
	ShareName  string `json:"shareName"`
	MountPoint string `json:"mountPoint"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Domain     string `json:"domain"`
	LocalPath  string `json:"localPath"`
	Enabled    bool   `json:"enabled"`
}

type FileServerManager struct {
	servers   map[string]*FileServerInstance
	processor *fileprocessor.ImprovedFileProcessor
	db        *database.DB
	mu        sync.RWMutex
}

type FileServerInstance struct {
	config    *FileServerConfig
	fs        *remotefs.RemoteFileSystem
	lastCheck time.Time
	isHealthy bool
}

func NewFileServerManager(processor *fileprocessor.ImprovedFileProcessor, db *database.DB) *FileServerManager {
	return &FileServerManager{
		servers:   make(map[string]*FileServerInstance),
		processor: processor,
		db:        db,
	}
}

func (fsm *FileServerManager) LoadServers() error {
	configs := getConfigsFromEnv()

	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	for id, config := range configs {
		if !config.Enabled {
			continue
		}
		instance, err := fsm.createServerInstance(id, config)
		if err != nil {
			log.Printf("failed to create server instance %s: %v", id, err)
			continue
		}
		fsm.servers[id] = instance
		if config.Type == "local" {
			log.Printf("file server added: %s (local folder: %s)", id, config.LocalPath)
		} else {
			log.Printf("file server added: %s (%s://%s)", id, config.Type, config.Server)
		}
	}

	if len(fsm.servers) == 0 {
		log.Printf("warning: no file sources are configured. scanning will not occur.")
	}
	return nil
}

func (fsm *FileServerManager) createServerInstance(id string, config *FileServerConfig) (*FileServerInstance, error) {
	if config.Type == "local" {
		return &FileServerInstance{
			config:    config,
			fs:        nil,
			lastCheck: time.Now(),
			isHealthy: true,
		}, nil
	}
	fs, err := remotefs.NewRemoteFileSystem()
	if err != nil {
		return nil, fmt.Errorf("failed to create remote file system: %w", err)
	}
	return &FileServerInstance{
		config:    config,
		fs:        fs,
		lastCheck: time.Now(),
		isHealthy: false,
	}, nil
}

func (fsm *FileServerManager) MountAllServers() error {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	var errors []string
	for id, instance := range fsm.servers {
		if instance.fs == nil {
			instance.isHealthy = true
			continue
		}
		if err := instance.fs.Mount(); err != nil {
			errors = append(errors, fmt.Sprintf("server %s: %v", id, err))
			instance.isHealthy = false
		} else {
			instance.isHealthy = true
			log.Printf("mounted server %s", id)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("mounting errors: %v", errors)
	}
	return nil
}

func (fsm *FileServerManager) ProcessAllServers() {
	fsm.mu.RLock()
	servers := make(map[string]*FileServerInstance)
	for k, v := range fsm.servers {
		servers[k] = v
	}
	fsm.mu.RUnlock()

	var wg sync.WaitGroup
	for id, instance := range servers {
		if !instance.isHealthy {
			log.Printf("skip unhealthy server: %s", id)
			continue
		}
		wg.Add(1)
		go func(serverID string, inst *FileServerInstance) {
			defer wg.Done()
			fsm.processServer(serverID, inst)
		}(id, instance)
	}
	wg.Wait()
}

// processServer обрабатывает файлы сервера в два этапа:
// 1. pre-check: один SELECT всей таблицы + os.Stat — без чтения контента файлов
// 2. параллельная обработка через семафор — только изменившихся файлов
func (fsm *FileServerManager) processServer(id string, instance *FileServerInstance) {
	var configDir string
	if instance.config.Type == "local" {
		configDir = instance.config.LocalPath
	} else {
		configDir = instance.fs.GetConfigDirectory()
	}

	files, err := fsm.processor.GetFilesInDirectory(configDir)
	if err != nil {
		log.Printf("error reading directory from server %s (path %q): %v", id, configDir, err)
		instance.isHealthy = false
		return
	}
	log.Printf("%d files found on server %s", len(files), id)

	// --- Этап 1: pre-check ---
	// Один SELECT вместо N — загружаем всё состояние в память,
	// потом сравниваем локально без обращений к БД.
	// При 20k файлов: было ~20k запросов (~30 сек), стало 1 запрос (~50 мс).
	fileStates, err := fsm.db.GetAllFileStates()
	if err != nil {
		log.Printf("GetAllFileStates failed for server %s: %v, processing all files", id, err)
		fileStates = map[string]*database.FileState{}
	}

	var candidates []string
	skipped := 0
	for _, filePath := range files {
		info, err := os.Stat(filePath)
		if err != nil {
			log.Printf("stat failed for %s: %v", filePath, err)
			continue
		}
		state, ok := fileStates[info.Name()]
		if ok && state.Size == info.Size() && state.ModTime.Equal(info.ModTime().Truncate(time.Millisecond)) {
			skipped++
			continue
		}
		candidates = append(candidates, filePath)
	}
	log.Printf("server %s: %d to process, %d skipped by mtime/size", id, len(candidates), skipped)

	if len(candidates) == 0 {
		instance.lastCheck = time.Now()
		return
	}

	// --- Этап 2: параллельная обработка + запись сразу ---
	sem := make(chan struct{}, fileWorkers)
	var wg sync.WaitGroup

	for _, filePath := range candidates {
		wg.Add(1)
		fp := filePath
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := fsm.processSingleFile(id, fp); err != nil {
				log.Printf("error processing file %s from server %s: %v", fp, id, err)
			}
		}()
	}

	wg.Wait()
	instance.lastCheck = time.Now()
}

// processSingleFile читает файл, проверяет hash, сохраняет версию и обновляет file state.
func (fsm *FileServerManager) processSingleFile(serverID, filePath string) error {
	fileInfo, err := fsm.processor.ProcessFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to process file: %w", err)
	}

	ctx := context.Background()
	_, err = fsm.processor.SaveVersion(ctx, fsm.db, fileInfo)
	if err != nil {
		return fmt.Errorf("failed to save version: %w", err)
	}

	// Обновляем file state чтобы следующий запуск пропустил этот файл по mtime/size
	if err := fsm.db.UpsertFileState(fileInfo.Hostname, fileInfo.Size, fileInfo.ModTime); err != nil {
		log.Printf("warning: failed to upsert file state for %s: %v", fileInfo.Hostname, err)
	}

	log.Printf("processed version for %s-%s", serverID, fileInfo.Hostname)
	return nil
}

func (fsm *FileServerManager) CheckHealth() {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	for id, instance := range fsm.servers {
		if instance.fs == nil {
			continue
		}
		healthy := true
		if err := instance.fs.CheckConnection(); err != nil {
			healthy = false
			log.Printf("health check failed for server %s: %v", id, err)
		}
		if !healthy && instance.isHealthy {
			log.Printf("server %s became unhealthy, attempting remount", id)
			if err := instance.fs.Mount(); err != nil {
				log.Printf("failed to remount server %s: %v", id, err)
			} else {
				healthy = true
			}
		}
		instance.isHealthy = healthy
	}
}

func (fsm *FileServerManager) GetStatus() map[string]interface{} {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	servers := make(map[string]interface{})
	for id, instance := range fsm.servers {
		servers[id] = map[string]interface{}{
			"type":      instance.config.Type,
			"server":    instance.config.Server,
			"enabled":   instance.config.Enabled,
			"healthy":   instance.isHealthy,
			"lastCheck": instance.lastCheck.Format(time.RFC3339),
		}
	}
	return map[string]interface{}{
		"servers":      servers,
		"totalServers": len(servers),
	}
}

func (fsm *FileServerManager) Close() error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	var errors []error
	for id, instance := range fsm.servers {
		if instance.fs != nil {
			if err := instance.fs.Unmount(); err != nil {
				errors = append(errors, fmt.Errorf("failed to unmount server %s: %w", id, err))
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("unmount errors: %v", errors)
	}
	return nil
}

func getConfigsFromEnv() map[string]*FileServerConfig {
	configs := make(map[string]*FileServerConfig)

	if getEnv("LOCAL_FS_ENABLED", "false") == "true" {
		localPath := getEnv("LOCAL_FS_PATH", "/app/configs")
		log.Printf("development mode: local storage %q", localPath)
		configs["local"] = &FileServerConfig{
			ID: "local", Type: "local", LocalPath: localPath, Enabled: true,
		}
		return configs
	}

	log.Println("production mode: remote file servers are used")

	if getEnv("FILE_SERVER_ENABLED", "true") == "true" {
		configs["server1"] = &FileServerConfig{
			ID:         "server1",
			Type:       getEnv("FILE_SERVER_TYPE", "nfs"),
			Server:     getEnv("NFS_SERVER", "192.168.70.149"),
			SharePath:  getEnv("NFS_SHARE_PATH", "/srv/share"),
			MountPoint: getEnv("NFS_MOUNT_POINT", "/mnt/nfs"),
			Username:   getEnv("SMB_USERNAME", "guest"),
			Password:   getEnv("SMB_PASSWORD", ""),
			Domain:     getEnv("SMB_DOMAIN", "WORKGROUP"),
			Enabled:    true,
		}
	}
	if getEnv("FILE_SERVER_2_ENABLED", "false") == "true" {
		configs["server2"] = &FileServerConfig{
			ID:         "server2",
			Type:       getEnv("FILE_SERVER_2_TYPE", "nfs"),
			Server:     getEnv("FILE_SERVER_2_SERVER", ""),
			SharePath:  getEnv("FILE_SERVER_2_SHARE_PATH", "/srv/share"),
			MountPoint: getEnv("FILE_SERVER_2_MOUNT_POINT", "/mnt/nfs2"),
			Enabled:    true,
		}
	}
	return configs
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ReloadConfigSource читает настройки источника из system_settings и применяет их.
// Вызывается каждые 30 секунд из main loop scheduler.
func (fsm *FileServerManager) ReloadConfigSource(db *database.DB) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	cfg, err := db.GetConfigSourceSettings()
	if err != nil {
		return fmt.Errorf("failed to get config source settings: %w", err)
	}

	switch cfg.Type {
	case "smb":
		// При переключении на SMB удаляем DB-local источник.
		if localInst, ok := fsm.servers["db-local"]; ok {
			if localInst.fs != nil {
				localInst.fs.Unmount()
			}
			delete(fsm.servers, "db-local")
		}

		smbServer := firstNonEmpty(cfg.SmbServer, os.Getenv("SMB_SERVER"))
		smbShare := firstNonEmpty(cfg.SmbShare, os.Getenv("SMB_SHARE_NAME"))
		smbUsername := firstNonEmpty(cfg.SmbUsername, os.Getenv("SMB_USERNAME"))
		smbPassword := firstNonEmpty(cfg.SmbPassword, os.Getenv("SMB_PASSWORD"))
		smbDomain := firstNonEmpty(cfg.SmbDomain, os.Getenv("SMB_DOMAIN"), "WORKGROUP")

		// Если SMB сервер уже настроен с теми же параметрами — пропускаем
		if inst, ok := fsm.servers["db-smb"]; ok {
			if inst.config.Server == smbServer &&
				inst.config.ShareName == smbShare &&
				inst.config.Username == smbUsername &&
				inst.config.Password == smbPassword &&
				inst.config.Domain == smbDomain {
				return nil
			}
			// Параметры изменились — размонтируем старый
			if inst.fs != nil {
				inst.fs.Unmount()
			}
			delete(fsm.servers, "db-smb")
		}

		// Передаём параметры через env — remotefs читает именно оттуда
		os.Setenv("FILE_SERVER_TYPE", "smb")
		os.Setenv("SMB_SERVER", smbServer)
		os.Setenv("SMB_SHARE_NAME", smbShare)
		os.Setenv("SMB_USERNAME", smbUsername)
		os.Setenv("SMB_PASSWORD", smbPassword)
		os.Setenv("SMB_DOMAIN", smbDomain)
		os.Setenv("SMB_MOUNT_POINT", "/mnt/smb-db")

		fs, err := remotefs.NewRemoteFileSystem()
		if err != nil {
			return fmt.Errorf("failed to create SMB filesystem: %w", err)
		}
		if err := fs.Mount(); err != nil {
			return fmt.Errorf("failed to mount SMB: %w", err)
		}
		fsm.servers["db-smb"] = &FileServerInstance{
			config: &FileServerConfig{
				ID:        "db-smb",
				Type:      "smb",
				Server:    smbServer,
				ShareName: smbShare,
				Username:  smbUsername,
				Password:  smbPassword,
				Domain:    smbDomain,
			},
			fs:        fs,
			lastCheck: time.Now(),
			isHealthy: true,
		}
		log.Printf("SMB server configured from DB: //%s/%s", smbServer, smbShare)

	case "local":
		localPath := firstNonEmpty(cfg.Path, os.Getenv("LOCAL_FS_PATH"), "/app/configs")

		// При переключении на local удаляем DB-SMB источник, чтобы не было двойного сканирования.
		if smbInst, ok := fsm.servers["db-smb"]; ok {
			if smbInst.fs != nil {
				smbInst.fs.Unmount()
			}
			delete(fsm.servers, "db-smb")
		}

		if localInst, ok := fsm.servers["db-local"]; ok {
			oldPath := localInst.config.LocalPath
			localInst.config.Type = "local"
			localInst.config.LocalPath = localPath
			localInst.config.Enabled = true
			localInst.isHealthy = true
			log.Printf("updated local config source path: %s -> %s", oldPath, localPath)
		} else {
			fsm.servers["db-local"] = &FileServerInstance{
				config: &FileServerConfig{
					ID:        "db-local",
					Type:      "local",
					LocalPath: localPath,
					Enabled:   true,
				},
				fs:        nil,
				lastCheck: time.Now(),
				isHealthy: true,
			}
			log.Printf("local config source configured from DB: %s", localPath)
		}
	}

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}