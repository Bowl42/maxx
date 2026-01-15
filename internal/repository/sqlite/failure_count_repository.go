package sqlite

import (
	"database/sql"
	"time"

	"github.com/awsl-project/maxx/internal/domain"
	"github.com/awsl-project/maxx/internal/repository"
)

type FailureCountRepository struct {
	db *DB
}

func NewFailureCountRepository(db *DB) repository.FailureCountRepository {
	return &FailureCountRepository{db: db}
}

func (r *FailureCountRepository) Get(providerID uint64, clientType string, reason string) (*domain.FailureCount, error) {
	query := `SELECT id, created_at, updated_at, provider_id, client_type, reason, count, last_failure_at
	          FROM failure_counts
	          WHERE provider_id = ? AND client_type = ? AND reason = ?`

	fc := &domain.FailureCount{}
	var createdAt, updatedAt, lastFailureAt string

	err := r.db.db.QueryRow(query, providerID, clientType, reason).Scan(
		&fc.ID, &createdAt, &updatedAt, &fc.ProviderID, &fc.ClientType, &fc.Reason, &fc.Count, &lastFailureAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	fc.CreatedAt, _ = parseTimeString(createdAt)
	fc.UpdatedAt, _ = parseTimeString(updatedAt)
	fc.LastFailureAt, _ = parseTimeString(lastFailureAt)

	return fc, nil
}

func (r *FailureCountRepository) GetAll() ([]*domain.FailureCount, error) {
	query := `SELECT id, created_at, updated_at, provider_id, client_type, reason, count, last_failure_at
	          FROM failure_counts`

	rows, err := r.db.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	failureCounts := make([]*domain.FailureCount, 0)
	for rows.Next() {
		fc := &domain.FailureCount{}
		var createdAt, updatedAt, lastFailureAt string
		if err := rows.Scan(&fc.ID, &createdAt, &updatedAt, &fc.ProviderID, &fc.ClientType, &fc.Reason, &fc.Count, &lastFailureAt); err != nil {
			return nil, err
		}

		fc.CreatedAt, _ = parseTimeString(createdAt)
		fc.UpdatedAt, _ = parseTimeString(updatedAt)
		fc.LastFailureAt, _ = parseTimeString(lastFailureAt)
		failureCounts = append(failureCounts, fc)
	}

	return failureCounts, rows.Err()
}

func (r *FailureCountRepository) Upsert(fc *domain.FailureCount) error {
	now := time.Now().UTC()

	// Check if exists
	existing, err := r.Get(fc.ProviderID, fc.ClientType, fc.Reason)
	if err != nil {
		return err
	}

	if existing != nil {
		// Update
		query := `UPDATE failure_counts
		          SET count = ?, last_failure_at = ?, updated_at = ?
		          WHERE provider_id = ? AND client_type = ? AND reason = ?`

		_, err = r.db.db.Exec(query, fc.Count, formatTime(fc.LastFailureAt), formatTime(now), fc.ProviderID, fc.ClientType, fc.Reason)
		return err
	}

	// Insert
	query := `INSERT INTO failure_counts (provider_id, client_type, reason, count, last_failure_at, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.db.Exec(query,
		fc.ProviderID,
		fc.ClientType,
		fc.Reason,
		fc.Count,
		formatTime(fc.LastFailureAt),
		formatTime(now),
		formatTime(now),
	)

	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	fc.ID = uint64(id)
	fc.CreatedAt = now
	fc.UpdatedAt = now

	return nil
}

func (r *FailureCountRepository) Delete(providerID uint64, clientType string, reason string) error {
	query := `DELETE FROM failure_counts WHERE provider_id = ? AND client_type = ? AND reason = ?`
	_, err := r.db.db.Exec(query, providerID, clientType, reason)
	return err
}

func (r *FailureCountRepository) DeleteAll(providerID uint64, clientType string) error {
	query := `DELETE FROM failure_counts WHERE provider_id = ? AND client_type = ?`
	_, err := r.db.db.Exec(query, providerID, clientType)
	return err
}

func (r *FailureCountRepository) DeleteExpired(olderThanSeconds int64) error {
	// Delete failure counts where last_failure_at is older than the specified duration
	query := `DELETE FROM failure_counts WHERE last_failure_at < datetime('now', '-' || ? || ' seconds')`
	_, err := r.db.db.Exec(query, olderThanSeconds)
	return err
}
