package sqlite

import (
	"errors"
	"log"
	"sort"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// Migration 表示一个数据库迁移
type Migration struct {
	Version     int
	Description string
	Up          func(db *gorm.DB) error
	Down        func(db *gorm.DB) error
}

// 所有迁移按版本号注册
// 注意：GORM AutoMigrate 会自动处理新增列，这里只需要处理特殊情况（重命名、数据迁移等）
var migrations = []Migration{
	{
		Version:     1,
		Description: "Convert cost from microUSD to nanoUSD (multiply by 1000)",
		Up: func(db *gorm.DB) error {
			// Convert cost in proxy_requests table
			if err := db.Exec("UPDATE proxy_requests SET cost = cost * 1000 WHERE cost > 0").Error; err != nil {
				return err
			}
			// Convert cost in proxy_upstream_attempts table
			if err := db.Exec("UPDATE proxy_upstream_attempts SET cost = cost * 1000 WHERE cost > 0").Error; err != nil {
				return err
			}
			// Convert cost in usage_stats table
			if err := db.Exec("UPDATE usage_stats SET cost = cost * 1000 WHERE cost > 0").Error; err != nil {
				return err
			}
			return nil
		},
		Down: func(db *gorm.DB) error {
			// Rollback: divide by 1000
			if err := db.Exec("UPDATE proxy_requests SET cost = cost / 1000").Error; err != nil {
				return err
			}
			if err := db.Exec("UPDATE proxy_upstream_attempts SET cost = cost / 1000").Error; err != nil {
				return err
			}
			if err := db.Exec("UPDATE usage_stats SET cost = cost / 1000").Error; err != nil {
				return err
			}
			return nil
		},
	},
	{
		Version:     2,
		Description: "Add index on proxy_requests.provider_id",
		Up: func(db *gorm.DB) error {
			// 说明：这是高频列表/过滤路径的关键优化点。
			// 不同数据库方言对 IF NOT EXISTS 的支持不同，这里做最小兼容处理。
			switch db.Dialector.Name() {
			case "mysql":
				err := db.Exec("CREATE INDEX idx_proxy_requests_provider_id ON proxy_requests(provider_id)").Error
				if isMySQLDuplicateIndexError(err) {
					return nil
				}
				return err
			default:
				return db.Exec("CREATE INDEX IF NOT EXISTS idx_proxy_requests_provider_id ON proxy_requests(provider_id)").Error
			}
		},
		Down: func(db *gorm.DB) error {
			switch db.Dialector.Name() {
			case "mysql":
				// MySQL 不支持 DROP INDEX IF EXISTS；这里尽量执行，失败则忽略（回滚不是主路径）。
				sql := "DROP INDEX idx_proxy_requests_provider_id ON proxy_requests"
				if err := db.Exec(sql).Error; err != nil {
					log.Printf("[Migration] Warning: rollback v2 failed (dialector=mysql) sql=%q err=%v", sql, err)
				}
				return nil
			default:
				return db.Exec("DROP INDEX IF EXISTS idx_proxy_requests_provider_id").Error
			}
		},
	},
}

func isMySQLDuplicateIndexError(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1061 // ER_DUP_KEYNAME
	}
	// 兜底：错误可能被包装成字符串，避免使用过宽的 "duplicate" 匹配导致吞掉其它错误。
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "duplicate key name") || strings.Contains(lower, "error 1061")
}

// RunMigrations 运行所有待执行的迁移
func (d *DB) RunMigrations() error {
	// 确保迁移表存在（由 GORM AutoMigrate 处理）
	if err := d.gorm.AutoMigrate(&SchemaMigration{}); err != nil {
		return err
	}

	// 如果没有迁移，直接返回
	if len(migrations) == 0 {
		return nil
	}

	// 获取当前版本
	currentVersion := d.getCurrentVersion()

	// 按版本号排序迁移
	sortedMigrations := make([]Migration, len(migrations))
	copy(sortedMigrations, migrations)
	sort.Slice(sortedMigrations, func(i, j int) bool {
		return sortedMigrations[i].Version < sortedMigrations[j].Version
	})

	// 运行所有版本大于当前版本的迁移
	for _, m := range sortedMigrations {
		if m.Version <= currentVersion {
			continue
		}

		log.Printf("[Migration] Running migration v%d: %s", m.Version, m.Description)

		if err := d.runMigration(m); err != nil {
			log.Printf("[Migration] Failed migration v%d: %v", m.Version, err)
			return err
		}

		log.Printf("[Migration] Completed migration v%d", m.Version)
	}

	return nil
}

