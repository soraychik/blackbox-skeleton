package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"blackbox-api/internal/audit"
	"blackbox-api/internal/db"
	"blackbox-api/internal/repository"
	"blackbox-api/internal/service"

	"github.com/gin-gonic/gin"
)

type DevicesHandler struct {
	db          *sql.DB
	deviceRepo  repository.DeviceRepository
	versionRepo repository.VersionRepository
	audit       *audit.Logger
}

func NewDevicesHandler(db *sql.DB, deviceRepo repository.DeviceRepository, versionRepo repository.VersionRepository, auditLog *audit.Logger) *DevicesHandler {
	return &DevicesHandler{
		db:          db,
		deviceRepo:  deviceRepo,
		versionRepo: versionRepo,
		audit:       auditLog,
	}
}

func (h *DevicesHandler) GetDevices(c *gin.Context) {
	devices, err := h.deviceRepo.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query devices"})
		return
	}

	siteMappings, err := db.GetSiteMappings(h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load site mappings"})
		return
	}
	for i := range devices {
		service.EnrichDeviceMetadata(&devices[i], siteMappings)
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

func (h *DevicesHandler) GetDeviceByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	device, err := h.deviceRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get device"})
		return
	}
	if device == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	siteMappings, err := db.GetSiteMappings(h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load site mappings"})
		return
	}
	service.EnrichDeviceMetadata(device, siteMappings)

	c.JSON(http.StatusOK, gin.H{"device": device})
}

func (h *DevicesHandler) GetDeviceVersions(c *gin.Context) {
	deviceID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	device, err := h.deviceRepo.GetByID(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get device"})
		return
	}
	if device == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	siteMappings, err := db.GetSiteMappings(h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load site mappings"})
		return
	}
	service.EnrichDeviceMetadata(device, siteMappings)

	fromStr := c.Query("from")
	toStr := c.Query("to")

	versions, err := h.versionRepo.GetByDevice(c.Request.Context(), deviceID, fromStr, toStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query versions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"device": device, "versions": versions})
}

func (h *DevicesHandler) GetLatestVersionsForDevices(c *gin.Context) {
	leftID, err := strconv.Atoi(c.Query("leftDeviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid left device ID"})
		return
	}
	rightID, err := strconv.Atoi(c.Query("rightDeviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid right device ID"})
		return
	}

	leftDevice, err := h.deviceRepo.GetByID(c.Request.Context(), leftID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get left device"})
		return
	}
	if leftDevice == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Left device not found"})
		return
	}

	rightDevice, err := h.deviceRepo.GetByID(c.Request.Context(), rightID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get right device"})
		return
	}
	if rightDevice == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Right device not found"})
		return
	}

	siteMappings, err := db.GetSiteMappings(h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load site mappings"})
		return
	}
	service.EnrichDeviceMetadata(leftDevice, siteMappings)
	service.EnrichDeviceMetadata(rightDevice, siteMappings)

	leftVersion, err := h.versionRepo.GetLatestForDevice(c.Request.Context(), leftID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get left latest version"})
		return
	}
	rightVersion, err := h.versionRepo.GetLatestForDevice(c.Request.Context(), rightID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get right latest version"})
		return
	}

	u, r, ip := audit.FromGin(c)
	h.audit.Log(u, r, "compare_devices", map[string]any{"left_device_id": leftID, "right_device_id": rightID}, ip)
	c.JSON(http.StatusOK, gin.H{
		"left": gin.H{
			"device":  leftDevice,
			"version": leftVersion,
		},
		"right": gin.H{
			"device":  rightDevice,
			"version": rightVersion,
		},
	})
}
