package sqlite

import (
	"database/sql"
	"time"

	"github.com/awsl-project/maxx/internal/domain"
)

type SessionRepository struct {
	db *DB
}

func NewSessionRepository(db *DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(s *domain.Session) error {
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now

	result, err := r.db.db.Exec(
		`INSERT INTO sessions (created_at, updated_at, session_id, client_type, project_id, rejected_at) VALUES (?, ?, ?, ?, ?, ?)`,
		s.CreatedAt, s.UpdatedAt, s.SessionID, s.ClientType, s.ProjectID, formatTimePtr(s.RejectedAt),
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	s.ID = uint64(id)
	return nil
}

func (r *SessionRepository) Update(s *domain.Session) error {
	s.UpdatedAt = time.Now()
	_, err := r.db.db.Exec(
		`UPDATE sessions SET updated_at = ?, client_type = ?, project_id = ?, rejected_at = ? WHERE id = ?`,
		s.UpdatedAt, s.ClientType, s.ProjectID, formatTimePtr(s.RejectedAt), s.ID,
	)
	return err
}

func (r *SessionRepository) GetBySessionID(sessionID string) (*domain.Session, error) {
	row := r.db.db.QueryRow(`SELECT id, created_at, updated_at, session_id, client_type, project_id, rejected_at FROM sessions WHERE session_id = ?`, sessionID)
	var s domain.Session
	var rejectedAt sql.NullString
	err := row.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt, &s.SessionID, &s.ClientType, &s.ProjectID, &rejectedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if rejectedAt.Valid && rejectedAt.String != "" {
		if t, err := parseTimeString(rejectedAt.String); err == nil && !t.IsZero() {
			s.RejectedAt = &t
		}
	}
	return &s, nil
}

func (r *SessionRepository) List() ([]*domain.Session, error) {
	rows, err := r.db.db.Query(`SELECT id, created_at, updated_at, session_id, client_type, project_id, rejected_at FROM sessions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]*domain.Session, 0)
	for rows.Next() {
		var s domain.Session
		var rejectedAt sql.NullString
		err := rows.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt, &s.SessionID, &s.ClientType, &s.ProjectID, &rejectedAt)
		if err != nil {
			return nil, err
		}
		if rejectedAt.Valid && rejectedAt.String != "" {
			if t, err := parseTimeString(rejectedAt.String); err == nil && !t.IsZero() {
				s.RejectedAt = &t
			}
		}
		sessions = append(sessions, &s)
	}
	return sessions, rows.Err()
}

// formatTimePtr formats a *time.Time for SQLite storage
func formatTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}
