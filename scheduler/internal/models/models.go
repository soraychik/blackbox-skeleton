package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type Device struct {
	ID        int       `json:"id"`
	Hostname  string    `json:"hostname"`
	MgmtIP    *string   `json:"mgmt_ip,omitempty"`
	Vendor    *string   `json:"vendor,omitempty"`
	Model     *string   `json:"model,omitempty"`
	Tags      Tags      `json:"tags,omitempty"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type Tags []string

func (t Tags) Value() (driver.Value, error) {
	if t == nil {
		return nil, nil
	}
	return json.Marshal(t)
}

func (t *Tags) Scan(value interface{}) error {
	if value == nil {
		*t = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, t)
}

type ConfigVersion struct {
	ID              int       `json:"id"`
	DeviceID        int       `json:"device_id"`
	VersionHash     string    `json:"version_hash"`
	StorageType     string    `json:"storage_type"`
	StoragePath     string    `json:"storage_path"`
	ParentVersionID *int      `json:"parent_version_id,omitempty"`
	ChainBaseID     *int      `json:"chain_base_id,omitempty"`
	ChainPosition   int       `json:"chain_position"`
	OriginalSize    uint32    `json:"original_size"`
	CompressedSize  uint32    `json:"compressed_size"`
	CreatedAt       time.Time `json:"created_at"`
}

type DiffIndex struct {
	ID              int       `json:"id"`
	LeftVersionID   int       `json:"left_version_id"`
	RightVersionID  int       `json:"right_version_id"`
	AddedLines      int       `json:"added_lines"`
	RemovedLines    int       `json:"removed_lines"`
	DiffStoragePath *string   `json:"diff_storage_path,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}


type FileInfo struct {
	Name     string
	Path     string
	Size     int64
	ModTime  time.Time
	Content  []byte
	Hash     string
	DeviceID int
	Hostname string
}
