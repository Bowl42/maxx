package repository

import (
	"github.com/Bowl42/maxx/internal/domain"
	"time"
)

// CooldownRepository 接口
type CooldownRepository interface {
	// GetAll returns all active cooldowns
	GetAll() ([]*domain.Cooldown, error)

	// GetByProvider returns cooldowns for a specific provider
	GetByProvider(providerID uint64) ([]*domain.Cooldown, error)

	// Upsert creates or updates a cooldown
	Upsert(cooldown *domain.Cooldown) error

	// Delete removes a cooldown
	Delete(providerID uint64, clientType string) error

	// DeleteAll removes all cooldowns for a provider
	DeleteAll(providerID uint64) error

	// DeleteExpired removes all expired cooldowns
	DeleteExpired() error

	// Get retrieves a specific cooldown
	Get(providerID uint64, clientType string) (*domain.Cooldown, error)
}

// CooldownInfo is a helper structure for returning cooldown information
type CooldownInfo struct {
	ProviderID   uint64    `json:"providerID"`
	ProviderName string    `json:"providerName"`
	ClientType   string    `json:"clientType"`
	Until        time.Time `json:"until"`
	Remaining    string    `json:"remaining"`
}
