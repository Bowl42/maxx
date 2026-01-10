package cooldown

import (
	"sync"
	"time"
)

// Manager manages provider cooldown states
// Cooldown is stored in memory and will be reset on restart
type Manager struct {
	mu        sync.RWMutex
	cooldowns map[uint64]time.Time // providerID -> cooldown end time
}

// NewManager creates a new cooldown manager
func NewManager() *Manager {
	return &Manager{
		cooldowns: make(map[uint64]time.Time),
	}
}

// Default global manager
var defaultManager = NewManager()

// Default returns the default global cooldown manager
func Default() *Manager {
	return defaultManager
}

// SetCooldown sets the cooldown end time for a provider
func (m *Manager) SetCooldown(providerID uint64, until time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cooldowns[providerID] = until
}

// SetCooldownDuration sets a cooldown for a provider with a duration from now
func (m *Manager) SetCooldownDuration(providerID uint64, duration time.Duration) {
	m.SetCooldown(providerID, time.Now().Add(duration))
}

// ClearCooldown removes the cooldown for a provider
func (m *Manager) ClearCooldown(providerID uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cooldowns, providerID)
}

// IsInCooldown checks if a provider is currently in cooldown
func (m *Manager) IsInCooldown(providerID uint64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	until, ok := m.cooldowns[providerID]
	if !ok {
		return false
	}

	// If cooldown has expired, it's not in cooldown
	return time.Now().Before(until)
}

// GetCooldownUntil returns the cooldown end time for a provider
// Returns zero time if not in cooldown
func (m *Manager) GetCooldownUntil(providerID uint64) time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	until, ok := m.cooldowns[providerID]
	if !ok {
		return time.Time{}
	}

	// If cooldown has expired, return zero time
	if time.Now().After(until) {
		return time.Time{}
	}

	return until
}

// GetAllCooldowns returns all active cooldowns (providerID -> end time)
func (m *Manager) GetAllCooldowns() map[uint64]time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	result := make(map[uint64]time.Time)

	for id, until := range m.cooldowns {
		if now.Before(until) {
			result[id] = until
		}
	}

	return result
}

// CleanupExpired removes expired cooldowns from memory
func (m *Manager) CleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, until := range m.cooldowns {
		if now.After(until) {
			delete(m.cooldowns, id)
		}
	}
}

// CooldownInfo represents cooldown information for API response
type CooldownInfo struct {
	ProviderID   uint64    `json:"providerID"`
	ProviderName string    `json:"providerName,omitempty"`
	Until        time.Time `json:"until"`
	Remaining    string    `json:"remaining"` // Human readable remaining time
}

// GetCooldownInfo returns cooldown info for a specific provider
func (m *Manager) GetCooldownInfo(providerID uint64, providerName string) *CooldownInfo {
	until := m.GetCooldownUntil(providerID)
	if until.IsZero() {
		return nil
	}

	remaining := time.Until(until)
	if remaining < 0 {
		return nil
	}

	return &CooldownInfo{
		ProviderID:   providerID,
		ProviderName: providerName,
		Until:        until,
		Remaining:    formatDuration(remaining),
	}
}

// formatDuration formats a duration as a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return d.Round(time.Second).String()
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		if seconds > 0 {
			return (time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second).String()
		}
		return (time.Duration(minutes) * time.Minute).String()
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if minutes > 0 {
		return (time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute).String()
	}
	return (time.Duration(hours) * time.Hour).String()
}
