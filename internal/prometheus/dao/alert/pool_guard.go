package alert

import (
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Lock the target AM parent before any child or association writes. Deletion
// acquires the same lock before checking every undeleted send-group reference.
// Leaving a source pool cannot introduce a reference, so only the target parent
// is required; migrations never hold two parent locks in opposing orders.
func lockLiveAlertManagerPool(tx *gorm.DB, poolID int) error {
	if poolID <= 0 {
		return fmt.Errorf("AlertManager池ID必须大于0")
	}
	var pool model.MonitorAlertManagerPool
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&pool, poolID).Error; err != nil {
		return fmt.Errorf("目标AlertManager池不存在或已删除: %w", err)
	}
	return nil
}
