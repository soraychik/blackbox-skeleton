package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type HostFS struct {
	mu          sync.RWMutex
	currentPath string
}

func NewHostFS() (*HostFS, error) {
	return &HostFS{}, nil
}

func (h *HostFS) Close() error {
	return nil
}

func (h *HostFS) SetHostPath(path string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	hostPath := "/host" + absPath
	if err := h.validatePath(hostPath); err != nil {
		return err
	}

	h.currentPath = hostPath
	return nil
}

func (h *HostFS) GetHostPath() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.currentPath
}

func (h *HostFS) GetOriginalPath() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if strings.HasPrefix(h.currentPath, "/host") {
		return strings.TrimPrefix(h.currentPath, "/host")
	}
	return h.currentPath
}

func (h *HostFS) validatePath(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path does not exist or is not accessible: %s: %w", path, err)
	}
	return nil
}

func (h *HostFS) ListFiles(path string) ([]string, error) {
	if path == "" {
		h.mu.RLock()
		path = h.currentPath
		h.mu.RUnlock()
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", path, err)
	}

	var files []string
	for _, entry := range entries {
		files = append(files, filepath.Join(path, entry.Name()))
	}

	return files, nil
}

func (h *HostFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
