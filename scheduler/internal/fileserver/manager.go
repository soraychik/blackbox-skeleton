package fileserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"strings"

	"blackbox-scheduler/internal/database"
	"blackbox-scheduler/internal/fileprocessor"
	"blackbox-scheduler/internal/models"
	"blackbox-scheduler/internal/nfsclient"
	"blackbox-scheduler/internal/smbclient"
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
	nfsClient *nfsclient.Client // для NFS (pure-Go, без mount)
	smbClient *smbclient.Client // для SMB (pure-Go, без mount)
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
		switch config.Type {
		case "local":
			log.Printf("file server added: %s (local: %s)", id, config.LocalPath)
		case "smb":
			log.Printf("file server added: %s (smb://%s/%s)", id, config.Server, config.ShareName)
		default:
			log.Printf("file server added: %s (%s://%s)", id, config.Type, config.Server)
		}
	}

	if len(fsm.servers) == 0 {
		log.Printf("warning: no file sources configured, scanning will not occur")
	}
	return nil
}

func (fsm *FileServerManager) createServerInstance(id string, config *FileServerConfig) (*FileServerInstance, error) {
	switch config.Type {
	case "local":
		return &FileServerInstance{config: config, isHealthy: true}, nil

	case "smb":
		client := smbclient.New(config.Server, config.ShareName, config.Username, config.Password, config.Domain)
		return &FileServerInstance{config: config, smbClient: client, isHealthy: false}, nil

	case "nfs":
		client := nfsclient.New(config.Server, config.SharePath)
		return &FileServerInstance{config: config, nfsClient: client, isHealthy: false}, nil

	default:
		return nil, fmt.Errorf("unsupported file server type: %s", config.Type)
	}
}

func (fsm *FileServerManager) MountAllServers() error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	var errs []string
	for id, instance := range fsm.servers {
		switch {
		case instance.smbClient != nil:
			if err := instance.smbClient.Connect(); err != nil {
				errs = append(errs, fmt.Sprintf("smb %s: %v", id, err))
				instance.isHealthy = false
			} else {
				instance.isHealthy = true
			}
		case instance.nfsClient != nil:
			if err := instance.nfsClient.Connect(); err != nil {
				errs = append(errs, fmt.Sprintf("nfs %s: %v", id, err))
				instance.isHealthy = false
			} else {
				instance.isHealthy = true
				log.Printf("nfs connected server %s", id)
			}
		default:
			instance.isHealthy = true // local
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("mount errors: %v", errs)
	}
	return nil
}

