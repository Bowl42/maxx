package sqlite

import (
	"time"

	"github.com/awsl-project/maxx/internal/domain"
	"gorm.io/gorm"
)

type AntigravityQuotaRepository struct {
	db *DB
}

func NewAntigravityQuotaRepository(d *DB) *AntigravityQuotaRepository {
	return &AntigravityQuotaRepository{db: d}
}

func (r *AntigravityQuotaRepository) Upsert(quota *domain.AntigravityQuota) error {
	now := time.Now()

	// Use FirstOrCreate to handle both insert and update properly
	// This ensures all fields including zeros are saved correctly
	model := &AntigravityQuota{
		Email:            quota.Email,
		Name:             quota.Name,
		Picture:          LongText(quota.Picture),
		GCPProjectID:     quota.GCPProjectID,
		SubscriptionTier: quota.SubscriptionTier,
		IsForbidden:      boolToInt(quota.IsForbidden),
		Models:           LongText(toJSON(quota.Models)),
		CreatedAt:        toTimestamp(now),
		UpdatedAt:        toTimestamp(now),
		DeletedAt:        0,
	}

	// FirstOrCreate with proper conflict handling
	result := r.db.gorm.FirstOrCreate(model, &AntigravityQuota{Email: quota.Email})
	if result.Error != nil {
		return result.Error
	}

	// If record already existed, update all fields explicitly (including zeros)
	if result.RowsAffected == 0 {
		updates := map[string]any{
			"updated_at":        toTimestamp(now),
			"name":              quota.Name,
			"picture":           LongText(quota.Picture),
			"gcp_project_id":    quota.GCPProjectID,
			"subscription_tier": quota.SubscriptionTier,
			"is_forbidden":      boolToInt(quota.IsForbidden),
			"models":            LongText(toJSON(quota.Models)),
		}
		if err := r.db.gorm.Model(&AntigravityQuota{}).
			Where("email = ? AND deleted_at = 0", quota.Email).
			Updates(updates).Error; err != nil {
			return err
		}
	}

	quota.ID = model.ID
	quota.CreatedAt = fromTimestamp(model.CreatedAt)
	quota.UpdatedAt = now

	return nil
}

func (r *AntigravityQuotaRepository) GetByEmail(email string) (*domain.AntigravityQuota, error) {
	var model AntigravityQuota
	err := r.db.gorm.Where("email = ? AND deleted_at = 0", email).First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&model), nil
}

func (r *AntigravityQuotaRepository) List() ([]*domain.AntigravityQuota, error) {
	var models []AntigravityQuota
	if err := r.db.gorm.Where("deleted_at = 0").Order("updated_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	return r.toDomainList(models), nil
}

func (r *AntigravityQuotaRepository) Delete(email string) error {
	now := time.Now().UnixMilli()
	return r.db.gorm.Model(&AntigravityQuota{}).
		Where("email = ?", email).
		Updates(map[string]any{
			"deleted_at": now,
			"updated_at": now,
		}).Error
}

func (r *AntigravityQuotaRepository) toModel(q *domain.AntigravityQuota) *AntigravityQuota {
	return &AntigravityQuota{
		SoftDeleteModel: SoftDeleteModel{
			BaseModel: BaseModel{
				ID:        q.ID,
				CreatedAt: toTimestamp(q.CreatedAt),
				UpdatedAt: toTimestamp(q.UpdatedAt),
			},
			DeletedAt: toTimestampPtr(q.DeletedAt),
		},
		Email:            q.Email,
		Name:             q.Name,
		Picture:          LongText(q.Picture),
		GCPProjectID:     q.GCPProjectID,
		SubscriptionTier: q.SubscriptionTier,
		IsForbidden:      boolToInt(q.IsForbidden),
		Models:           LongText(toJSON(q.Models)),
	}
}

func (r *AntigravityQuotaRepository) toDomain(m *AntigravityQuota) *domain.AntigravityQuota {
	return &domain.AntigravityQuota{
		ID:               m.ID,
		CreatedAt:        fromTimestamp(m.CreatedAt),
		UpdatedAt:        fromTimestamp(m.UpdatedAt),
		DeletedAt:        fromTimestampPtr(m.DeletedAt),
		Email:            m.Email,
		Name:             m.Name,
		Picture:          string(m.Picture),
		GCPProjectID:     m.GCPProjectID,
		SubscriptionTier: m.SubscriptionTier,
		IsForbidden:      m.IsForbidden == 1,
		Models:           fromJSON[[]domain.AntigravityModelQuota](string(m.Models)),
	}
}

func (r *AntigravityQuotaRepository) toDomainList(models []AntigravityQuota) []*domain.AntigravityQuota {
	quotas := make([]*domain.AntigravityQuota, len(models))
	for i, m := range models {
		quotas[i] = r.toDomain(&m)
	}
	return quotas
}
