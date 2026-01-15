package sqlite

import (
	"database/sql"
	"time"

	"github.com/awsl-project/maxx/internal/domain"
)

type RetryConfigRepository struct {
	db *DB
}

func NewRetryConfigRepository(db *DB) *RetryConfigRepository {
	return &RetryConfigRepository{db: db}
}

func (r *RetryConfigRepository) Create(c *domain.RetryConfig) error {
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now

	isDefault := 0
	if c.IsDefault {
		isDefault = 1
	}

	result, err := r.db.db.Exec(
		`INSERT INTO retry_configs (created_at, updated_at, name, is_default, max_retries, initial_interval_ms, backoff_rate, max_interval_ms) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.CreatedAt, c.UpdatedAt, c.Name, isDefault, c.MaxRetries, c.InitialInterval.Milliseconds(), c.BackoffRate, c.MaxInterval.Milliseconds(),
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	c.ID = uint64(id)
	return nil
}

func (r *RetryConfigRepository) Update(c *domain.RetryConfig) error {
	c.UpdatedAt = time.Now()
	isDefault := 0
	if c.IsDefault {
		isDefault = 1
	}
	_, err := r.db.db.Exec(
		`UPDATE retry_configs SET updated_at = ?, name = ?, is_default = ?, max_retries = ?, initial_interval_ms = ?, backoff_rate = ?, max_interval_ms = ? WHERE id = ?`,
		c.UpdatedAt, c.Name, isDefault, c.MaxRetries, c.InitialInterval.Milliseconds(), c.BackoffRate, c.MaxInterval.Milliseconds(), c.ID,
	)
	return err
}

func (r *RetryConfigRepository) Delete(id uint64) error {
	_, err := r.db.db.Exec(`DELETE FROM retry_configs WHERE id = ?`, id)
	return err
}

func (r *RetryConfigRepository) GetByID(id uint64) (*domain.RetryConfig, error) {
	row := r.db.db.QueryRow(`SELECT id, created_at, updated_at, name, is_default, max_retries, initial_interval_ms, backoff_rate, max_interval_ms FROM retry_configs WHERE id = ?`, id)
	return r.scanConfig(row)
}

func (r *RetryConfigRepository) GetDefault() (*domain.RetryConfig, error) {
	row := r.db.db.QueryRow(`SELECT id, created_at, updated_at, name, is_default, max_retries, initial_interval_ms, backoff_rate, max_interval_ms FROM retry_configs WHERE is_default = 1 LIMIT 1`)
	return r.scanConfig(row)
}

func (r *RetryConfigRepository) List() ([]*domain.RetryConfig, error) {
	rows, err := r.db.db.Query(`SELECT id, created_at, updated_at, name, is_default, max_retries, initial_interval_ms, backoff_rate, max_interval_ms FROM retry_configs ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := make([]*domain.RetryConfig, 0)
	for rows.Next() {
		c, err := r.scanConfigRows(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

func (r *RetryConfigRepository) scanConfig(row *sql.Row) (*domain.RetryConfig, error) {
	var c domain.RetryConfig
	var isDefault int
	var initialMs, maxMs int64
	err := row.Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt, &c.Name, &isDefault, &c.MaxRetries, &initialMs, &c.BackoffRate, &maxMs)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	c.IsDefault = isDefault == 1
	c.InitialInterval = time.Duration(initialMs) * time.Millisecond
	c.MaxInterval = time.Duration(maxMs) * time.Millisecond
	return &c, nil
}

func (r *RetryConfigRepository) scanConfigRows(rows *sql.Rows) (*domain.RetryConfig, error) {
	var c domain.RetryConfig
	var isDefault int
	var initialMs, maxMs int64
	err := rows.Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt, &c.Name, &isDefault, &c.MaxRetries, &initialMs, &c.BackoffRate, &maxMs)
	if err != nil {
		return nil, err
	}
	c.IsDefault = isDefault == 1
	c.InitialInterval = time.Duration(initialMs) * time.Millisecond
	c.MaxInterval = time.Duration(maxMs) * time.Millisecond
	return &c, nil
}