func (fsm *FileServerManager) ProcessAllServers() {
	fsm.mu.RLock()
	servers := make(map[string]*FileServerInstance, len(fsm.servers))
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

func (fsm *FileServerManager) processServer(id string, instance *FileServerInstance) {
	if instance.smbClient != nil {
		fsm.processSMBServer(id, instance)
		return
	}
	if instance.nfsClient != nil {
		fsm.processNFSServer(id, instance)
		return
	}

	var configDir string
	if instance.config.Type == "local" {
		configDir = instance.config.LocalPath
	} else {
		log.Printf("unknown server type for %s, skipping", id)
		return
	}

	files, err := fsm.processor.GetFilesInDirectory(configDir)
	if err != nil {
		log.Printf("error reading directory from server %s (path %q): %v", id, configDir, err)
		instance.isHealthy = false
		return
	}
	log.Printf("%d files found on server %s", len(files), id)

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

func (fsm *FileServerManager) processSMBServer(id string, instance *FileServerInstance) {
	entries, err := instance.smbClient.ListFiles()
	if err != nil {
		log.Printf("smb list files failed for %s: %v", id, err)
		instance.isHealthy = false
		return
	}
	log.Printf("%d files found on SMB server %s", len(entries), id)

	fileStates, err := fsm.db.GetAllFileStates()
	if err != nil {
		log.Printf("GetAllFileStates failed for SMB server %s: %v, processing all files", id, err)
		fileStates = map[string]*database.FileState{}
	}

	var candidates []smbclient.FileEntry
	skipped := 0
	for _, e := range entries {
		state, ok := fileStates[e.Name]
		if ok && state.Size == e.Size && state.ModTime.Equal(e.ModTime) {
			skipped++
			continue
		}
		candidates = append(candidates, e)
	}
	log.Printf("SMB server %s: %d to process, %d skipped by mtime+size", id, len(candidates), skipped)

	if len(candidates) == 0 {
		instance.lastCheck = time.Now()
		return
	}

	sem := make(chan struct{}, fileWorkers)
	var wg sync.WaitGroup

	for _, entry := range candidates {
		wg.Add(1)
		e := entry
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := fsm.processSMBFile(id, instance.smbClient, e); err != nil {
				log.Printf("error processing SMB file %s from server %s: %v", e.Name, id, err)
			}
		}()
	}

	wg.Wait()
	instance.lastCheck = time.Now()
}

func (fsm *FileServerManager) processNFSServer(id string, instance *FileServerInstance) {
	entries, err := instance.nfsClient.ListFiles()
	if err != nil {
		log.Printf("nfs list files failed for %s: %v", id, err)
		instance.isHealthy = false
		return
	}
	log.Printf("%d files found on NFS server %s", len(entries), id)

	fileStates, err := fsm.db.GetAllFileStates()
	if err != nil {
		log.Printf("GetAllFileStates failed for NFS server %s: %v, processing all files", id, err)
		fileStates = map[string]*database.FileState{}
	}

	var candidates []nfsclient.FileEntry
	skipped := 0
	for _, e := range entries {
		state, ok := fileStates[e.Name]
		if ok && state.Size == e.Size && state.ModTime.Equal(e.ModTime) {
			skipped++
			continue
		}
		candidates = append(candidates, e)
	}
	log.Printf("NFS server %s: %d to process, %d skipped by mtime+size", id, len(candidates), skipped)

	if len(candidates) == 0 {
		instance.lastCheck = time.Now()
		return
	}

	sem := make(chan struct{}, fileWorkers)
	var wg sync.WaitGroup

	for _, entry := range candidates {
		wg.Add(1)
		e := entry
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := fsm.processNFSFile(id, instance.nfsClient, e); err != nil {
				log.Printf("error processing NFS file %s from server %s: %v", e.Name, id, err)
			}
		}()
	}

	wg.Wait()
	instance.lastCheck = time.Now()
}

func (fsm *FileServerManager) processNFSFile(serverID string, client *nfsclient.Client, entry nfsclient.FileEntry) error {
	content, err := client.ReadFile(entry.Name)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	normalizedContent := bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	hash := fmt.Sprintf("%x", sha256.Sum256(normalizedContent))

	fileInfo := &models.FileInfo{
		Name:     entry.Name,
		Path:     fmt.Sprintf("%s:%s/%s", client.Server(), client.SharePath(), entry.Name),
		Size:     entry.Size,
		ModTime:  entry.ModTime,
		Content:  normalizedContent,
		Hash:     hash,
		Hostname: entry.Name,
	}

	ctx := context.Background()
	_, err = fsm.processor.SaveVersion(ctx, fsm.db, fileInfo)
	if err != nil {
		return fmt.Errorf("save version: %w", err)
	}

	if err := fsm.db.UpsertFileState(fileInfo.Hostname, fileInfo.Size, fileInfo.ModTime); err != nil {
		log.Printf("warning: failed to upsert file state for %s: %v", fileInfo.Hostname, err)
	}

	log.Printf("processed NFS version for %s-%s", serverID, fileInfo.Hostname)
	return nil
}

