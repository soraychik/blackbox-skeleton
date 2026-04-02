package repository

import (
	"context"
	"database/sql"

	"blackbox-api/internal/models"
)

type diffRepository struct {
	db *sql.DB
}

func NewDiffRepository(db *sql.DB) DiffRepository {
	return &diffRepository{db: db}
}

func (r *diffRepository) GetIndex(ctx context.Context, leftID, rightID int) (*models.DiffIndex, error) {
	var diffIndex models.DiffIndex
	err := r.db.QueryRowContext(ctx, `
		SELECT id, left_version_id, right_version_id, added_lines, removed_lines, diff_content, created_at 
		FROM diff_index 
		WHERE left_version_id = ? AND right_version_id = ?`,
		leftID, rightID,
	).Scan(&diffIndex.ID, &diffIndex.LeftVersionID, &diffIndex.RightVersionID,
		&diffIndex.AddedLines, &diffIndex.RemovedLines, &diffIndex.DiffContent, &diffIndex.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &diffIndex, nil
}

func (r *diffRepository) SaveIndex(ctx context.Context, leftID, rightID, added, removed int, diff string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO diff_index (left_version_id, right_version_id, added_lines, removed_lines, diff_content) 
		VALUES (?, ?, ?, ?, ?)`,
		leftID, rightID, added, removed, diff,
	)
	return err
}
