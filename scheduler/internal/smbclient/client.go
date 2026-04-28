package smbclient

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/hirochachacha/go-smb2"
)

// Client — pure-Go SMB2 клиент без монтирования в ОС.
// Соединяется напрямую по TCP:445.
type Client struct {
	server    string
	shareName string
	username  string
	password  string
	domain    string

	conn  net.Conn
	sess  *smb2.Session
	share *smb2.Share
}

func New(server, shareName, username, password, domain string) *Client {
	if domain == "" {
		domain = "WORKGROUP"
	}
	return &Client{
		server:    server,
		shareName: shareName,
		username:  username,
		password:  password,
		domain:    domain,
	}
}

func (c *Client) Connect() error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(c.server, "445"), 15*time.Second)
	if err != nil {
		return fmt.Errorf("tcp connect %s:445: %w", c.server, err)
	}

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     c.username,
			Password: c.password,
			Domain:   c.domain,
		},
	}

	sess, err := d.Dial(conn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smb2 auth: %w", err)
	}

	share, err := sess.Mount(c.shareName)
	if err != nil {
		sess.Logoff()
		conn.Close()
		return fmt.Errorf("mount share %q: %w", c.shareName, err)
	}

	c.conn = conn
	c.sess = sess
	c.share = share
	log.Printf("smb: connected //%s/%s", c.server, c.shareName)
	return nil
}

func (c *Client) Disconnect() {
	if c.share != nil {
		c.share.Umount()
		c.share = nil
	}
	if c.sess != nil {
		c.sess.Logoff()
		c.sess = nil
	}
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

func (c *Client) IsConnected() bool {
	return c.share != nil
}

func (c *Client) reconnect() error {
	c.Disconnect()
	return c.Connect()
}

// FileEntry — метаданные одного файла на шаре.
type FileEntry struct {
	Name    string
	Size    int64
	ModTime time.Time
}

// ListFiles возвращает список файлов (без директорий) в корне шары.
func (c *Client) ListFiles() ([]FileEntry, error) {
	if !c.IsConnected() {
		if err := c.Connect(); err != nil {
			return nil, err
		}
	}

	entries, err := c.share.ReadDir(".")
	if err != nil {
		if err2 := c.reconnect(); err2 != nil {
			return nil, fmt.Errorf("readdir: %w (reconnect: %v)", err, err2)
		}
		entries, err = c.share.ReadDir(".")
		if err != nil {
			return nil, fmt.Errorf("readdir after reconnect: %w", err)
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

	f, err := c.share.Open(name)
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
func (c *Client) ShareName() string { return c.shareName }