// processSMBFile читает файл через SMB и сохраняет версию без обращения к локальной ФС.
func (fsm *FileServerManager) processSMBFile(serverID string, client *smbclient.Client, entry smbclient.FileEntry) error {
	content, err := client.ReadFile(entry.Name)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	normalizedContent := bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	hash := fmt.Sprintf("%x", sha256.Sum256(normalizedContent))

	fileInfo := &models.FileInfo{
		Name:     entry.Name,
		Path:     fmt.Sprintf("//%s/%s/%s", client.Server(), client.ShareName(), entry.Name),
		Size:     entry.Size,
		ModTime:  entry.ModTime,
		Content:  normalizedContent,
		Hash:     hash,
		Hostname: entry.Name,
	}

	ctx := context.Background()
	_, err = fsm.processor.SaveVersion(ctx, fsm.db, fileInfo)
	if err != nil {
		return fmt.Errorf("save version: %w", err)
	}

	if err := fsm.db.UpsertFileState(fileInfo.Hostname, fileInfo.Size, fileInfo.ModTime); err != nil {
		log.Printf("warning: failed to upsert file state for %s: %v", fileInfo.Hostname, err)
	}

	log.Printf("processed SMB version for %s-%s", serverID, fileInfo.Hostname)
	return nil
}

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
		switch {
		case instance.smbClient != nil:
			_, err := instance.smbClient.ListFiles()
			if err != nil {
				log.Printf("smb health check failed for %s: %v, reconnecting", id, err)
				if err2 := instance.smbClient.Connect(); err2 != nil {
					log.Printf("smb reconnect failed for %s: %v", id, err2)
					instance.isHealthy = false
				} else {
					instance.isHealthy = true
				}
			}
		case instance.nfsClient != nil:
			_, err := instance.nfsClient.ListFiles()
			if err != nil {
				log.Printf("nfs health check failed for %s: %v, reconnecting", id, err)
				if err2 := instance.nfsClient.Connect(); err2 != nil {
					log.Printf("nfs reconnect failed for %s: %v", id, err2)
					instance.isHealthy = false
				} else {
					instance.isHealthy = true
				}
			}
		}
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

	var errs []error
	for _, instance := range fsm.servers {
		switch {
		case instance.smbClient != nil:
			instance.smbClient.Disconnect()
		case instance.nfsClient != nil:
			instance.nfsClient.Disconnect()
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// ReloadConfigSource читает настройки источника из system_settings и применяет их.
func (fsm *FileServerManager) ReloadConfigSource(db *database.DB) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	cfg, err := db.GetConfigSourceSettings()
	if err != nil {
		return fmt.Errorf("failed to get config source settings: %w", err)
	}

	switch cfg.Type {
	case "smb":
		if inst, ok := fsm.servers["db-nfs"]; ok {
			inst.nfsClient.Disconnect()
			delete(fsm.servers, "db-nfs")
		}
		delete(fsm.servers, "db-local")

		smbServer := firstNonEmpty(cfg.SmbServer, os.Getenv("SMB_SERVER"))
		smbShare := firstNonEmpty(cfg.SmbShare, os.Getenv("SMB_SHARE_NAME"))
		smbUsername := firstNonEmpty(cfg.SmbUsername, os.Getenv("SMB_USERNAME"))
		smbPassword := firstNonEmpty(cfg.SmbPassword, os.Getenv("SMB_PASSWORD"))
		smbDomain := firstNonEmpty(cfg.SmbDomain, os.Getenv("SMB_DOMAIN"), "WORKGROUP")

		// Пропускаем если параметры не изменились
		if inst, ok := fsm.servers["db-smb"]; ok {
			c := inst.smbClient
			if c != nil &&
				c.Server() == smbServer &&
				c.ShareName() == smbShare &&
				inst.config.Username == smbUsername &&
				inst.config.Password == smbPassword &&
				inst.config.Domain == smbDomain {
				return nil
			}
			inst.smbClient.Disconnect()
			delete(fsm.servers, "db-smb")
		}

		if smbServer == "" || smbShare == "" {
			log.Printf("smb: server or share not configured, skipping")
			return nil
		}

		client := smbclient.New(smbServer, smbShare, smbUsername, smbPassword, smbDomain)
		if err := client.Connect(); err != nil {
			return fmt.Errorf("smb connect //%s/%s: %w", smbServer, smbShare, err)
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
				Enabled:   true,
			},
			smbClient: client,
			lastCheck: time.Now(),
			isHealthy: true,
		}
		log.Printf("SMB server configured from DB: //%s/%s", smbServer, smbShare)

	case "nfs":
		if inst, ok := fsm.servers["db-smb"]; ok {
			inst.smbClient.Disconnect()
			delete(fsm.servers, "db-smb")
		}
		delete(fsm.servers, "db-local")

		nfsServer := firstNonEmpty(cfg.NfsServer, os.Getenv("NFS_SERVER"))
		nfsPath := firstNonEmpty(cfg.NfsPath, os.Getenv("NFS_SHARE_PATH"), "/srv/share")

		if inst, ok := fsm.servers["db-nfs"]; ok {
			c := inst.nfsClient
			if c != nil && c.Server() == nfsServer && c.SharePath() == nfsPath {
				return nil
			}
			inst.nfsClient.Disconnect()
			delete(fsm.servers, "db-nfs")
		}

		if nfsServer == "" {
			log.Printf("nfs: server not configured, skipping")
			return nil
		}

		client := nfsclient.New(nfsServer, nfsPath)
		if err := client.Connect(); err != nil {
			return fmt.Errorf("nfs connect %s:%s: %w", nfsServer, nfsPath, err)
		}

		fsm.servers["db-nfs"] = &FileServerInstance{
			config: &FileServerConfig{
				ID:        "db-nfs",
				Type:      "nfs",
				Server:    nfsServer,
				SharePath: nfsPath,
				Enabled:   true,
			},
			nfsClient: client,
			lastCheck: time.Now(),
			isHealthy: true,
		}
		log.Printf("NFS server configured from DB: %s:%s", nfsServer, nfsPath)

	case "local":
		localPath := resolveHostPath(firstNonEmpty(cfg.Path, os.Getenv("LOCAL_FS_PATH"), "/app/configs"))

		if smbInst, ok := fsm.servers["db-smb"]; ok {
			smbInst.smbClient.Disconnect()
			delete(fsm.servers, "db-smb")
		}
		if nfsInst, ok := fsm.servers["db-nfs"]; ok {
			nfsInst.nfsClient.Disconnect()
			delete(fsm.servers, "db-nfs")
		}

		if localInst, ok := fsm.servers["db-local"]; ok {
			oldPath := localInst.config.LocalPath
			localInst.config.LocalPath = localPath
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
				lastCheck: time.Now(),
				isHealthy: true,
			}
			log.Printf("local config source configured from DB: %s", localPath)
		}
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
		fsType := getEnv("FILE_SERVER_TYPE", "nfs")
		var cfg *FileServerConfig
		if fsType == "smb" {
			cfg = &FileServerConfig{
				ID:        "server1",
				Type:      "smb",
				Server:    getEnv("SMB_SERVER", ""),
				ShareName: getEnv("SMB_SHARE_NAME", ""),
				Username:  getEnv("SMB_USERNAME", "guest"),
				Password:  getEnv("SMB_PASSWORD", ""),
				Domain:    getEnv("SMB_DOMAIN", "WORKGROUP"),
				Enabled:   true,
			}
		} else {
			cfg = &FileServerConfig{
				ID:         "server1",
				Type:       fsType,
				Server:     getEnv("NFS_SERVER", ""),
				SharePath:  getEnv("NFS_SHARE_PATH", "/srv/share"),
				MountPoint: getEnv("NFS_MOUNT_POINT", "/mnt/nfs"),
				Enabled:    true,
			}
		}
		configs["server1"] = cfg
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

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// resolveHostPath транслирует путь пользователя (с хост-машины) в путь внутри контейнера.
// Хостовая ФС примонтирована в /host (см. docker-compose volumes: /:/host:ro).
// Linux:   /srv/configs     → /host/srv/configs
// Windows: C:\configs       → /host/c/configs
// Windows: C:/configs       → /host/c/configs
func resolveHostPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	// Уже внутри контейнера (dev-режим или уже /host/...)
	if strings.HasPrefix(p, "/host/") || strings.HasPrefix(p, "/app/") {
		return p
	}
	// Windows: C:\path или C:/path
	if len(p) >= 2 && p[1] == ':' {
		drive := strings.ToLower(string(p[0]))
		rest := strings.ReplaceAll(p[2:], "\\", "/")
		rest = strings.TrimPrefix(rest, "/")
		return "/host/" + drive + "/" + rest
	}
	// Linux: /absolute/path
	return "/host" + p
}

