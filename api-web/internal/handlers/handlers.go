package handlers

import (
	"database/sql"

	"blackbox-api/internal/repository"
	"blackbox-api/internal/storage"
)

type Handlers struct {
	Devices   *DevicesHandler
	Versions  *VersionsHandler
	Search    *SearchHandler
	Export    *ExportHandler
	Dashboard *DashboardHandler
}

func NewHandlers(
	db *sql.DB,
	minio *storage.MinIOImprovedClient,
	versionRepo repository.VersionRepository,
	deviceRepo repository.DeviceRepository,
	diffRepo repository.DiffRepository,
) *Handlers {
	return &Handlers{
		Devices:   NewDevicesHandler(db, deviceRepo, versionRepo),
		Versions:  NewVersionsHandler(db, minio, versionRepo, diffRepo),
		Search:    NewSearchHandler(minio, versionRepo, deviceRepo, db),
		Export:    NewExportHandler(db, minio, versionRepo),
		Dashboard: NewDashboardHandler(db),
	}
}