package remotefs

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type FileSystemType string

const (
	NFS FileSystemType = "nfs"
	SMB FileSystemType = "smb"
)

type RemoteFileSystem struct {
	server       string
	sharePath    string
	shareName    string
	mountPoint   string
	fsType       FileSystemType
	username     string
	password     string
	domain       string
	isMounted    bool
	mountOptions []string
}

// NewRemoteFileSystem создает новую удаленную файловую систему
func NewRemoteFileSystem() (*RemoteFileSystem, error) {
	return NewRemoteFileSystemFromEnv("")
}

// NewRemoteFileSystemFromEnv создает файловую систему из переменных окружения с префиксом
func NewRemoteFileSystemFromEnv(prefix string) (*RemoteFileSystem, error) {
	fsType := FileSystemType(getEnvWithPrefix(prefix, "FILE_SERVER_TYPE"))
	if fsType == "" {
		fsType = NFS
	}

	rfs := &RemoteFileSystem{
		fsType: fsType,
	}

	switch fsType {
	case NFS:
		rfs.server = getEnvWithPrefix(prefix, "NFS_SERVER")
		rfs.sharePath = getEnvWithPrefix(prefix, "NFS_SHARE_PATH")
		rfs.mountPoint = getEnvWithPrefix(prefix, "NFS_MOUNT_POINT")
		if rfs.mountPoint == "" {
			rfs.mountPoint = getEnvWithPrefix(prefix, "MOUNT_POINT", "/mnt/nfs")
		}
		rfs.mountOptions = []string{"soft", "intr", "timeo=30", "retrans=3"}

	case SMB:
		rfs.server = getEnvWithPrefix(prefix, "SMB_SERVER")
		rfs.shareName = getEnvWithPrefix(prefix, "SMB_SHARE_NAME")
		rfs.mountPoint = getEnvWithPrefix(prefix, "SMB_MOUNT_POINT")
		if rfs.mountPoint == "" {
			rfs.mountPoint = getEnvWithPrefix(prefix, "MOUNT_POINT", "/mnt/smb")
		}
		rfs.username = getEnvWithPrefix(prefix, "SMB_USERNAME")
		rfs.password = getEnvWithPrefix(prefix, "SMB_PASSWORD")
		rfs.domain = getEnvWithPrefix(prefix, "SMB_DOMAIN")
		if rfs.domain == "" {
			rfs.domain = "WORKGROUP"
		}

	default:
		return nil, fmt.Errorf("unsupported file system type: %s", fsType)
	}

	if err := rfs.validateConfig(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return rfs, nil
}

// getEnvWithPrefix получает переменную окружения с префиксом
func getEnvWithPrefix(prefix, key string, defaultValue ...string) string {
	fullKey := key
	if prefix != "" {
		fullKey = prefix + "_" + key
	}
	if value := os.Getenv(fullKey); value != "" {
		return value
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// validateConfig проверяет конфигурацию файловой системы
func (rfs *RemoteFileSystem) validateConfig() error {
	switch rfs.fsType {
	case NFS:
		if rfs.server == "" || rfs.sharePath == "" {
			return fmt.Errorf("nfs_server and nfs_share_path must be set for nfs")
		}
	case SMB:
		if rfs.server == "" || rfs.shareName == "" {
			return fmt.Errorf("smb_server and smb_share_name must be set for smb")
		}
	}
	return nil
}

// Mount монтирует удаленную файловую систему
func (rfs *RemoteFileSystem) Mount() error {
	if rfs.isMounted {
		return nil
	}

	log.Printf("mounting file system %s from server %s", rfs.fsType, rfs.server)

	if err := os.MkdirAll(rfs.mountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point %s: %w", rfs.mountPoint, err)
	}

	var cmd *exec.Cmd

	switch rfs.fsType {
	case NFS:
		remotePath := fmt.Sprintf("%s:%s", rfs.server, rfs.sharePath)
		options := strings.Join(rfs.mountOptions, ",")
		cmd = exec.Command("mount", "-t", "nfs", "-o", options, remotePath, rfs.mountPoint)

	case SMB:
		remotePath := fmt.Sprintf("//%s/%s", rfs.server, rfs.shareName)
		options := fmt.Sprintf("username=%s,password=%s,domain=%s,iocharset=utf8,vers=3.0",
			rfs.username, rfs.password, rfs.domain)
		cmd = exec.Command("mount", "-t", "cifs", "-o", options, remotePath, rfs.mountPoint)

	default:
		return fmt.Errorf("unsupported file system type: %s", rfs.fsType)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to mount %s: %w, output: %s", rfs.fsType, err, string(output))
	}

	rfs.isMounted = true
	log.Printf("mounted file system %s on %s", rfs.fsType, rfs.mountPoint)
	return nil
}

// Unmount размонтирует файловую систему
func (rfs *RemoteFileSystem) Unmount() error {
	if !rfs.isMounted {
		return nil
	}

	log.Printf("unmounting filesystem %s from %s", rfs.fsType, rfs.mountPoint)

	if output, err := exec.Command("umount", rfs.mountPoint).CombinedOutput(); err != nil {
		log.Printf("warning: failed to unmount %s: %v, output: %s", rfs.fsType, err, string(output))
	}

	rfs.isMounted = false
	return nil
}

// GetConfigDirectory возвращает директорию конфигураций
func (rfs *RemoteFileSystem) GetConfigDirectory() string {
	return rfs.mountPoint
}

// IsMounted проверяет смонтирована ли файловая система
func (rfs *RemoteFileSystem) IsMounted() bool {
	if !rfs.isMounted {
		return false
	}

	if _, err := os.Stat(rfs.mountPoint); err != nil {
		rfs.isMounted = false
		return false
	}

	return true
}

// CheckConnection проверяет соединение с файловой системой
func (rfs *RemoteFileSystem) CheckConnection() error {
	if !rfs.IsMounted() {
		if err := rfs.Mount(); err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
	}

	testFile := filepath.Join(rfs.mountPoint, ".connection_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("cannot write to mount point: %w", err)
	}

	os.Remove(testFile)
	return nil
}
