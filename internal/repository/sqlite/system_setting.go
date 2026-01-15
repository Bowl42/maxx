package sqlite

import (
	"database/sql"
	"time"

	"github.com/awsl-project/maxx/internal/domain"
)

type SystemSettingRepository struct {
	db *DB
}

func NewSystemSettingRepository(db *DB) *SystemSettingRepository {
	return &SystemSettingRepository{db: db}
}

func (r *SystemSettingRepository) Get(key string) (string, error) {
	var value string
	err := r.db.db.QueryRow("SELECT value FROM system_settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (r *SystemSettingRepository) Set(key, value string) error {
	now := time.Now()
	_, err := r.db.db.Exec(`
		INSERT INTO system_settings (key, value, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = ?
	`, key, value, now, now, value, now)
	return err
}

func (r *SystemSettingRepository) GetAll() ([]*domain.SystemSetting, error) {
	rows, err := r.db.db.Query("SELECT key, value, created_at, updated_at FROM system_settings ORDER BY key")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make([]*domain.SystemSetting, 0)
	for rows.Next() {
		s := &domain.SystemSetting{}
		var createdAt, updatedAt sql.NullTime
		if err := rows.Scan(&s.Key, &s.Value, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		s.CreatedAt = parseTime(createdAt)
		s.UpdatedAt = parseTime(updatedAt)
		settings = append(settings, s)
	}
	return settings, nil
}

func (r *SystemSettingRepository) Delete(key string) error {
	_, err := r.db.db.Exec("DELETE FROM system_settings WHERE key = ?", key)
	return err
}
