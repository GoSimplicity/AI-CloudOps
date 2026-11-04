package model

import (
	"fmt"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/gin-gonic/gin/binding"
	gormschema "gorm.io/gorm/schema"
)

func TestMonitorConfigLiteralTagsMatchSharedLimits(t *testing.T) {
	for _, tc := range []struct {
		value  any
		field  string
		size   int
		binary bool
	}{
		{MonitorConfig{}, "Name", MonitorConfigNameMaxRunes, false},
		{MonitorConfig{}, "InstanceIP", MonitorConfigKeyMaxRunes, false},
		{MonitorConfigNamespaceGuard{}, "InstanceKey", MonitorConfigKeyMaxBytes, true},
		{MonitorConfigArtifactEvent{}, "InstanceKey", MonitorConfigKeyMaxBytes, true},
	} {
		t.Run(fmt.Sprintf("%T/%s", tc.value, tc.field), func(t *testing.T) {
			parsed, err := gormschema.Parse(tc.value, &sync.Map{}, gormschema.NamingStrategy{})
			if err != nil {
				t.Fatal(err)
			}
			field := parsed.LookUpField(tc.field)
			if field == nil {
				t.Fatal("width field missing")
			}
			if tc.binary {
				if string(field.DataType) != fmt.Sprintf("varbinary(%d)", tc.size) {
					t.Fatal("binary literal differs from shared byte capacity")
				}
			} else if field.Size != tc.size {
				t.Fatal("text literal differs from shared rune capacity")
			}
		})
	}
	for _, value := range []any{CreateMonitorConfigReq{}, UpdateMonitorConfigReq{}} {
		typ := reflect.TypeOf(value)
		for _, tc := range []struct {
			field string
			limit int
		}{{"Name", MonitorConfigNameMaxRunes}, {"InstanceIP", MonitorConfigKeyMaxRunes}} {
			field, _ := typ.FieldByName(tc.field)
			if !strings.Contains(field.Tag.Get("binding"), fmt.Sprintf("max=%d", tc.limit)) {
				t.Fatal("DTO literal differs from shared limit")
			}
		}
	}
}

func TestMonitorConfigDTOUnicodeWidthAndOmittedUpdate(t *testing.T) {
	for _, tc := range []struct {
		name            string
		keyLen, nameLen int
		update, valid   bool
	}{
		{"full unicode", 128, 255, false, true}, {"key over", 129, 255, false, false}, {"name over", 128, 256, false, false},
		{"update full", 128, 255, true, true}, {"update omitted key", 0, 1, true, true}, {"create missing key", 0, 1, false, false}, {"required name", 1, 0, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key, name := strings.Repeat("🌍", tc.keyLen), strings.Repeat("🌍", tc.nameLen)
			var value any = CreateMonitorConfigReq{Name: name, InstanceIP: key, PoolID: 7, ConfigType: 5, ConfigContent: "synthetic", Status: 1}
			if tc.update {
				value = UpdateMonitorConfigReq{ID: 1, Name: name, InstanceIP: key, PoolID: 7, ConfigType: 5, ConfigContent: "synthetic", Status: 1}
			}
			err := binding.Validator.ValidateStruct(value)
			if (err == nil) != tc.valid {
				t.Fatalf("unexpected DTO acceptance: %v", err == nil)
			}
		})
	}
}

func TestMonitorConfigCanonicalWebhookFitsWithoutTruncation(t *testing.T) {
	ip := "ffff:ffff:ffff:ffff:ffff:ffff:255.255.255.255"
	if net.ParseIP(ip) == nil || len(ip) != 45 {
		t.Fatal("invalid IPv6 boundary fixture")
	}
	for _, group := range []string{"1", "9223372036854775807"} {
		key := ip + "_" + group
		name := "webhook_" + strings.Repeat("🌍", 50) + "_" + group + "_" + ip
		if utf8.RuneCountInString(key) > MonitorConfigKeyMaxRunes || utf8.RuneCountInString(name) > MonitorConfigNameMaxRunes || len(key) > MonitorConfigKeyMaxBytes {
			t.Fatal("canonical webhook exceeds shared widths")
		}
		if len(key) <= 45 || utf8.RuneCountInString(name) <= 100 {
			t.Fatal("fixture does not demonstrate released width defect")
		}
		request := CreateMonitorConfigReq{Name: name, InstanceIP: key, PoolID: 7, ConfigType: 5, ConfigContent: "synthetic", Status: 1}
		if binding.Validator.ValidateStruct(request) != nil {
			t.Fatal("legal canonical webhook request rejected before persistence")
		}
	}
}
