package handlers

import (
	"database/sql"

	"blackbox-api/internal/audit"
	"blackbox-api/internal/repository"
	"blackbox-api/internal/storage"
)

type Handlers struct {
	Devices   *DevicesHandler
	Versions  *VersionsHandler
	Search    *SearchHandler
	Export    *ExportHandler
	Dashboard *DashboardHandler
	Settings  *SettingsHandler
	Scan      *ScanHandler
	Audit     *AuditHandler
}

func NewHandlers(
	db *sql.DB,
	minio *storage.MinIOImprovedClient,
	versionRepo repository.VersionRepository,
	deviceRepo repository.DeviceRepository,
	diffRepo repository.DiffRepository,
	auditLog *audit.Logger,
) *Handlers {
	return &Handlers{
		Devices:   NewDevicesHandler(db, deviceRepo, versionRepo, auditLog),
		Versions:  NewVersionsHandler(db, minio, versionRepo, diffRepo, auditLog),
		Search:    NewSearchHandler(minio, versionRepo, deviceRepo, db, auditLog),
		Export:    NewExportHandler(db, minio, versionRepo, auditLog),
		Dashboard: NewDashboardHandler(db),
		Settings:  NewSettingsHandler(db, auditLog),
		Scan:      NewScanHandler(auditLog),
		Audit:     NewAuditHandler(db),
	}
}
