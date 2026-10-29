package dao

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeliveryMasksMatchPersistedPlanAndNeverExposeSecrets(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mask.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.AutoMigrate(&model.WorkorderNotificationEvent{}, &model.WorkorderNotificationDelivery{}); err != nil {
		t.Fatal(err)
	}
	plan := `[ {"config":{"id":5},"channel":"email","recipients":[{"address":"alice.private@example.test"}],"subject":"subject-canary","content":"body-canary"},{"config":{"id":7},"channel":"feishu","recipients":[{"address":"ou_privateidentity0123"}],"content":"token-canary"} ]`
	event := model.WorkorderNotificationEvent{Status: "uncertain", Snapshot: []byte("snapshot-canary"), Plan: []byte(plan), NextAttemptAt: time.Now().UTC()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	hash := func(address string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(address))) }
	for _, item := range []struct {
		id               int
		channel, address string
	}{
		{5, "email", "alice.private@example.test"}, {7, "feishu", "ou_privateidentity0123"},
		{6, "email", "alice.private@example.test"}, {7, "email", "ou_privateidentity0123"}, {5, "email", "nobody@example.test"},
	} {
		if err := db.Create(&model.WorkorderNotificationDelivery{EventID: event.ID, NotificationID: item.id, Channel: item.channel, RecipientHash: hash(item.address), Status: "uncertain"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	store := NewWorkorderInstanceDAO(db, zap.NewNop()).(*workorderInstanceDAO)
	rows, err := store.ListNotificationEventDeliveries(context.Background(), event.ID)
	if err != nil || len(rows) != 5 {
		t.Fatalf("rows=%d err=%v", len(rows), err)
	}
	if rows[0].RecipientMasked != "a***@e***" || rows[1].RecipientMasked != "ou_***0123" {
		t.Fatal("expected safe matching masks")
	}
	for _, row := range rows[2:] {
		if row.RecipientMasked != "" {
			t.Fatal("unmatched recipient displayed")
		}
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"alice.private@example.test", "ou_privateidentity0123", "subject-canary", "body-canary", "snapshot-canary", "token-canary"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatal("delivery response leaked private plan data")
		}
	}
	for _, raw := range [][]byte{nil, []byte("{bad"), []byte(strings.Repeat("x", 1048577))} {
		if err := db.Model(&event).Update("plan", raw).Error; err != nil {
			t.Fatal(err)
		}
		rows, err = store.ListNotificationEventDeliveries(context.Background(), event.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range rows {
			if row.RecipientMasked != "" {
				t.Fatal("missing, malformed or oversized plan returned mask")
			}
		}
	}
}

func TestMaskShortIDsAndUntrustedAddressAreFailClosed(t *testing.T) {
	if got := maskNotificationRecipient("feishu", "ou_a"); got != "ou_***" {
		t.Fatal("short identifier revealed")
	}
	for _, plan := range []string{
		`[{"config":{"id":1},"channel":"feishu","recipients":[{"address":"https://example.test/hook/private"}]}]`,
		`[{"config":{"id":1},"channel":"email","recipients":[{"address":"Name <person@example.test>"}]}]`,
		`[{"config":{"id":1},"channel":"email","recipients":[{"address":"UPPER@example.test"}]}]`,
	} {
		if len(maskedPlanRecipients([]byte(plan))) != 0 {
			t.Fatal("invalid or noncanonical recipient accepted")
		}
	}
}

func TestConcurrentDeliveryReadsAndPlanClear(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mask-concurrent.db")+"?_busy_timeout=5000&_journal_mode=WAL"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.AutoMigrate(&model.WorkorderNotificationEvent{}, &model.WorkorderNotificationDelivery{}); err != nil {
		t.Fatal(err)
	}
	address := "alice.private@example.test"
	event := model.WorkorderNotificationEvent{Status: "uncertain", Snapshot: []byte("snapshot-canary"), Plan: []byte(`[{"config":{"id":5},"channel":"email","recipients":[{"address":"alice.private@example.test"}]}]`), NextAttemptAt: time.Now().UTC()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.WorkorderNotificationDelivery{EventID: event.ID, NotificationID: 5, Channel: "email", RecipientHash: fmt.Sprintf("%x", sha256.Sum256([]byte(address))), Status: "uncertain"}).Error; err != nil {
		t.Fatal(err)
	}
	store := NewWorkorderInstanceDAO(db, zap.NewNop()).(*workorderInstanceDAO)
	start := make(chan struct{})
	results := make(chan error, 21)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			rows, err := store.ListNotificationEventDeliveries(context.Background(), event.ID)
			if err == nil && (len(rows) != 1 || (rows[0].RecipientMasked != "" && rows[0].RecipientMasked != "a***@e***")) {
				err = fmt.Errorf("inconsistent mask")
			}
			results <- err
		}()
	}
	wg.Add(1)
	go func() { defer wg.Done(); <-start; results <- db.Model(&event).Update("plan", nil).Error }()
	close(start)
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	rows, err := store.ListNotificationEventDeliveries(context.Background(), event.ID)
	if err != nil || len(rows) != 1 || rows[0].RecipientMasked != "" {
		t.Fatal("mask survived completed plan clearing")
	}
}
