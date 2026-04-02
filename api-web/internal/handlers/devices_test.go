package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"blackbox-api/internal/db"
	"blackbox-api/internal/models"
	"blackbox-api/internal/repository/mocks"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupSiteMappings() {
	db.SetSiteMappingsForTest(map[string]string{
		"ekb": "Екатеринбург",
		"ntg": "Нижний Тагил",
	})
}

func TestGetDeviceByID_Found(t *testing.T) {
	setupSiteMappings()

	mockDeviceRepo := &mocks.MockDeviceRepository{
		GetByIDFn: func(ctx context.Context, id int) (*models.Device, error) {
			if id == 1 {
				ip := "192.168.1.1"
				return &models.Device{
					ID:        1,
					Hostname:  "router-01",
					MgmtIP:    &ip,
					Enabled:   true,
					CreatedAt: time.Now(),
				}, nil
			}
			return nil, nil
		},
	}

	handler := &DevicesHandler{
		deviceRepo: mockDeviceRepo,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/devices/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.GetDeviceByID(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	device, ok := response["device"].(map[string]interface{})
	if !ok {
		t.Fatal("expected device in response")
	}
	if device["id"].(float64) != 1 {
		t.Errorf("expected device id 1, got %v", device["id"])
	}
	if device["hostname"].(string) != "router-01" {
		t.Errorf("expected hostname router-01, got %v", device["hostname"])
	}
}

func TestGetDeviceByID_NotFound(t *testing.T) {
	setupSiteMappings()

	mockDeviceRepo := &mocks.MockDeviceRepository{
		GetByIDFn: func(ctx context.Context, id int) (*models.Device, error) {
			return nil, nil
		},
	}

	handler := &DevicesHandler{
		deviceRepo: mockDeviceRepo,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/devices/999", nil)
	c.Params = gin.Params{{Key: "id", Value: "999"}}

	handler.GetDeviceByID(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestGetDeviceByID_InvalidID(t *testing.T) {
	mockDeviceRepo := &mocks.MockDeviceRepository{}

	handler := &DevicesHandler{
		deviceRepo: mockDeviceRepo,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/devices/abc", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	handler.GetDeviceByID(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetDevices_Empty(t *testing.T) {
	setupSiteMappings()

	mockDeviceRepo := &mocks.MockDeviceRepository{
		GetAllFn: func(ctx context.Context) ([]models.Device, error) {
			return []models.Device{}, nil
		},
	}

	handler := &DevicesHandler{
		deviceRepo: mockDeviceRepo,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/devices", nil)

	handler.GetDevices(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	devices, ok := response["devices"].([]interface{})
	if !ok {
		t.Fatal("expected devices in response")
	}
	if len(devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(devices))
	}
}

func TestGetDevices_WithDevices(t *testing.T) {
	setupSiteMappings()

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"
	mockDeviceRepo := &mocks.MockDeviceRepository{
		GetAllFn: func(ctx context.Context) ([]models.Device, error) {
			return []models.Device{
				{ID: 1, Hostname: "router-01.ekb-config", MgmtIP: &ip1, Enabled: true, CreatedAt: time.Now()},
				{ID: 2, Hostname: "router-02.ntg-config", MgmtIP: &ip2, Enabled: true, CreatedAt: time.Now()},
			}, nil
		},
	}

	handler := &DevicesHandler{
		deviceRepo: mockDeviceRepo,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/devices", nil)

	handler.GetDevices(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	devices, ok := response["devices"].([]interface{})
	if !ok {
		t.Fatal("expected devices in response")
	}
	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}
}

func TestGetDevices_DBError(t *testing.T) {
	mockDeviceRepo := &mocks.MockDeviceRepository{
		GetAllFn: func(ctx context.Context) ([]models.Device, error) {
			return nil, errors.New("database error")
		},
	}

	handler := &DevicesHandler{
		deviceRepo: mockDeviceRepo,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/devices", nil)

	handler.GetDevices(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}
