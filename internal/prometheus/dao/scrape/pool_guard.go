package scrape

import (
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LockLiveScrapePool serializes definition writes with parent deletion. Call it
// inside the write transaction, before acquiring any child row lock. Only the
// target parent is needed: leaving a source pool cannot introduce a reference.
func LockLiveScrapePool(tx *gorm.DB, poolID int) error {
	if poolID <= 0 {
		return fmt.Errorf("采集池 ID 必须大于 0")
	}
	var pool model.MonitorScrapePool
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&pool, poolID).Error; err != nil {
		return fmt.Errorf("目标采集池不存在或已删除: %w", err)
	}
	return nil
}
