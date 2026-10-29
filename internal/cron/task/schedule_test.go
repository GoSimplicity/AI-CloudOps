package task

import (
	"strings"
	"testing"

	"github.com/hibiken/asynq"
)

func TestParseScheduleMatchesAsynqRegistration(t *testing.T) {
	tests := []struct {
		name, expression string
		valid            bool
	}{
		{"five fields", "*/5 1-20/2 * * MON-FRI", true},
		{"descriptor", "@daily", true},
		{"duration", "@every 1h30m", true},
		{"zero duration follows sdk", "@every 0s", true},
		{"explicit timezone", "CRON_TZ=Asia/Shanghai 0 9 * * *", true},
		{"six fields rejected", "0 0 9 * * *", false},
		{"out of range", "61 * * * *", false},
		{"invalid descriptor", "@sometimes", false},
		{"empty", "", false},
		{"malformed timezone", "CRON_TZ=Asia/Shanghai", false},
		{"malformed tz", "TZ=UTC", false},
		{"database length limit", strings.Repeat(" ", 101) + "* * * * *", false},
	}
	scheduler := asynq.NewScheduler(asynq.RedisClientOpt{Addr: "127.0.0.1:0"}, nil)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseSchedule(test.expression)
			if (err == nil) != test.valid {
				t.Fatalf("ParseSchedule(%q): err=%v", test.expression, err)
			}
			// 时区缺表达式会触发上游panic；长度限制来自数据库字段
			if test.name == "malformed timezone" || test.name == "malformed tz" || test.name == "database length limit" {
				return
			}
			entryID, err := scheduler.Register(test.expression, asynq.NewTask("test", nil))
			if (err == nil) != test.valid {
				t.Fatalf("SDK Register(%q): err=%v", test.expression, err)
			}
			if err == nil {
				if err := scheduler.Unregister(entryID); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
