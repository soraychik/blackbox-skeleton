package nfsclient

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/willscott/go-nfs-client/nfs"
	"github.com/willscott/go-nfs-client/nfs/rpc"
)

// Client — pure-Go NFS3 клиент без монтирования в ОС.
// Соединяется через portmapper → mountd → nfsd.
type Client struct {
	server    string
	sharePath string

	mount  *nfs.Mount
	target *nfs.Target
}

func New(server, sharePath string) *Client {
	return &Client{
		server:    server,
		sharePath: sharePath,
	}
}

func (c *Client) Connect() error {
	mount, err := nfs.DialMount(c.server, 15*time.Second)
	if err != nil {
		return fmt.Errorf("dial nfs mount %s: %w", c.server, err)
	}

	auth := rpc.NewAuthUnix("blackbox-scheduler", 0, 0)
	target, err := mount.Mount(c.sharePath, auth.Auth())
	if err != nil {
		mount.Close()
		return fmt.Errorf("nfs mount %s:%s: %w", c.server, c.sharePath, err)
	}

	c.mount = mount
	c.target = target
	log.Printf("nfs: connected %s:%s", c.server, c.sharePath)
	return nil
}

func (c *Client) Disconnect() {
	if c.target != nil {
		c.target.Close()
		c.target = nil
	}
	if c.mount != nil {
		c.mount.Unmount()
		c.mount.Close()
		c.mount = nil
	}
}

func (c *Client) IsConnected() bool {
	return c.target != nil
}

func (c *Client) reconnect() error {
	c.Disconnect()
	return c.Connect()
}

// FileEntry — метаданные одного файла на экспорте.
type FileEntry struct {
	Name    string
	Size    int64
	ModTime time.Time
}

// ListFiles возвращает список файлов (без директорий) в корне экспорта.
func (c *Client) ListFiles() ([]FileEntry, error) {
	if !c.IsConnected() {
		if err := c.Connect(); err != nil {
			return nil, err
		}
	}

	entries, err := c.target.ReadDirPlus(".")
	if err != nil {
		if err2 := c.reconnect(); err2 != nil {
			return nil, fmt.Errorf("readdirplus: %w (reconnect: %v)", err, err2)
		}
		entries, err = c.target.ReadDirPlus(".")
		if err != nil {
			return nil, fmt.Errorf("readdirplus after reconnect: %w", err)
		}
	}

	var files []FileEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		files = append(files, FileEntry{
			Name:    e.Name(),
			Size:    e.Size(),
			ModTime: e.ModTime(),
		})
	}
	return files, nil
}

// ReadFile читает содержимое файла по имени.
func (c *Client) ReadFile(name string) ([]byte, error) {
	if !c.IsConnected() {
		if err := c.Connect(); err != nil {
			return nil, err
		}
	}

	f, err := c.target.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	defer f.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, f); err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	return buf.Bytes(), nil
}

func (c *Client) Server() string    { return c.server }
func (c *Client) SharePath() string { return c.sharePath }
