package mocks

import (
	"context"

	"blackbox-api/internal/models"
)

type MockVersionRepository struct {
	GetByIDFn            func(ctx context.Context, id int) (*models.ConfigVersion, error)
	GetByDeviceFn        func(ctx context.Context, deviceID int, from, to string) ([]models.ConfigVersion, error)
	GetPairsByDeviceFn   func(ctx context.Context, deviceID int, from, to string) ([]models.VersionPair, error)
	GetLastDateFn        func(ctx context.Context, deviceID int) (string, error)
	GetLatestForDeviceFn func(ctx context.Context, deviceID int) (*models.ConfigVersion, error)
	ResolveByDateFn      func(ctx context.Context, deviceID int, date1, date2 string) (int, int, error)
}

func (m *MockVersionRepository) GetByID(ctx context.Context, id int) (*models.ConfigVersion, error) {
	return m.GetByIDFn(ctx, id)
}

func (m *MockVersionRepository) GetByDevice(ctx context.Context, deviceID int, from, to string) ([]models.ConfigVersion, error) {
	return m.GetByDeviceFn(ctx, deviceID, from, to)
}

func (m *MockVersionRepository) GetPairsByDevice(ctx context.Context, deviceID int, from, to string) ([]models.VersionPair, error) {
	return m.GetPairsByDeviceFn(ctx, deviceID, from, to)
}

func (m *MockVersionRepository) GetLastDate(ctx context.Context, deviceID int) (string, error) {
	return m.GetLastDateFn(ctx, deviceID)
}

func (m *MockVersionRepository) GetLatestForDevice(ctx context.Context, deviceID int) (*models.ConfigVersion, error) {
	return m.GetLatestForDeviceFn(ctx, deviceID)
}

func (m *MockVersionRepository) ResolveByDate(ctx context.Context, deviceID int, date1, date2 string) (int, int, error) {
	return m.ResolveByDateFn(ctx, deviceID, date1, date2)
}

type MockDeviceRepository struct {
	GetAllFn                         func(ctx context.Context) ([]models.Device, error)
	GetByIDFn                        func(ctx context.Context, id int) (*models.Device, error)
	GetAllEnabledFn                  func(ctx context.Context) ([]models.DeviceRow, error)
	GetAllEnabledWithLatestVersionFn func(ctx context.Context, deviceID *int) ([]models.DeviceVersionRow, error)
}

func (m *MockDeviceRepository) GetAll(ctx context.Context) ([]models.Device, error) {
	return m.GetAllFn(ctx)
}

func (m *MockDeviceRepository) GetByID(ctx context.Context, id int) (*models.Device, error) {
	return m.GetByIDFn(ctx, id)
}

func (m *MockDeviceRepository) GetAllEnabled(ctx context.Context) ([]models.DeviceRow, error) {
	return m.GetAllEnabledFn(ctx)
}

func (m *MockDeviceRepository) GetAllEnabledWithLatestVersion(ctx context.Context, deviceID *int) ([]models.DeviceVersionRow, error) {
	return m.GetAllEnabledWithLatestVersionFn(ctx, deviceID)
}

type MockDiffRepository struct {
	GetIndexFn  func(ctx context.Context, leftID, rightID int) (*models.DiffIndex, error)
	SaveIndexFn func(ctx context.Context, leftID, rightID, added, removed int, storagePath string) error
}

func (m *MockDiffRepository) GetIndex(ctx context.Context, leftID, rightID int) (*models.DiffIndex, error) {
	return m.GetIndexFn(ctx, leftID, rightID)
}

func (m *MockDiffRepository) SaveIndex(ctx context.Context, leftID, rightID, added, removed int, storagePath string) error {
	return m.SaveIndexFn(ctx, leftID, rightID, added, removed, storagePath)
}
