package models

import "time"

// Device представляет сетевое устройство
type Device struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// ConfigVersion представляет версию конфигурации
type ConfigVersion struct {
	ID              int       `json:"id"`
	DeviceID        int       `json:"device_id"`
	VersionDate     time.Time `json:"version_date"`
	FilePath        string    `json:"file_path"`
	FileHash        string    `json:"file_hash"`
	StorageType     string    `json:"storage_type"`      // "full", "diff", "base"
	ParentVersionID *int      `json:"parent_version_id"` // для diff версий
	MinioObjectName string    `json:"minio_object_name"` // путь к объекту в MinIO
	DiffSizeBytes   int       `json:"diff_size_bytes"`   // размер diff в байтах
	CreatedAt       time.Time `json:"created_at"`
}

// FileInfo информация о файле для обработки
type FileInfo struct {
	Name    string
	Path    string
	Size    int64
	ModTime time.Time
	Content []byte
	Hash    string
}
