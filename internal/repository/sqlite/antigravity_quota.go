package sqlite

import (
	"database/sql"
	"time"

	"github.com/awsl-project/maxx/internal/domain"
)

type AntigravityQuotaRepository struct {
	db *sql.DB
}

func NewAntigravityQuotaRepository(d *DB) *AntigravityQuotaRepository {
	return &AntigravityQuotaRepository{db: d.db}
}

func (r *AntigravityQuotaRepository) Upsert(quota *domain.AntigravityQuota) error {
	now := time.Now()
	modelsJSON := toJSON(quota.Models)

	// Try to update first
	result, err := r.db.Exec(`
		UPDATE antigravity_quotas
		SET updated_at = ?, name = ?, picture = ?, project_id = ?, subscription_tier = ?, is_forbidden = ?, models = ?, last_updated = ?
		WHERE email = ?
	`, now, quota.Name, quota.Picture, quota.ProjectID, quota.SubscriptionTier, quota.IsForbidden, modelsJSON, quota.LastUpdated, quota.Email)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// If no rows updated, insert new record
	if rowsAffected == 0 {
		result, err = r.db.Exec(`
			INSERT INTO antigravity_quotas (created_at, updated_at, email, name, picture, project_id, subscription_tier, is_forbidden, models, last_updated)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, now, now, quota.Email, quota.Name, quota.Picture, quota.ProjectID, quota.SubscriptionTier, quota.IsForbidden, modelsJSON, quota.LastUpdated)
		if err != nil {
			return err
		}
		id, _ := result.LastInsertId()
		quota.ID = uint64(id)
		quota.CreatedAt = now
	}
	quota.UpdatedAt = now

	return nil
}

func (r *AntigravityQuotaRepository) GetByEmail(email string) (*domain.AntigravityQuota, error) {
	row := r.db.QueryRow(`
		SELECT id, created_at, updated_at, email, name, picture, project_id, subscription_tier, is_forbidden, models, last_updated
		FROM antigravity_quotas
		WHERE email = ?
	`, email)

	return r.scanQuota(row)
}

func (r *AntigravityQuotaRepository) List() ([]*domain.AntigravityQuota, error) {
	rows, err := r.db.Query(`
		SELECT id, created_at, updated_at, email, name, picture, project_id, subscription_tier, is_forbidden, models, last_updated
		FROM antigravity_quotas
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	quotas := make([]*domain.AntigravityQuota, 0)
	for rows.Next() {
		quota, err := r.scanQuotaRow(rows)
		if err != nil {
			return nil, err
		}
		quotas = append(quotas, quota)
	}

	return quotas, rows.Err()
}

func (r *AntigravityQuotaRepository) Delete(email string) error {
	_, err := r.db.Exec(`DELETE FROM antigravity_quotas WHERE email = ?`, email)
	return err
}

func (r *AntigravityQuotaRepository) scanQuota(row *sql.Row) (*domain.AntigravityQuota, error) {
	var quota domain.AntigravityQuota
	var createdAt, updatedAt time.Time
	var modelsJSON string

	err := row.Scan(
		&quota.ID,
		&createdAt,
		&updatedAt,
		&quota.Email,
		&quota.Name,
		&quota.Picture,
		&quota.ProjectID,
		&quota.SubscriptionTier,
		&quota.IsForbidden,
		&modelsJSON,
		&quota.LastUpdated,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	quota.CreatedAt = createdAt
	quota.UpdatedAt = updatedAt
	quota.Models = fromJSON[[]domain.AntigravityModelQuota](modelsJSON)

	return &quota, nil
}

func (r *AntigravityQuotaRepository) scanQuotaRow(rows *sql.Rows) (*domain.AntigravityQuota, error) {
	var quota domain.AntigravityQuota
	var createdAt, updatedAt time.Time
	var modelsJSON string

	err := rows.Scan(
		&quota.ID,
		&createdAt,
		&updatedAt,
		&quota.Email,
		&quota.Name,
		&quota.Picture,
		&quota.ProjectID,
		&quota.SubscriptionTier,
		&quota.IsForbidden,
		&modelsJSON,
		&quota.LastUpdated,
	)
	if err != nil {
		return nil, err
	}

	quota.CreatedAt = createdAt
	quota.UpdatedAt = updatedAt
	quota.Models = fromJSON[[]domain.AntigravityModelQuota](modelsJSON)

	return &quota, nil
}
