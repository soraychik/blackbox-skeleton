package handlers

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"blackbox-api/internal/audit"

	"github.com/gin-gonic/gin"
)

type ScanHandler struct {
	audit *audit.Logger
}

func NewScanHandler(auditLog *audit.Logger) *ScanHandler {
	return &ScanHandler{audit: auditLog}
}

func (h *ScanHandler) PostTriggerScan(c *gin.Context) {
	schedulerURL := getSchedulerURL()
	url := strings.TrimSuffix(schedulerURL, "/") + "/scan"

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, url, bytes.NewReader(nil))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("trigger scan request to scheduler failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Scheduler unreachable"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": string(body)})
		return
	}

	u, r, ip := audit.FromGin(c)
	h.audit.Log(u, r, "trigger_scan", map[string]any{}, ip)
	c.Data(resp.StatusCode, "application/json", body)
}

func GetScanStatus(c *gin.Context) {
	schedulerURL := getSchedulerURL()
	url := strings.TrimSuffix(schedulerURL, "/") + "/status"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("scan status request to scheduler failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Scheduler unreachable"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": string(body)})
		return
	}
	c.Data(http.StatusOK, "application/json", body)
}

func getSchedulerURL() string {
	if v := os.Getenv("SCHEDULER_TRIGGER_URL"); v != "" {
		return v
	}
	return "http://scheduler:9090"
}
