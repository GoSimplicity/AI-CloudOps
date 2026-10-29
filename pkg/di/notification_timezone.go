package di

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// notificationTimeZone rejects machine-dependent or implicit MySQL DATETIME interpretation.
func notificationTimeZone(dsn string) (string, error) {
	databasePart := dsn[strings.LastIndex(dsn, "/")+1:]
	_, raw, found := strings.Cut(databasePart, "?")
	if !found {
		return "", errors.New("MySQL DSN 必须显式设置固定 loc")
	}
	query, err := url.ParseQuery(raw)
	if err != nil || len(query["loc"]) != 1 {
		return "", errors.New("MySQL DSN 必须显式设置固定 loc")
	}
	name := query.Get("loc")
	if name == "" || strings.EqualFold(name, "local") {
		return "", errors.New("MySQL loc 不允许 Local 或空值，请使用 UTC 或明确的 IANA 时区")
	}
	if _, err := time.LoadLocation(name); err != nil {
		return "", errors.New("MySQL loc 无效")
	}
	return name, nil
}

// CheckNotificationTimeZone never converts historical DATETIME values. A database
// with legacy notifications requires an operator to verify or migrate the data
// before explicitly recording the chosen zone in the clock table.
func CheckNotificationTimeZone(db *gorm.DB, zone string) error {
	if db == nil || zone == "" {
		return errors.New("通知时区配置缺失")
	}
	if err := db.AutoMigrate(&model.WorkorderNotificationClock{}); err != nil {
		return err
	}
	read := func() (*model.WorkorderNotificationClock, error) {
		var marker model.WorkorderNotificationClock
		err := db.First(&marker, 1).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return &marker, nil
	}
	marker, err := read()
	if err != nil {
		return err
	}
	if marker == nil {
		var legacy int64
		for _, check := range []struct {
			table string
			query any
			args  []any
		}{
			{"cl_workorder_notification_event", "1 = 1", nil},
			{"cl_workorder_notification", "scheduled_time IS NOT NULL", nil},
		} {
			var probe int
			result := db.Table(check.table).Select("1").Where(check.query, check.args...).Limit(1).Scan(&probe)
			err := result.Error
			if err != nil {
				var mysqlErr *mysqlDriver.MySQLError
				if db.Dialector.Name() == "mysql" && errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 {
					continue
				}
				if db.Dialector.Name() == "sqlite" {
					exists, existsErr := notificationTableExists(db, check.table)
					if existsErr != nil {
						return existsErr
					}
					if !exists {
						continue
					}
				}
				return err
			}
			legacy += result.RowsAffected
		}
		// Another new instance may have registered the marker meanwhile.
		marker, err = read()
		if err != nil {
			return err
		}
		if marker == nil && legacy > 0 {
			return errors.New("存在历史通知时间数据，请先停写、备份并核对迁移后登记时区")
		}
		if marker == nil {
			if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.WorkorderNotificationClock{ID: 1, TimeZone: zone}).Error; err != nil {
				return err
			}
			marker, err = read()
			if err != nil {
				return err
			}
		}
	}
	if marker == nil || marker.TimeZone != zone {
		return fmt.Errorf("通知数据库时区与实例 MySQL loc 不一致")
	}
	return nil
}

func notificationTableExists(db *gorm.DB, table string) (bool, error) {
	var count int64
	var result *gorm.DB
	switch db.Dialector.Name() {
	case "sqlite":
		result = db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count)
	default:
		return false, errors.New("不支持的通知数据库类型")
	}
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}
