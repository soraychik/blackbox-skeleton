package repository

import (
	"context"
	"database/sql"

	"blackbox-api/internal/models"
)

type deviceRepository struct {
	db *sql.DB
}

func NewDeviceRepository(db *sql.DB) DeviceRepository {
	return &deviceRepository{db: db}
}

func (r *deviceRepository) GetAll(ctx context.Context) ([]models.Device, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, hostname, mgmt_ip, vendor, model, enabled, created_at FROM devices ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var d models.Device
		if err := rows.Scan(&d.ID, &d.Hostname, &d.MgmtIP, &d.Vendor, &d.Model, &d.Enabled, &d.CreatedAt); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, nil
}

func (r *deviceRepository) GetByID(ctx context.Context, id int) (*models.Device, error) {
	var d models.Device
	err := r.db.QueryRowContext(ctx,
		"SELECT id, hostname, mgmt_ip, vendor, model, enabled, created_at FROM devices WHERE id = ?",
		id,
	).Scan(&d.ID, &d.Hostname, &d.MgmtIP, &d.Vendor, &d.Model, &d.Enabled, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *deviceRepository) GetAllEnabled(ctx context.Context) ([]models.DeviceRow, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, hostname, mgmt_ip, vendor, model FROM devices WHERE enabled = 1 ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []models.DeviceRow
	for rows.Next() {
		var d models.DeviceRow
		var mgmtIP, vendor, model sql.NullString
		if err := rows.Scan(&d.ID, &d.Hostname, &mgmtIP, &vendor, &model); err != nil {
			continue
		}
		if mgmtIP.Valid {
			d.MgmtIP = &mgmtIP.String
		}
		if vendor.Valid {
			d.Vendor = &vendor.String
		}
		if model.Valid {
			d.Model = &model.String
		}
		devices = append(devices, d)
	}
	return devices, nil
}

func (r *deviceRepository) GetAllEnabledWithLatestVersion(ctx context.Context, deviceID *int) ([]models.DeviceVersionRow, error) {
	query := `
		SELECT d.id, d.hostname, d.mgmt_ip, cv.id, cv.storage_type, cv.storage_path
		FROM devices d
		JOIN config_versions cv ON cv.id = (
			SELECT id FROM config_versions
			WHERE device_id = d.id
			ORDER BY created_at DESC
			LIMIT 1
		)
		WHERE d.enabled = 1`
	var args []interface{}

	if deviceID != nil {
		query += " AND d.id = ?"
		args = append(args, *deviceID)
	}
	query += " ORDER BY d.id"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []models.DeviceVersionRow
	for rows.Next() {
		var d models.DeviceVersionRow
		var mgmtIP sql.NullString
		if err := rows.Scan(&d.DeviceID, &d.Hostname, &mgmtIP, &d.VersionID, &d.StorageType, &d.StoragePath); err != nil {
			continue
		}
		if mgmtIP.Valid {
			d.MgmtIP = &mgmtIP.String
		}
		devices = append(devices, d)
	}
	return devices, nil
}
