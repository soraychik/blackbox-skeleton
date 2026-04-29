package repository

import (
	"context"
	"database/sql"
	"strconv"

	"blackbox-api/internal/models"
)

type versionRepository struct {
	db *sql.DB
}

func NewVersionRepository(db *sql.DB) VersionRepository {
	return &versionRepository{db: db}
}

func (r *versionRepository) GetByID(ctx context.Context, id int) (*models.ConfigVersion, error) {
	var v models.ConfigVersion
	var parentID, chainBaseID sql.NullInt32
	err := r.db.QueryRowContext(ctx, `
		SELECT id, device_id, version_hash, storage_type, storage_path, 
		       parent_version_id, chain_base_id, chain_position, 
		       original_size, compressed_size, created_at 
		FROM config_versions WHERE id = ?`, id).Scan(
		&v.ID, &v.DeviceID, &v.VersionHash, &v.StorageType, &v.StoragePath,
		&parentID, &chainBaseID, &v.ChainPosition, &v.OriginalSize, &v.CompressedSize, &v.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		pid := int(parentID.Int32)
		v.ParentVersionID = &pid
	}
	if chainBaseID.Valid {
		cid := int(chainBaseID.Int32)
		v.ChainBaseID = &cid
	}
	return &v, nil
}

func (r *versionRepository) GetByDevice(ctx context.Context, deviceID int, from, to string) ([]models.ConfigVersion, error) {
	query := `SELECT id, device_id, version_hash, storage_type, storage_path, 
		       parent_version_id, chain_base_id, chain_position, 
		       original_size, compressed_size, created_at 
		FROM config_versions 
		WHERE device_id = ?`
	args := []interface{}{deviceID}

	if from != "" && len(from) == 10 && from[4] == '-' && from[7] == '-' {
		query += " AND DATE(created_at) >= ?"
		args = append(args, from)
	} else if from != "" {
		if fromID, parseErr := strconv.Atoi(from); parseErr == nil {
			query += " AND id >= ?"
			args = append(args, fromID)
		}
	}
	if to != "" && len(to) == 10 && to[4] == '-' && to[7] == '-' {
		query += " AND DATE(created_at) <= ?"
		args = append(args, to)
	} else if to != "" {
		if toID, parseErr := strconv.Atoi(to); parseErr == nil {
			query += " AND id <= ?"
			args = append(args, toID)
		}
	}

	query += " ORDER BY id DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []models.ConfigVersion
	for rows.Next() {
		var v models.ConfigVersion
		var parentID, chainBaseID sql.NullInt32
		if err := rows.Scan(&v.ID, &v.DeviceID, &v.VersionHash, &v.StorageType, &v.StoragePath,
			&parentID, &chainBaseID, &v.ChainPosition, &v.OriginalSize, &v.CompressedSize, &v.CreatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			pid := int(parentID.Int32)
			v.ParentVersionID = &pid
		}
		if chainBaseID.Valid {
			cid := int(chainBaseID.Int32)
			v.ChainBaseID = &cid
		}
		versions = append(versions, v)
	}
	return versions, nil
}

func (r *versionRepository) GetPairsByDevice(ctx context.Context, deviceID int, from, to string) ([]models.VersionPair, error) {
	query := `SELECT id, device_id, version_hash, storage_type, storage_path, 
		       parent_version_id, chain_base_id, chain_position, 
		       original_size, compressed_size, created_at 
		FROM config_versions WHERE device_id = ?`
	args := []interface{}{deviceID}
	if from != "" && len(from) == 10 {
		query += " AND DATE(created_at) >= ?"
		args = append(args, from)
	}
	if to != "" && len(to) == 10 {
		query += " AND DATE(created_at) <= ?"
		args = append(args, to)
	}
	query += " ORDER BY created_at ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*models.ConfigVersion
	for rows.Next() {
		var v models.ConfigVersion
		var parentID, chainBaseID sql.NullInt32
		if err := rows.Scan(&v.ID, &v.DeviceID, &v.VersionHash, &v.StorageType, &v.StoragePath,
			&parentID, &chainBaseID, &v.ChainPosition, &v.OriginalSize, &v.CompressedSize, &v.CreatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			pid := int(parentID.Int32)
			v.ParentVersionID = &pid
		}
		if chainBaseID.Valid {
			cid := int(chainBaseID.Int32)
			v.ChainBaseID = &cid
		}
		vCopy := v
		versions = append(versions, &vCopy)
	}

	var pairs []models.VersionPair
	for i := 0; i+1 < len(versions); i++ {
		pairs = append(pairs, models.VersionPair{Left: versions[i], Right: versions[i+1]})
	}
	return pairs, nil
}

func (r *versionRepository) GetLastDate(ctx context.Context, deviceID int) (string, error) {
	var lastDate sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT DATE_FORMAT(MAX(created_at), '%Y-%m-%d') FROM config_versions WHERE device_id = ?
	`, deviceID).Scan(&lastDate)
	if err != nil || !lastDate.Valid {
		return "", err
	}
	return lastDate.String, nil
}

func (r *versionRepository) GetLatestForDevice(ctx context.Context, deviceID int) (*models.ConfigVersion, error) {
	var v models.ConfigVersion
	var parentID, chainBaseID sql.NullInt32
	err := r.db.QueryRowContext(ctx, `
		SELECT id, device_id, version_hash, storage_type, storage_path,
		       parent_version_id, chain_base_id, chain_position,
		       original_size, compressed_size, created_at
		FROM config_versions
		WHERE device_id = ?
		ORDER BY id DESC
		LIMIT 1`,
		deviceID,
	).Scan(&v.ID, &v.DeviceID, &v.VersionHash, &v.StorageType, &v.StoragePath,
		&parentID, &chainBaseID, &v.ChainPosition, &v.OriginalSize, &v.CompressedSize, &v.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		pid := int(parentID.Int32)
		v.ParentVersionID = &pid
	}
	if chainBaseID.Valid {
		cid := int(chainBaseID.Int32)
		v.ChainBaseID = &cid
	}
	return &v, nil
}

func (r *versionRepository) ResolveByDate(ctx context.Context, deviceID int, date1, date2 string) (int, int, error) {
	for _, d := range []string{date1, date2} {
		if len(d) != 10 || d[4] != '-' || d[7] != '-' {
			return 0, 0, nil
		}
	}

	var v1ID, v2ID sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		SELECT (SELECT id FROM config_versions WHERE device_id = ? AND DATE(created_at) = ? ORDER BY created_at DESC LIMIT 1) AS v1,
		       (SELECT id FROM config_versions WHERE device_id = ? AND DATE(created_at) = ? ORDER BY created_at DESC LIMIT 1) AS v2`,
		deviceID, date1, deviceID, date2,
	).Scan(&v1ID, &v2ID)
	if err != nil {
		return 0, 0, err
	}

	if !v1ID.Valid || !v2ID.Valid {
		return 0, 0, nil
	}
	return int(v1ID.Int64), int(v2ID.Int64), nil
}