// getCurrentVersion 获取当前数据库版本
func (d *DB) getCurrentVersion() int {
	var maxVersion int
	d.gorm.Model(&SchemaMigration{}).Select("COALESCE(MAX(version), 0)").Scan(&maxVersion)
	return maxVersion
}

// runMigration 在事务中运行单个迁移
func (d *DB) runMigration(m Migration) error {
	// 注意：MySQL 的 DDL（如 CREATE/DROP INDEX）会触发隐式提交（implicit commit），
	// 这意味着即使这里用 gorm.Transaction 包裹，MySQL 路径也无法提供严格的“DDL + 迁移记录”原子性。
	//
	// 因此迁移实现必须尽量幂等：例如重复执行 CREATE INDEX 时，仅在 ER_DUP_KEYNAME(1061) 场景下视为成功。
	return d.gorm.Transaction(func(tx *gorm.DB) error {
		// 运行迁移
		if m.Up != nil {
			if err := m.Up(tx); err != nil {
				return err
			}
		}

		// 记录迁移
		return tx.Create(&SchemaMigration{
			Version:     m.Version,
			Description: m.Description,
			AppliedAt:   time.Now().UnixMilli(),
		}).Error
	})
}

// RollbackMigration 回滚到指定版本
func (d *DB) RollbackMigration(targetVersion int) error {
	currentVersion := d.getCurrentVersion()

	if targetVersion >= currentVersion {
		log.Printf("[Migration] Already at version %d, target is %d, nothing to rollback", currentVersion, targetVersion)
		return nil
	}

	// 按版本号降序排序
	sortedMigrations := make([]Migration, len(migrations))
	copy(sortedMigrations, migrations)
	sort.Slice(sortedMigrations, func(i, j int) bool {
		return sortedMigrations[i].Version > sortedMigrations[j].Version
	})

	// 回滚所有版本大于目标版本的迁移
	for _, m := range sortedMigrations {
		if m.Version <= targetVersion {
			break
		}
		if m.Version > currentVersion {
			continue
		}

		log.Printf("[Migration] Rolling back migration v%d: %s", m.Version, m.Description)

		if err := d.rollbackMigration(m); err != nil {
			log.Printf("[Migration] Failed rollback v%d: %v", m.Version, err)
			return err
		}

		log.Printf("[Migration] Rolled back migration v%d", m.Version)
	}

	return nil
}

// rollbackMigration 在事务中回滚单个迁移
func (d *DB) rollbackMigration(m Migration) error {
	// 同 runMigration：MySQL DDL 在回滚路径同样可能发生隐式提交，因此这里的事务主要用于把“回滚逻辑”
	// 与“删除迁移记录”尽量绑定在一起，但不应假设 MySQL 上能做到严格原子回滚。
	return d.gorm.Transaction(func(tx *gorm.DB) error {
		// 运行回滚
		if m.Down != nil {
			if err := m.Down(tx); err != nil {
				return err
			}
		}

		// 删除迁移记录
		return tx.Where("version = ?", m.Version).Delete(&SchemaMigration{}).Error
	})
}

// GetMigrationStatus 获取迁移状态
func (d *DB) GetMigrationStatus() ([]MigrationStatus, error) {
	// 获取已应用的迁移
	var applied []SchemaMigration
	if err := d.gorm.Find(&applied).Error; err != nil {
		return nil, err
	}

	appliedMap := make(map[int]int64)
	for _, m := range applied {
		appliedMap[m.Version] = m.AppliedAt
	}

	// 构建状态列表
	var statuses []MigrationStatus
	for _, m := range migrations {
		status := MigrationStatus{
			Version:     m.Version,
			Description: m.Description,
			Applied:     false,
		}
		if appliedAt, ok := appliedMap[m.Version]; ok {
			status.Applied = true
			status.AppliedAt = fromTimestamp(appliedAt)
		}
		statuses = append(statuses, status)
	}

	// 按版本号排序
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Version < statuses[j].Version
	})

	return statuses, nil
}

// MigrationStatus 迁移状态
type MigrationStatus struct {
	Version     int
	Description string
	Applied     bool
	AppliedAt   time.Time
}
