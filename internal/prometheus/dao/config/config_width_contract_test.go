package config

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestCanonicalWebhookNamespaceWidthPreservesRawIP(t *testing.T) {
	const ip = "ffff:ffff:ffff:ffff:ffff:ffff:255.255.255.255"
	for _, group := range []string{"1", "9223372036854775807"} {
		t.Run(group, func(t *testing.T) {
			dao, db := newNamespaceWriteContractDB(t)
			key := ip + "_" + group
			row := namespaceWriteRow("synthetic webhook", key, model.ConfigTypeWebhookFile, 7, model.ConfigStatusInactive)
			if err := dao.CreateMonitorConfig(context.Background(), row); err != nil {
				t.Fatal("valid IPv6 webhook key rejected")
			}
			var saved model.MonitorConfig
			if db.First(&saved, row.ID).Error != nil || saved.InstanceIP != key || saved.Status != model.ConfigStatusInactive {
				t.Fatal("canonical webhook bytes/status changed")
			}
			var guard model.MonitorConfigNamespaceGuard
			if db.First(&guard, "config_type=? AND instance_key=?", model.ConfigTypeWebhookFile, []byte(key)).Error != nil || string(guard.InstanceKey) != key {
				t.Fatal("canonical namespace bytes changed")
			}
		})
	}
}
