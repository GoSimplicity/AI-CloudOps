package model

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

// job_type 不得有列默认值：漏传类型时应由 NOT NULL 报错，而不是静默落成命令行(2)任务。
// 注意：去掉 default 后，启动时 AutoMigrate 会对已有表执行 MODIFY COLUMN 去掉 DEFAULT 2，
// 见 docs/release/cron-job-type-default.md
func TestCronJobTypeHasNoColumnDefault(t *testing.T) {
	s, err := schema.Parse(&CronJob{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	field := s.LookUpField("job_type")
	if field == nil {
		t.Fatal("缺少 job_type 字段")
	}
	if field.HasDefaultValue || field.DefaultValue != "" {
		t.Fatalf("job_type 不应声明列默认值, got %q", field.DefaultValue)
	}
	if !field.NotNull {
		t.Fatal("job_type 必须保持 NOT NULL")
	}
}
