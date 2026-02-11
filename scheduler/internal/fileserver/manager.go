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

type FileServerConfig struct {
	ID         string `json:"id"`
	Type       string `json:"type"`       // nfs, smb, local
	Server     string `json:"server"`     // IP или имя хоста
	SharePath  string `json:"sharePath"`  // Для NFS
	ShareName  string `json:"shareName"`  // Для SMB
	MountPoint string `json:"mountPoint"` // Локальный путь монтирования
	Username   string `json:"username"`   // Для SMB
	Password   string `json:"password"`   // Для SMB
	Domain     string `json:"domain"`     // Для SMB
	LocalPath  string `json:"localPath"`  // Для локальной ФС
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

// LoadServers загружает и инициализирует все настроенные файловые серверы
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
			log.Printf("Не удалось создать экземпляр сервера %s: %v", id, err)
			continue
		}

		fsm.servers[id] = instance
		log.Printf("Добавлен файловый сервер: %s (%s://%s)", id, config.Type, config.Server)
	}

	return nil
}

// createServerInstance создает экземпляр файлового сервера
func (fsm *FileServerManager) createServerInstance(id string, config *FileServerConfig) (*FileServerInstance, error) {
	var fs *remotefs.RemoteFileSystem
	var err error

	if config.Type == "local" {
		// Для локальной файловой системы нам не нужен remotefs
		return &FileServerInstance{
			config:    config,
			fs:        nil,
			lastCheck: time.Now(),
			isHealthy: true,
		}, nil
	}

	// Для удаленных файловых систем создаем экземпляр remotefs
	// Пока используем существующий NewRemoteFileSystem который читает из env
	// В будущем можно расширить для принятия конфигурации
	fs, err = remotefs.NewRemoteFileSystem()
	if err != nil {
		return nil, fmt.Errorf("не удалось создать удаленную файловую систему: %w", err)
	}

	return &FileServerInstance{
		config:    config,
		fs:        fs,
		lastCheck: time.Now(),
		isHealthy: false,
	}, nil
}

// MountAllServers монтирует все файловые серверы
func (fsm *FileServerManager) MountAllServers() error {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	var errors []string

	for id, instance := range fsm.servers {
		if instance.fs == nil {
			// Локальная файловая система, монтирование не требуется
			instance.isHealthy = true
			continue
		}

		if err := instance.fs.Mount(); err != nil {
			errors = append(errors, fmt.Sprintf("Сервер %s: %v", id, err))
			instance.isHealthy = false
		} else {
			instance.isHealthy = true
			log.Printf("Успешно смонтирован сервер %s", id)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("ошибки монтирования: %v", errors)
	}

	return nil
}

// ProcessAllServers обрабатывает все серверы параллельно
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
			log.Printf("Пропуск нездорового сервера: %s", id)
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

// processServer обрабатывает файлы на одном сервере
func (fsm *FileServerManager) processServer(id string, instance *FileServerInstance) {
	var configDir string

	if instance.config.Type == "local" {
		configDir = instance.config.LocalPath
	} else {
		configDir = instance.fs.GetConfigDirectory()
	}

	files, err := fsm.processor.GetFilesInDirectory(configDir)
	if err != nil {
		log.Printf("Ошибка чтения директории с сервера %s: %v", id, err)
		instance.isHealthy = false
		return
	}

	log.Printf("Найдено %d файлов на сервере %s", len(files), id)

	for _, filePath := range files {
		if err := fsm.processSingleFile(id, filePath); err != nil {
			log.Printf("Ошибка обработки файла %s с сервера %s: %v", filePath, id, err)
		}
	}

	instance.lastCheck = time.Now()
}

// processSingleFile обрабатывает один файл
func (fsm *FileServerManager) processSingleFile(serverID, filePath string) error {
	log.Printf("Обработка файла: %s (сервер: %s)", filePath, serverID)

	fileInfo, err := fsm.processor.ProcessFile(filePath)
	if err != nil {
		return fmt.Errorf("не удалось обработать файл: %w", err)
	}

	// Включаем ID сервера в имя устройства для уникальности
	deviceName := fmt.Sprintf("%s-%s", serverID, fileInfo.Name)

	device, err := fsm.db.GetOrCreateDevice(deviceName)
	if err != nil {
		return fmt.Errorf("не удалось получить/создать устройство: %w", err)
	}

	ctx := context.Background()
	_, err = fsm.processor.SaveVersion(ctx, fsm.db, fileInfo, device.ID)
	if err != nil {
		return fmt.Errorf("не удалось сохранить версию: %w", err)
	}

	log.Printf("Успешно обработана версия для %s", deviceName)
	return nil
}

// CheckHealth проверяет состояние всех серверов
func (fsm *FileServerManager) CheckHealth() {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	for id, instance := range fsm.servers {
		if instance.fs == nil {
			// Локальная файловая система всегда здорова
			continue
		}

		healthy := true
		if err := instance.fs.CheckConnection(); err != nil {
			healthy = false
			log.Printf("Проверка состояния не удалась для сервера %s: %v", id, err)
		}

		if !healthy && instance.isHealthy {
			log.Printf("Сервер %s стал нездоровым, попытка перемонтирования", id)
			if err := instance.fs.Mount(); err != nil {
				log.Printf("Не удалось перемонтировать сервер %s: %v", id, err)
			} else {
				healthy = true
			}
		}

		instance.isHealthy = healthy
	}
}

// GetStatus возвращает статус всех серверов
func (fsm *FileServerManager) GetStatus() map[string]interface{} {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	status := make(map[string]interface{})
	servers := make(map[string]interface{})

	for id, instance := range fsm.servers {
		serverInfo := map[string]interface{}{
			"type":      instance.config.Type,
			"server":    instance.config.Server,
			"enabled":   instance.config.Enabled,
			"healthy":   instance.isHealthy,
			"lastCheck": instance.lastCheck.Format(time.RFC3339),
		}
		servers[id] = serverInfo
	}

	status["servers"] = servers
	status["totalServers"] = len(servers)

	return status
}

// Close закрывает все соединения и размонтирует серверы
func (fsm *FileServerManager) Close() error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	var errors []error

	for id, instance := range fsm.servers {
		if instance.fs != nil {
			if err := instance.fs.Unmount(); err != nil {
				errors = append(errors, fmt.Errorf("не удалось размонтировать сервер %s: %w", id, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("ошибки размонтирования: %v", errors)
	}

	return nil
}

// getConfigsFromEnv загружает конфигурации серверов из переменных окружения
func getConfigsFromEnv() map[string]*FileServerConfig {
	configs := make(map[string]*FileServerConfig)

	// Конфигурация для сервера 1
	configs["server1"] = &FileServerConfig{
		ID:         "server1",
		Type:       getEnv("FILE_SERVER_TYPE", "nfs"),
		Server:     getEnv("NFS_SERVER", "192.168.50.149"),
		SharePath:  getEnv("NFS_SHARE_PATH", "/srv/share"),
		MountPoint: getEnv("NFS_MOUNT_POINT", "/mnt/nfs"),
		Username:   getEnv("SMB_USERNAME", "guest"),
		Password:   getEnv("SMB_PASSWORD", ""),
		Domain:     getEnv("SMB_DOMAIN", "WORKGROUP"),
		Enabled:    true,
	}

	// Конфигурация для сервера 2 (пример)
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

	// Добавить больше серверов по мере необходимости...

	return configs
}

// getEnv получает значение переменной окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
