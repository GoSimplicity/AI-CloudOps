package config

import (
	"reflect"
	"testing"

	"github.com/spf13/viper"
)

// webhookNestedEnvLoops 每种写法的重复加载次数：旧实现按 map 随机顺序写入，约一半概率丢值
const webhookNestedEnvLoops = 200

// 配置文件里 im_feishu 为 null（子键全部注释掉）或 {} 时，子键的环境变量覆盖必须稳定生效，
// 且 cfg 与全局 viper 取值一致。旧实现会把父键 webhook.im_feishu 的默认值 map 按随机顺序写在子键之后，
// 覆盖掉 WEBHOOK_IM_FEISHU_* 的环境变量值
func TestLoadWebhookNestedEnvStableWhenParentNullOrEmpty(t *testing.T) {
	t.Setenv(WebhookMySQLAddrEnv, fakeWebhookDSN)
	t.Setenv("WEBHOOK_PORT", "")
	t.Setenv("WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_SECRET", "test-secret-value")
	t.Setenv("WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_ID", "test-app-id")
	t.Setenv("WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS", "7")

	cases := []struct{ name, body string }{
		{"null", "webhook:\n  port: \"8889\"\n  im_feishu:\n  #  private_robot_app_secret: xxx\n"},
		{"empty-map", "webhook:\n  port: \"8889\"\n  im_feishu: {}\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeWebhookYAML(t, tc.body)
			t.Cleanup(viper.Reset)
			bad := 0
			for i := 0; i < webhookNestedEnvLoops; i++ {
				viper.Reset()
				cfg, _, err := loadWebhookFromFile(path)
				if err != nil {
					t.Fatalf("第 %d 次加载失败: %v", i, err)
				}
				// 不输出密钥值，只报告哪个字段不对
				ok := cfg.ImFeishu.PrivateRobotAppSecret == "test-secret-value" &&
					cfg.ImFeishu.PrivateRobotAppID == "test-app-id" &&
					cfg.ImFeishu.RequestTimeoutSeconds == 7 &&
					viper.GetString("webhook.im_feishu.private_robot_app_secret") == "test-secret-value" &&
					viper.GetString("webhook.im_feishu.private_robot_app_id") == "test-app-id" &&
					viper.GetInt("webhook.im_feishu.request_timeout_seconds") == 7
				if !ok {
					bad++
					continue
				}
				// 未设环境变量的子键取默认值，cfg 与全局逐字段一致
				if cfg.ImFeishu.GroupMessageAPI != "https://open.feishu.cn/open-apis/bot/v2/hook" {
					t.Fatalf("第 %d 次：未覆盖的子键应取默认值", i)
				}
				if n := compareWebhookCfgWithGlobal(t, reflect.ValueOf(*cfg), "webhook"); n != 12 {
					t.Fatalf("应比对 12 个字段, got %d", n)
				}
			}
			if bad > 0 {
				t.Fatalf("%d/%d 次加载中 im_feishu 子键的环境变量丢失或 cfg 与全局不一致", bad, webhookNestedEnvLoops)
			}
		})
	}
}

// 合并值只含叶子键、按键名排序：父键 webhook.im_feishu 不出现，写入顺序确定
func TestCollectWebhookLeafValuesSkipsParentKeys(t *testing.T) {
	t.Setenv("WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_SECRET", "test-secret-value")
	v := viper.New()
	v.SetConfigFile(writeWebhookYAML(t, "webhook:\n  im_feishu:\n"))
	setWebhookDefaults(v)
	if err := v.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	bindStructEnvVars(v, reflect.TypeOf(WebhookConfig{}), "webhook")

	m, err := collectWebhookLeafValues(v, webhookFileScan{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.vals["webhook.im_feishu"]; ok {
		t.Fatal("父键 webhook.im_feishu 不应进入合并值")
	}
	for i := 1; i < len(m.keys); i++ {
		if m.keys[i-1] >= m.keys[i] {
			t.Fatalf("键应严格升序: %s, %s", m.keys[i-1], m.keys[i])
		}
	}
	if len(m.keys) != 12 || len(m.vals) != 12 {
		t.Fatalf("应有 12 个叶子键, got keys=%d vals=%d", len(m.keys), len(m.vals))
	}
	for _, key := range m.keys {
		if _, composite := webhookCompositeKind(m.vals[key]); composite {
			t.Fatalf("%s 的值不应是 map 或列表", key)
		}
	}
}

// apply 必须按 keys 的固定顺序写入（cfg 解码与全局发布两次遍历顺序相同），不能按 map 随机顺序
func TestWebhookValuesApplyUsesSortedKeyOrder(t *testing.T) {
	m := webhookValues{vals: map[string]any{}}
	for _, k := range []string{"webhook.a", "webhook.b", "webhook.c", "webhook.d", "webhook.e", "webhook.f", "webhook.g", "webhook.h"} {
		m.keys = append(m.keys, k)
		m.vals[k] = k
	}
	for i := 0; i < webhookNestedEnvLoops; i++ {
		var got []string
		m.apply(func(key string, _ any) { got = append(got, key) })
		if !reflect.DeepEqual(got, m.keys) {
			t.Fatalf("第 %d 次写入顺序与 keys 不一致: %v", i, got)
		}
	}
}
