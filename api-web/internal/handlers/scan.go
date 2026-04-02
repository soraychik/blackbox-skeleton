package handlers

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"blackbox-api/internal/service"

	"github.com/gin-gonic/gin"
)

func PostTriggerScan(c *gin.Context) {
	schedulerURL := service.GetSchedulerURL()
	url := strings.TrimSuffix(schedulerURL, "/") + "/scan"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(nil))
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
	c.Data(resp.StatusCode, "application/json", body)
}

func GetScanStatus(c *gin.Context) {
	schedulerURL := service.GetSchedulerURL()
	url := strings.TrimSuffix(schedulerURL, "/") + "/status"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("scan status request to scheduler failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "scheduler unreachable"})
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
