package sqlite

import (
    "database/sql"
    "time"

    "github.com/awsl-project/maxx/internal/domain"
)

type RoutingStrategyRepository struct {
    db *DB
}

func NewRoutingStrategyRepository(db *DB) *RoutingStrategyRepository {
    return &RoutingStrategyRepository{db: db}
}

func (r *RoutingStrategyRepository) Create(s *domain.RoutingStrategy) error {
    now := time.Now()
    s.CreatedAt = now
    s.UpdatedAt = now

    result, err := r.db.db.Exec(
        `INSERT INTO routing_strategies (created_at, updated_at, project_id, type, config) VALUES (?, ?, ?, ?, ?)`,
        s.CreatedAt, s.UpdatedAt, s.ProjectID, s.Type, toJSON(s.Config),
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

func (r *RoutingStrategyRepository) Update(s *domain.RoutingStrategy) error {
    s.UpdatedAt = time.Now()
    _, err := r.db.db.Exec(
        `UPDATE routing_strategies SET updated_at = ?, project_id = ?, type = ?, config = ? WHERE id = ?`,
        s.UpdatedAt, s.ProjectID, s.Type, toJSON(s.Config), s.ID,
    )
    return err
}

func (r *RoutingStrategyRepository) Delete(id uint64) error {
    _, err := r.db.db.Exec(`DELETE FROM routing_strategies WHERE id = ?`, id)
    return err
}

func (r *RoutingStrategyRepository) GetByProjectID(projectID uint64) (*domain.RoutingStrategy, error) {
    row := r.db.db.QueryRow(`SELECT id, created_at, updated_at, project_id, type, config FROM routing_strategies WHERE project_id = ?`, projectID)
    var s domain.RoutingStrategy
    var configJSON string
    err := row.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt, &s.ProjectID, &s.Type, &configJSON)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, domain.ErrNotFound
        }
        return nil, err
    }
    s.Config = fromJSON[*domain.RoutingStrategyConfig](configJSON)
    return &s, nil
}

func (r *RoutingStrategyRepository) List() ([]*domain.RoutingStrategy, error) {
    rows, err := r.db.db.Query(`SELECT id, created_at, updated_at, project_id, type, config FROM routing_strategies ORDER BY id`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    strategies := make([]*domain.RoutingStrategy, 0)
    for rows.Next() {
        var s domain.RoutingStrategy
        var configJSON string
        err := rows.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt, &s.ProjectID, &s.Type, &configJSON)
        if err != nil {
            return nil, err
        }
        s.Config = fromJSON[*domain.RoutingStrategyConfig](configJSON)
        strategies = append(strategies, &s)
    }
    return strategies, rows.Err()
}
