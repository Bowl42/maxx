package repository

import (
	"github.com/Bowl42/maxx/internal/domain"
)

// FailureCountRepository manages failure count persistence
type FailureCountRepository interface {
	// Get retrieves a failure count by provider, client type, and reason
	Get(providerID uint64, clientType string, reason string) (*domain.FailureCount, error)

	// GetAll retrieves all failure counts
	GetAll() ([]*domain.FailureCount, error)

	// Upsert inserts or updates a failure count
	Upsert(fc *domain.FailureCount) error

	// Delete deletes a failure count
	Delete(providerID uint64, clientType string, reason string) error

	// DeleteAll deletes all failure counts for a provider+clientType
	DeleteAll(providerID uint64, clientType string) error

	// DeleteExpired deletes failure counts where last failure was too long ago
	// (e.g., if no failures in last 24 hours, reset the count)
	DeleteExpired(olderThan int64) error
}
