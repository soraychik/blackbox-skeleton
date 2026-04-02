package repository

import (
	"context"

	"blackbox-api/internal/models"
)

type VersionRepository interface {
	GetByID(ctx context.Context, id int) (*models.ConfigVersion, error)
	GetByDevice(ctx context.Context, deviceID int, from, to string) ([]models.ConfigVersion, error)
	GetPairsByDevice(ctx context.Context, deviceID int, from, to string) ([]models.VersionPair, error)
	GetLastDate(ctx context.Context, deviceID int) (string, error)
	GetLatestForDevice(ctx context.Context, deviceID int) (*models.ConfigVersion, error)
	ResolveByDate(ctx context.Context, deviceID int, date1, date2 string) (id1, id2 int, err error)
}

type DeviceRepository interface {
	GetAll(ctx context.Context) ([]models.Device, error)
	GetByID(ctx context.Context, id int) (*models.Device, error)
	GetAllEnabled(ctx context.Context) ([]models.DeviceRow, error)
	GetAllEnabledWithLatestVersion(ctx context.Context, deviceID *int) ([]models.DeviceVersionRow, error)
}

type DiffRepository interface {
	GetIndex(ctx context.Context, leftID, rightID int) (*models.DiffIndex, error)
	SaveIndex(ctx context.Context, leftID, rightID, added, removed int, diff string) error
}
