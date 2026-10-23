package config

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/viper"
)

// 第二轮 Review（e4c43a2 FAIL）补的用例：父键遮蔽、非 YAML 格式、别名键、带点号键、带引号 <<、显式 tag、
// 大小写重复键、解析错误脱敏、并发发布、<< 合并优先级、超时下限、报错来源、整数环境变量去空白。
// 子测试名只用标签，不带取值；写进配置的假值都带 leakMarkers 片段，报错/告警中出现即失败。

const strictHead = "webhook:\n  port: \"8889\"\n"

// writeWebhookFileNamed 在临时目录写指定文件名的配置文件
func writeWebhookFileNamed(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// strictLoad 在干净的全局 viper 和环境变量上加载 body（写成 name），返回结果和告警
func strictLoad(t *testing.T, name, body string, env map[string]string) (*WebhookConfig, error, []string) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
	clearAllWebhookEnv(t)
	for k, v := range env {
		t.Setenv(k, v)
	}
	warns := captureConfigWarn(t)
	cfg, _, err := loadWebhookFromFile(writeWebhookFileNamed(t, name, body))
	return cfg, err, *warns
}

// expectStartupFailure 断言启动失败（不是可继续的 NonFatal）、报错含全部片段、不回显值、不发布任何全局键
func expectStartupFailure(t *testing.T, err error, mustContain ...string) {
	t.Helper()
	if err == nil || IsNonFatal(err) {
		t.Fatalf("应启动失败, got err=%v", err != nil)
	}
	msg := err.Error()
	assertNoLeak(t, msg)
	for _, s := range mustContain {
		if !strings.Contains(msg, s) {
			t.Fatalf("报错应包含 %q: %s", s, msg)
		}
	}
	if len(viper.AllKeys()) != 0 {
		t.Fatal("失败时不应发布任何键到全局 viper")
	}
}

func expectLoaded(t *testing.T, cfg *WebhookConfig, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("应加载成功: %v", err)
	}
}

// P2-A（S1-S4）：非空的 WEBHOOK / WEBHOOK_IM_FEISHU 会遮蔽整棵子树（viper IsSet=false，叶子静默变 0/""），一律启动失败，只列变量名
func TestLoadWebhookParentEnvVarRejected(t *testing.T) {
	repoLike := strictHead + "  im_feishu:\n    group_message_api: \"https://g\"\n    request_timeout_seconds: 10\n"
	cases := []struct {
		name  string
		body  string
		env   map[string]string
		names []string
	}{
		{"S1-im-feishu", repoLike, map[string]string{"WEBHOOK_IM_FEISHU": "fake-leak-1"}, []string{"WEBHOOK_IM_FEISHU"}},
		{"S2-im-feishu-no-file-value", strictHead, map[string]string{"WEBHOOK_IM_FEISHU": "fake-leak-x"}, []string{"WEBHOOK_IM_FEISHU"}},
		{"S3-webhook-with-port", repoLike, map[string]string{"WEBHOOK": "fake-leak-1", "WEBHOOK_PORT": "8889"}, []string{"WEBHOOK"}},
		{"S4-webhook-all-ints-env", repoLike, map[string]string{"WEBHOOK": "fake-leak-1", "WEBHOOK_PORT": "8889", "WEBHOOK_FIXED_WORKERS": "1",
			"WEBHOOK_DEFAULT_UPGRADE_MINUTES": "1", "WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS": "1"}, []string{"WEBHOOK"}},
		{"both", repoLike, map[string]string{"WEBHOOK": "fake-leak-1", "WEBHOOK_IM_FEISHU": "fake-leak-2"}, []string{"WEBHOOK", "WEBHOOK_IM_FEISHU"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err, _ := strictLoad(t, "webhook.yaml", c.body, c.env)
			expectStartupFailure(t, err, c.names...)
		})
	}
	// 对照：空串视为未设置（S9）；叶子变量 WEBHOOK_PORT / WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS 不误伤
	t.Run("S9-empty-is-unset", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.yaml", repoLike, map[string]string{"WEBHOOK_IM_FEISHU": "", "WEBHOOK": ""})
		expectLoaded(t, cfg, err)
		if cfg.ImFeishu.RequestTimeoutSeconds != 10 || viper.GetInt("webhook.im_feishu.request_timeout_seconds") != 10 {
			t.Fatalf("空的父键变量不应影响取值: %d", cfg.ImFeishu.RequestTimeoutSeconds)
		}
	})
	t.Run("leaf-env-not-misfired", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.yaml", repoLike, map[string]string{"WEBHOOK_PORT": "9001", "WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS": "20"})
		expectLoaded(t, cfg, err)
		if cfg.Port != "9001" || cfg.ImFeishu.RequestTimeoutSeconds != 20 {
			t.Fatalf("叶子环境变量应生效: port=%s timeout=%d", cfg.Port, cfg.ImFeishu.RequestTimeoutSeconds)
		}
	})
}

// P2-A 裁定 1：IsSet=false 一律报错，不再跳过（直接构造被遮蔽的 viper 实例）
func TestCollectWebhookLeafValuesShadowedLeafIsError(t *testing.T) {
	for _, parent := range []string{"webhook.im_feishu", "webhook"} {
		t.Run(parent, func(t *testing.T) {
			v := viper.New()
			setWebhookDefaults(v)
			v.Set(parent, 5) // 父键被标量遮蔽：其下叶子 IsSet=false
			_, err := collectWebhookLeafValues(v, webhookFileScan{})
			if err == nil {
				t.Fatal("父键被遮蔽时应报错，而不是让叶子静默变成零值")
			}
			if !strings.Contains(err.Error(), "webhook.im_feishu.request_timeout_seconds") {
				t.Fatalf("应列出取不到值的叶子键: %v", err)
			}
		})
	}
}

// P2-A 裁定 1：schema 里每个叶子都有内置默认值（否则 IsSet=false 会在正常配置下误报）
func TestWebhookEveryLeafHasDefault(t *testing.T) {
	v := viper.New()
	setWebhookDefaults(v)
	for _, key := range webhookKeySchema.leafKeys {
		if !v.IsSet(key) || v.Get(key) == nil {
			t.Errorf("叶子 %s 没有内置默认值", key)
		}
	}
	if len(webhookKeySchema.leafKeys) != len(webhookLeafSpecs) {
		t.Fatalf("叶子数 %d 与测试清单 %d 不一致", len(webhookKeySchema.leafKeys), len(webhookLeafSpecs))
	}
}

// P2-A 裁定 3：配置文件只接受 .yaml/.yml（.json 按 YAML 子集走同一套结构扫描），其他格式一律启动失败
func TestLoadWebhookRejectsNonYAMLFormats(t *testing.T) {
	for _, c := range []struct{ name, file, body string }{
		{"S5-toml-parent-scalar", "webhook.toml", "[webhook]\nport = \"8889\"\nim_feishu = 5\n"},
		{"S6-toml-parent-empty", "webhook.toml", "[webhook]\nport = \"8889\"\nim_feishu = \"\"\n"},
		{"toml-valid", "webhook.toml", "[webhook]\nport = \"8889\"\nfixed_workers = 5\n"},
		{"S7-properties", "webhook.properties", "webhook.port=8889\nwebhook.im_feishu=5\n"},
		{"S11-ini", "webhook.ini", "[webhook]\nport=8889\nim_feishu=5\n"},
		{"hcl", "webhook.hcl", "webhook {\n port = \"8889\"\n}\n"},
		{"dotenv", "webhook.env", "webhook.port=8889\n"},
		{"upper-YAML-ext", "webhook.YAML", strictHead},
		{"no-ext", "webhook", strictHead},
		{"txt", "webhook.txt", strictHead},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err, _ := strictLoad(t, c.file, c.body, nil)
			expectStartupFailure(t, err, "只支持")
		})
	}
	t.Run("missing-toml-still-rejected", func(t *testing.T) {
		viper.Reset()
		clearAllWebhookEnv(t)
		_, _, err := loadWebhookFromFile(filepath.Join(t.TempDir(), "missing.toml"))
		expectStartupFailure(t, err, "只支持")
	})
	for _, ext := range []string{"yaml", "yml", "json"} {
		t.Run("accept-"+ext, func(t *testing.T) {
			body := strictHead + "  fixed_workers: 5\n"
			if ext == "json" {
				body = `{"webhook":{"port":"8889","fixed_workers":5}}`
			}
			cfg, err, _ := strictLoad(t, "webhook."+ext, body, nil)
			expectLoaded(t, cfg, err)
			if cfg.FixedWorkers != 5 {
				t.Fatalf("fixed_workers 应为 5, got %d", cfg.FixedWorkers)
			}
		})
	}
	// JSON 与 YAML 走同一套结构扫描
	for _, c := range []struct{ name, body, key string }{
		{"S10-json-parent-string", `{"webhook":{"port":"8889","im_feishu":"x"}}`, "webhook.im_feishu"},
		{"J1-json-float", `{"webhook":{"port":"8889","fixed_workers":1e1}}`, "webhook.fixed_workers"},
		{"J4-json-root-dotted", `{"webhook":{"port":"8889"},"webhook.fixed_workers":3}`, "webhook.fixed_workers"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err, _ := strictLoad(t, "webhook.json", c.body, nil)
			expectStartupFailure(t, err, c.key)
		})
	}
}

// P2-A 裁定 4：写成 YAML 别名的键一律拒绝（S8/B1-B5，含指向根键 webhook 的别名、webhook 子树外的别名键）；
// 作为值的别名、<< 合并仍展开后检查
func TestLoadWebhookRejectsAliasKeys(t *testing.T) {
	for _, c := range []struct{ name, body string }{
		{"B1-leaf", "x: &fw fixed_workers\n" + strictHead + "  *fw : 7\n"},
		{"B2-S8-parent-scalar", "x: &im im_feishu\n" + strictHead + "  *im : 5\n"},
		{"B3-parent-map", "x: &im im_feishu\n" + strictHead + "  *im : {request_timeout_seconds: 5}\n"},
		{"B4-root-webhook", "x: &w webhook\n*w :\n  port: \"8889\"\n  im_feishu: 5\n"},
		{"B5-leaf-under-im-feishu", "x: &t request_timeout_seconds\n" + strictHead + "  im_feishu:\n    *t : 5\n"},
		{"outside-webhook", "x: &k fake-leak-k\nother:\n  *k : 1\n" + strictHead},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err, _ := strictLoad(t, "webhook.yaml", c.body, nil)
			expectStartupFailure(t, err, "别名")
		})
	}
	t.Run("alias-value-ok", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.yaml", "x: &n 7\n"+strictHead+"  fixed_workers: *n\n", nil)
		expectLoaded(t, cfg, err)
		if cfg.FixedWorkers != 7 {
			t.Fatalf("值别名应展开, got %d", cfg.FixedWorkers)
		}
	})
	t.Run("alias-value-bad-still-checked", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", "x: &n 0x10\n"+strictHead+"  fixed_workers: *n\n", nil)
		expectStartupFailure(t, err, "webhook.fixed_workers")
	})
	// 别名键带进来的值来自配置文件，报错不能标成“内置默认值”（Review 探针 B5）
	for _, c := range []struct{ name, body string }{
		{"alias-key-map-value-labelled-file", "x: &t request_timeout_seconds\n" + strictHead + "  im_feishu:\n    *t : {a: 1}\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err, _ := strictLoad(t, "webhook.yaml", c.body, nil)
			expectStartupFailure(t, err, "别名")
			if strings.Contains(err.Error(), "内置默认值") {
				t.Fatalf("取自配置文件的值不应标成内置默认值: %v", err)
			}
		})
	}
}

// 第 5 点：根节点下带点号的键一律拒绝；webhook、im_feishu 下带点号的键也拒绝
func TestLoadWebhookRejectsDottedKeys(t *testing.T) {
	for _, c := range []struct{ name, body, want string }{
		{"A1-root-dotted-leaf", strictHead + "\"webhook.im_feishu.request_timeout_seconds\": 5\n", "点号"},
		{"A2-root-dotted-int", strictHead + "\"webhook.fixed_workers\": 7\n", "点号"},
		{"A8-root-dotted-with-nested", strictHead + "  fixed_workers: 5\n\"webhook.fixed_workers\": 0x10\n", "点号"},
		{"A9-root-dotted-parent", strictHead + "\"webhook.im_feishu\": 5\n", "点号"},
		{"A11-root-dotted-unknown", strictHead + "\"webhook.min_workers\": 3\n", "点号"},
		{"A12-only-root-dotted", "\"webhook.port\": \"8889\"\n\"webhook.im_feishu.request_timeout_seconds\": 5\n", "点号"},
		{"root-dotted-unrelated", strictHead + "\"other.key\": 1\n", "点号"},
		{"A13-under-webhook", strictHead + "  \"im_feishu.request_timeout_seconds\": 5\n", "点号"},
		{"under-im-feishu", strictHead + "  im_feishu:\n    \"a.b\": 1\n", "点号"},
		{"merged-root-dotted", "b: &b {\"webhook.fixed_workers\": 7}\n<<: *b\n" + strictHead, "点号"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err, _ := strictLoad(t, "webhook.yaml", c.body, nil)
			expectStartupFailure(t, err, c.want)
		})
	}
	// 窝窝头追加：根节点写 "webhook.im_feishu": 5，同时有正常的 webhook 块
	t.Run("root-dotted-parent-with-normal-block", func(t *testing.T) {
		body := "webhook:\n  port: \"8889\"\n  im_feishu:\n    request_timeout_seconds: 10\n\"webhook.im_feishu\": 5\n"
		_, err, _ := strictLoad(t, "webhook.yaml", body, nil)
		expectStartupFailure(t, err, "webhook.im_feishu", "点号")
	})
}

// 第 6 点：带引号或带 tag 的 << 一律拒绝（只支持不带引号、不带 tag 的 <<）。
// yaml.v3 里只有带引号的 "<<"（无 tag）是普通键，不会合并；!!merge "<<"、!!merge << 都会真的合并
func TestLoadWebhookRejectsQuotedMergeKey(t *testing.T) {
	for _, c := range []struct{ name, body string }{
		{"C1-double-quoted", strictHead + "  \"<<\": {fixed_workers: 7}\n"},
		{"single-quoted", strictHead + "  '<<': {fixed_workers: 7}\n"},
		{"root", "\"<<\": {other: 1}\n" + strictHead},
		{"tagged-str", strictHead + "  !!str <<: {fixed_workers: 7}\n"},
		{"tagged-merge-quoted", strictHead + "  !!merge \"<<\": {fixed_workers: 7}\n"},
		{"tagged-merge-plain", strictHead + "  !!merge <<: {fixed_workers: 7}\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err, _ := strictLoad(t, "webhook.yaml", c.body, nil)
			expectStartupFailure(t, err, "带引号或带 tag 的 << 不被支持")
		})
	}
}

// 第 6 点：叶子只接受 !!str、!!int、!!null 或不写 tag；!foo、!!float、!!binary、!!bool、!!timestamp 等显式 tag 拒绝
func TestLoadWebhookLeafExplicitTags(t *testing.T) {
	for _, c := range []struct{ name, line, key string }{
		{"E3-float-on-int", "  fixed_workers: !!float 10\n", "webhook.fixed_workers"},
		{"E8-custom-on-int", "  fixed_workers: !foo 5\n", "webhook.fixed_workers"},
		{"E9-binary-on-secret", "  im_feishu:\n    private_robot_app_secret: !!binary ZmFrZS1sZWFr\n", "webhook.im_feishu.private_robot_app_secret"},
		{"bool-on-string", "  front_domain: !!bool true\n", "webhook.front_domain"},
		{"timestamp-on-string", "  front_domain: !!timestamp 2001-12-14\n", "webhook.front_domain"},
		{"custom-on-string", "  front_domain: !foo fake-leak-f\n", "webhook.front_domain"},
		{"custom-on-port", "webhook:\n  port: !foo 8889\n", "webhook.port"},
	} {
		t.Run(c.name, func(t *testing.T) {
			body := strictHead + c.line
			if strings.HasPrefix(c.line, "webhook:") {
				body = c.line
			}
			_, err, _ := strictLoad(t, "webhook.yaml", body, nil)
			expectStartupFailure(t, err, c.key, "tag")
		})
	}
	for _, c := range []struct {
		name, line string
		fw         int
		front      string
	}{
		{"E2-str-int", "  fixed_workers: !!str 10\n", 10, "http://localhost:3000"},
		{"int-int", "  fixed_workers: !!int 12\n", 12, "http://localhost:3000"},
		{"E4-int-quoted", "  fixed_workers: !!int \"11\"\n", 11, "http://localhost:3000"},
		{"null-tag", "  fixed_workers: !!null\n", 10, "http://localhost:3000"},
		{"str-on-string", "  front_domain: !!str 007\n", 10, "007"},
		{"untagged-bool-on-string", "  front_domain: true\n", 10, "true"},
		{"untagged-float-on-string", "  front_domain: 1e3\n", 10, "1e3"},
	} {
		t.Run(c.name, func(t *testing.T) {
			cfg, err, _ := strictLoad(t, "webhook.yaml", strictHead+c.line, nil)
			expectLoaded(t, cfg, err)
			if cfg.FixedWorkers != c.fw || cfg.FrontDomain != c.front {
				t.Fatalf("got fw=%d front=%q", cfg.FixedWorkers, cfg.FrontDomain)
			}
		})
	}
}

// 第 7 点：同一父节点下小写后相同的键（含完全相同）一律不合规，口径同 ee753b4 check-webhook-env.sh（整份文件、列表项不参与）；
// 只写一个大写键（B3：只有 PORT）仍接受
func TestLoadWebhookRejectsCaseDuplicateKeys(t *testing.T) {
	for _, c := range []struct{ name, body, path string }{
		{"leaf-int", strictHead + "  fixed_workers: 5\n  FIXED_WORKERS: 7\n", "webhook.fixed_workers"},
		{"leaf-string", strictHead + "  front_domain: a\n  FRONT_DOMAIN: b\n", "webhook.front_domain"},
		{"port-PORT", "webhook:\n  port: \"8889\"\n  PORT: \"9000\"\n", "webhook.port"},
		{"root-Webhook", strictHead + "  fixed_workers: 5\nWebhook:\n  port: \"9000\"\n", "webhook"},
		{"im-feishu-leaf", strictHead + "  im_feishu:\n    private_robot_app_id: a\n    PRIVATE_ROBOT_APP_ID: b\n", "webhook.im_feishu.private_robot_app_id"},
		{"im-feishu-parent", strictHead + "  im_feishu: {}\n  IM_FEISHU: {}\n", "webhook.im_feishu"},
		{"mysql-addr", strictHead + "mysql:\n  addr: \"\"\n  ADDR: \"\"\n", "mysql.addr"},
		{"unknown-key", strictHead + "  foo: 1\n  FOO: 2\n", "webhook.foo"},
		{"merged-case-variant", "b: &b {FIXED_WORKERS: 3}\nwebhook:\n  port: \"8889\"\n  <<: *b\n  fixed_workers: 7\n", "webhook.fixed_workers"},
		{"exact-duplicate-json", `{"webhook":{"port":"8889","front_domain":"a","front_domain":"b"}}`, "webhook.front_domain"},
	} {
		t.Run(c.name, func(t *testing.T) {
			name := "webhook.yaml"
			if strings.HasPrefix(c.body, "{") {
				name = "webhook.json"
			}
			_, err, _ := strictLoad(t, name, c.body, nil)
			expectStartupFailure(t, err, c.path, "重复")
		})
	}
	t.Run("B3-single-upper-port-accepted", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.yaml", "webhook:\n  PORT: \"9100\"\n", nil)
		expectLoaded(t, cfg, err)
		if cfg.Port != "9100" {
			t.Fatalf("单独的 PORT 应生效, got %s", cfg.Port)
		}
	})
	t.Run("single-Webhook-root-accepted", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.yaml", "Webhook:\n  port: \"9101\"\n", nil)
		expectLoaded(t, cfg, err)
		if cfg.Port != "9101" {
			t.Fatalf("单独的 Webhook 根应生效, got %s", cfg.Port)
		}
	})
	t.Run("list-items-not-compared", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.yaml", strictHead+"other:\n  - {a: 1, A: 2}\n  - a: 3\n", nil)
		expectLoaded(t, cfg, err)
	})
	t.Run("merge-same-spelling-overridden-ok", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.yaml", "b: &b {fixed_workers: 3}\nwebhook:\n  port: \"8889\"\n  <<: *b\n  fixed_workers: 7\n", nil)
		expectLoaded(t, cfg, err)
		if cfg.FixedWorkers != 7 {
			t.Fatalf("本层键应覆盖合并值, got %d", cfg.FixedWorkers)
		}
	})
}

// 第 8 点：viper/YAML 解析错误会回显文件内容，只保留行号和原因
func TestLoadWebhookParseErrorSanitized(t *testing.T) {
	for _, c := range []struct{ name, file, body, line string }{
		{"M3-unterminated", "webhook.yaml", strictHead + "  im_feishu:\n    private_robot_app_secret: \"fake-leak-secret\n  x: [\n", "第 4 行"},
		{"M4-dup-key", "webhook.yaml", strictHead + "  im_feishu:\n    private_robot_app_secret: a\n    private_robot_app_secret: b\n  fake-leak-k: 1\n  fake-leak-k: 2\n", "第 7 行"},
		{"M8-tab", "webhook.yaml", strictHead + "  im_feishu:\n\tprivate_robot_app_secret: fake-leak-secret\n", "第 4 行"},
		{"M9-map-key", "webhook.yaml", strictHead + "  ? {fake-leak-secret: 1}\n  : 1\n", ""},
		{"P3-seq-key", "webhook.yaml", strictHead + "  ? [fake-leak-a, b]\n  : 1\n", ""},
		{"P5-root-list", "webhook.yaml", "- fake-leak-w\n", "第 1 行"},
		{"P6-root-scalar", "webhook.yaml", "fake-leak-hello\n", "第 1 行"},
		{"E5-null-tag-with-value", "webhook.yaml", strictHead + "  fixed_workers: !!null fake-leak-5\n", ""},
		{"M6-json", "webhook.json", `{"webhook":{"im_feishu":{"private_robot_app_secret":"fake-leak-secret" "x":1}}}`, ""},
		{"json-bad-literal", "webhook.json", `{"webhook":{"port":fake-leak-7391}}`, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err, _ := strictLoad(t, c.file, c.body, nil)
			expectStartupFailure(t, err, "读取Webhook配置文件失败", c.line)
		})
	}
}

// 第 9 点：并发加载时全局 viper 写入受锁保护（go test -race 下验证）
func TestLoadWebhookConcurrentPublishRaceFree(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	clearAllWebhookEnv(t)
	p := writeWebhookYAML(t, strictHead+"  fixed_workers: 5\n")
	var wg sync.WaitGroup
	errs := make(chan error, 64)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 8; j++ {
				if _, _, err := loadWebhookFromFile(p); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if viper.GetInt("webhook.fixed_workers") != 5 {
		t.Fatal("并发加载后全局值不对")
	}
}

// 第 10 点（Review 存活变异 X04/X05/X09）
func TestLoadWebhookReviewGapCases(t *testing.T) {
	t.Run("X04-merge-list", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", "a: &a {fixed_workers: 0x10}\nwebhook:\n  port: \"8889\"\n  <<: [*a]\n", nil)
		expectStartupFailure(t, err, "配置文件中的 webhook.fixed_workers")
	})
	t.Run("X05-upper-root", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", "Webhook:\n  port: \"8889\"\n  fixed_workers: 0x10\n", nil)
		expectStartupFailure(t, err, "配置文件中的 webhook.fixed_workers")
	})
	t.Run("X09-stable-order", func(t *testing.T) {
		body := strictHead + "  fixed_workers: 0x10\n  front_domain: {a: 1}\n  default_upgrade_minutes: 1.5\n"
		want := "Webhook 配置写法或类型错误: 配置文件中的 webhook.default_upgrade_minutes 应为十进制整数；" +
			"配置文件中的 webhook.fixed_workers 应为十进制整数；配置文件中的 webhook.front_domain 应为字符串"
		for i := 0; i < 20; i++ {
			_, err, _ := strictLoad(t, "webhook.yaml", body, nil)
			if err == nil || err.Error() != want {
				t.Fatalf("第 %d 次报错不符:\n got %v\nwant %s", i, err, want)
			}
		}
	})
}

// 第 14 点（N1）：<< 按 YAML 规则先算合并结果（本层键优先、列表里靠前的优先），只检查最终生效的条目
func TestLoadWebhookMergePrecedence(t *testing.T) {
	t.Run("overridden-bad-base-ok", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.yaml", "b: &b {fixed_workers: 0x10}\nwebhook: {port: \"8889\", <<: *b, fixed_workers: 7}\n", nil)
		expectLoaded(t, cfg, err)
		if cfg.FixedWorkers != 7 || viper.GetInt("webhook.fixed_workers") != 7 {
			t.Fatalf("本层 7 应覆盖合并来的值, got %d", cfg.FixedWorkers)
		}
	})
	t.Run("local-null-over-base-99", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.yaml", "b: &b {fixed_workers: 99}\nwebhook:\n  port: \"8889\"\n  <<: *b\n  fixed_workers: ~\n", nil)
		expectLoaded(t, cfg, err)
		// 与 viper 结果一致：本层 null 覆盖了 99，取默认值 10
		if cfg.FixedWorkers != 10 || viper.GetInt("webhook.fixed_workers") != 10 {
			t.Fatalf("本层 null 应回落默认值 10, got cfg=%d global=%d", cfg.FixedWorkers, viper.GetInt("webhook.fixed_workers"))
		}
	})
	t.Run("list-first-wins-good", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.yaml", "b: &b {fixed_workers: 0x10}\nc: &c {fixed_workers: 4}\nwebhook:\n  port: \"8889\"\n  <<: [*c, *b]\n", nil)
		expectLoaded(t, cfg, err)
		if cfg.FixedWorkers != 4 {
			t.Fatalf("列表里靠前的 4 应生效, got %d", cfg.FixedWorkers)
		}
	})
	t.Run("list-first-wins-bad", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", "b: &b {fixed_workers: 0x10}\nc: &c {fixed_workers: 4}\nwebhook:\n  port: \"8889\"\n  <<: [*b, *c]\n", nil)
		expectStartupFailure(t, err, "配置文件中的 webhook.fixed_workers")
	})
	t.Run("nested-base-merge", func(t *testing.T) {
		body := "a: &a {fixed_workers: 0x10, default_upgrade_minutes: 5}\nb: &b {<<: *a, fixed_workers: 6}\nwebhook:\n  port: \"8889\"\n  <<: *b\n"
		cfg, err, _ := strictLoad(t, "webhook.yaml", body, nil)
		expectLoaded(t, cfg, err)
		if cfg.FixedWorkers != 6 || cfg.DefaultUpgradeMinutes != 5 {
			t.Fatalf("got fw=%d dum=%d", cfg.FixedWorkers, cfg.DefaultUpgradeMinutes)
		}
	})
	// 第三轮 P3-4：合并来源自身带 <<（多级合并链）时，深层来源的条目同样要检查
	const h = "webhook:\n  port: \"8889\"\n"
	t.Run("N1-chain-bad-int", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", "i: &i {fixed_workers: 0x10}\no: &o {<<: *i}\n"+h+"  <<: *o\n", nil)
		expectStartupFailure(t, err, "配置文件中的 webhook.fixed_workers")
	})
	t.Run("N3-chain-case-dup", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", "i: &i {Fixed_Workers: 3}\no: &o {<<: *i}\n"+h+"  <<: *o\n  fixed_workers: 2\n", nil)
		expectStartupFailure(t, err, "webhook.fixed_workers 有仅大小写不同或完全相同的同层重复键")
	})
	t.Run("overridden-parent-scalar-ok", func(t *testing.T) {
		body := "b: &b {im_feishu: 5}\nwebhook:\n  port: \"8889\"\n  <<: *b\n  im_feishu:\n    request_timeout_seconds: 9\n"
		cfg, err, _ := strictLoad(t, "webhook.yaml", body, nil)
		expectLoaded(t, cfg, err)
		if cfg.ImFeishu.RequestTimeoutSeconds != 9 {
			t.Fatalf("got %d", cfg.ImFeishu.RequestTimeoutSeconds)
		}
	})
}

// 第 15 点（N2）：im_feishu.request_timeout_seconds 范围 1-600，0 不再表示不设超时
func TestLoadWebhookFeishuTimeoutZeroRejected(t *testing.T) {
	t.Run("file-0", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", strictHead+"  im_feishu:\n    request_timeout_seconds: 0\n", nil)
		expectStartupFailure(t, err, "webhook.im_feishu.request_timeout_seconds", "1 到 600")
	})
	t.Run("env-0", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", strictHead, map[string]string{"WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS": "0"})
		expectStartupFailure(t, err, "WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS", "1 到 600")
	})
	for _, n := range []int{1, 600} {
		t.Run(fmt.Sprintf("accept-%d", n), func(t *testing.T) {
			cfg, err, _ := strictLoad(t, "webhook.yaml", fmt.Sprintf("%s  im_feishu:\n    request_timeout_seconds: %d\n", strictHead, n), nil)
			expectLoaded(t, cfg, err)
		})
	}
}

// 第 16 点（N3）：扫描阶段（文件本身）的错误写“配置文件中的 KEY”，只有取自环境变量的错误才写“环境变量”
func TestLoadWebhookErrorNamesActualSource(t *testing.T) {
	t.Run("file-bad-env-good", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", strictHead+"  fixed_workers: 0x10\n", map[string]string{"WEBHOOK_FIXED_WORKERS": "5"})
		expectStartupFailure(t, err, "配置文件中的 webhook.fixed_workers")
		if strings.Contains(err.Error(), "环境变量") {
			t.Fatalf("错误来自配置文件，不应提环境变量: %v", err)
		}
	})
	t.Run("json-duplicate", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.json", `{"webhook":{"port":"8889","fixed_workers":"x","fixed_workers":5}}`, nil)
		expectStartupFailure(t, err, "配置文件中的 webhook.fixed_workers")
		if strings.Contains(err.Error(), "环境变量") {
			t.Fatalf("错误来自配置文件，不应提环境变量: %v", err)
		}
	})
	t.Run("file-out-of-range", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", strictHead+"  fixed_workers: 2000\n", nil)
		expectStartupFailure(t, err, "配置文件中的 webhook.fixed_workers", "1 到 1024")
		if strings.Contains(err.Error(), "环境变量") {
			t.Fatalf("越界值来自配置文件，不应提环境变量: %v", err)
		}
	})
	t.Run("file-bad-port", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", "webhook:\n  port: \"0\"\n", nil)
		expectStartupFailure(t, err, "配置文件中的 webhook.port")
		if strings.Contains(err.Error(), "环境变量") {
			t.Fatalf("端口来自配置文件，不应提环境变量: %v", err)
		}
	})
	t.Run("env-bad", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", strictHead, map[string]string{"WEBHOOK_FIXED_WORKERS": "0x10"})
		expectStartupFailure(t, err, "webhook.fixed_workers（环境变量 WEBHOOK_FIXED_WORKERS）")
		if strings.Contains(err.Error(), "配置文件中的") {
			t.Fatalf("错误来自环境变量: %v", err)
		}
	})
	t.Run("env-out-of-range", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", strictHead, map[string]string{"WEBHOOK_FIXED_WORKERS": "2000"})
		expectStartupFailure(t, err, "webhook.fixed_workers（环境变量 WEBHOOK_FIXED_WORKERS）必须在 1 到 1024 之间")
	})
}

// 第 17 点（N5）：整数类环境变量先去首尾 ASCII 空白（空格、\t、\r、\n）再严格解析；字符串类环境变量保持原样
func TestLoadWebhookIntEnvTrimsASCIISpace(t *testing.T) {
	for _, c := range []struct{ name, val string }{
		{"lead-space", " 10"}, {"trail-space", "10 "}, {"lf", "10\n"}, {"crlf", "10\r\n"}, {"tab", "\t10\t"},
	} {
		t.Run(c.name, func(t *testing.T) {
			cfg, err, _ := strictLoad(t, "webhook.yaml", strictHead, map[string]string{"WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS": c.val})
			expectLoaded(t, cfg, err)
			if cfg.ImFeishu.RequestTimeoutSeconds != 10 || viper.GetInt("webhook.im_feishu.request_timeout_seconds") != 10 {
				t.Fatalf("got %d", cfg.ImFeishu.RequestTimeoutSeconds)
			}
		})
	}
	for _, c := range []struct{ name, val string }{
		{"inner-space", "1 0"}, {"nbsp", "\u00a010"}, {"fullwidth", "１０"}, {"only-space", "  "}, {"vtab", "\v10"},
	} {
		t.Run("reject-"+c.name, func(t *testing.T) {
			_, err, _ := strictLoad(t, "webhook.yaml", strictHead, map[string]string{"WEBHOOK_FIXED_WORKERS": c.val})
			expectStartupFailure(t, err, "WEBHOOK_FIXED_WORKERS")
		})
	}
	t.Run("quoted-yaml-space-still-rejected", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", strictHead+"  fixed_workers: \" 10\"\n", nil)
		expectStartupFailure(t, err, "配置文件中的 webhook.fixed_workers")
	})
	// 第三轮 P3-7：URL 类字段改为去首尾空白（见 TestLoadWebhookURLFieldsTrimmed），其余字符串环境变量仍原样保留
	t.Run("string-env-kept-as-is", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.yaml", strictHead, map[string]string{"WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_ID": " cli_a \n"})
		expectLoaded(t, cfg, err)
		if cfg.ImFeishu.PrivateRobotAppID != " cli_a \n" {
			t.Fatalf("非 URL 字符串环境变量应原样保留, got %q", cfg.ImFeishu.PrivateRobotAppID)
		}
	})
}

// 第 12 点：K8s 中名为 webhook 的 Service 会注入 WEBHOOK_PORT=tcp://…，按端口校验启动失败（文档记为升级风险）
func TestLoadWebhookK8sServiceLinkPortFails(t *testing.T) {
	_, err, _ := strictLoad(t, "webhook.yaml", strictHead, map[string]string{"WEBHOOK_PORT": "tcp://10.0.0.1:8889"})
	expectStartupFailure(t, err, "WEBHOOK_PORT")
}

// 第 11 点：仓库 config/webhook.yaml 删掉 7 个未知键后不再有未知键告警，也能直接加载
func TestRepoWebhookConfigHasNoUnknownKeys(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	clearAllWebhookEnv(t)
	warns := captureConfigWarn(t)
	if _, _, err := loadWebhookFromFile(webhookConfigFile); err != nil {
		t.Fatalf("仓库 webhook.yaml 应能加载: %v", err)
	}
	if len(*warns) != 0 {
		t.Fatalf("仓库 webhook.yaml 不应再有告警: %d 条", len(*warns))
	}
}

// 第 18 点：仓库 webhook.yaml 与内置默认值的飞书群消息地址为 bot/v2/hook（不带尾斜杠）
func TestWebhookFeishuGroupMessageAPIDefault(t *testing.T) {
	const want = "https://open.feishu.cn/open-apis/bot/v2/hook"
	v := viper.New()
	setWebhookDefaults(v)
	if got := v.GetString("webhook.im_feishu.group_message_api"); got != want {
		t.Fatalf("默认值应为 %s, got %s", want, got)
	}
	f, ok := reflectFieldTag(ImFeishuConfig{}, "GroupMessageAPI", "default")
	if !ok || f != want {
		t.Fatalf("ImFeishuConfig.GroupMessageAPI default 标签应为 %s, got %s", want, f)
	}
	viper.Reset()
	t.Cleanup(viper.Reset)
	clearAllWebhookEnv(t)
	cfg, _, err := loadWebhookFromFile(webhookConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ImFeishu.GroupMessageAPI != want {
		t.Fatalf("仓库 webhook.yaml 应为 %s, got %s", want, cfg.ImFeishu.GroupMessageAPI)
	}
}

func reflectFieldTag(v any, field, tag string) (string, bool) {
	f, ok := reflect.TypeOf(v).FieldByName(field)
	if !ok {
		return "", false
	}
	return f.Tag.Get(tag), true
}

// Review 存活变异 X08：字符串叶子取到 nil（环境变量/默认值路径不会出现，属于防御分支）时按空串处理，不当作类型错误
func TestNormalizeWebhookStringNil(t *testing.T) {
	v, ok := normalizeWebhookString(nil, "", false)
	if !ok || v != "" {
		t.Fatalf("nil 应归一为空串, got %v %v", v, ok)
	}
}

// Review 存活变异 X17：超出 int64 的无符号整数应夹到 MaxInt，再由范围校验按“超过上限”拒绝；
// 不能归一成 0（加载层面两者都会被拒绝，只能在单元层面区分）
func TestNormalizeWebhookIntHugeUint(t *testing.T) {
	for _, in := range []any{uint64(math.MaxUint64), uint64(math.MaxInt64) + 1, uint(math.MaxUint)} {
		v, ok := normalizeWebhookInt(in, "", false)
		if !ok || v != math.MaxInt {
			t.Fatalf("%T 超大值应夹到 MaxInt, got %v %v", in, v, ok)
		}
	}
	if v, ok := normalizeWebhookInt(uint64(7), "", false); !ok || v != 7 {
		t.Fatalf("普通无符号整数应原样返回, got %v %v", v, ok)
	}
}

// 变异存活补测（第二轮）：结构类报错的确切说明、只在扫描层能区分的检查、来源标签
func TestWebhookScanSurvivorGaps(t *testing.T) {
	t.Run("parent-scalar-exact-message", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", strictHead+"  im_feishu: 5\n", nil)
		expectStartupFailure(t, err, "配置文件中的 webhook.im_feishu 应为 map 或留空")
		if strings.Contains(err.Error(), "取不到值") {
			t.Fatalf("父键写成标量应只报类型错误: %v", err)
		}
	})
	t.Run("root-webhook-scalar-exact-message", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", "webhook: 5\n", nil)
		expectStartupFailure(t, err, "配置文件中的 webhook 应为 map 或留空")
		if strings.Contains(err.Error(), "取不到值") {
			t.Fatalf("webhook 写成标量应只报类型错误: %v", err)
		}
	})
	t.Run("dotted-unknown-key-name-not-echoed", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", strictHead+"  fake-leak.k: 1\nfake-leak.r: 2\n", nil)
		expectStartupFailure(t, err, "第 3 行的键名带点号", "第 4 行的键名带点号")
		if strings.Contains(err.Error(), "fake-leak") {
			t.Fatalf("非 schema 的带点号键名不应回显: %v", err)
		}
	})
	t.Run("invalid-map-key-truncated", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", strictHead+"  ? {fake-leak-secret: 1}\n  : 1\n", nil)
		expectStartupFailure(t, err)
		if !strings.HasSuffix(err.Error(), "配置文件解析失败: 键不是合法的 map 键") {
			t.Fatalf("invalid map key 只输出白名单原因: %v", err)
		}
	})
	// viper 读取时已拒绝同层多个 <<（mapping key 重复），结构扫描自身的检查只能直接调用扫描函数验证
	t.Run("scan-multiple-merge-keys", func(t *testing.T) {
		scan, err := scanWebhookConfigData([]byte("a: &a {fixed_workers: 3}\nb: &b {fixed_workers: 4}\n"+strictHead+"  <<: *a\n  <<: *b\n"), false)
		if err != nil {
			t.Fatal(err)
		}
		if e := scan.issues.err(); e == nil || !strings.Contains(e.Error(), "配置文件第 6 行：同一层只能写一个 <<") {
			t.Fatalf("应报同层多个 <<, got %v", e)
		}
	})
	t.Run("scan-parse-error-sanitized", func(t *testing.T) {
		_, err := scanWebhookConfigData([]byte(strictHead+"  im_feishu:\n    private_robot_app_secret: \"fake-leak-secret\n  x: [\n"), false)
		if err == nil || strings.Contains(err.Error(), "fake-leak") || !strings.Contains(err.Error(), "配置文件第 4 行") {
			t.Fatalf("扫描层解析错误应脱敏并保留行号, got %v", err)
		}
	})
	t.Run("source-labels", func(t *testing.T) {
		for _, c := range []struct{ src, want string }{
			{webhookSrcFile, "配置文件中的 webhook.fixed_workers"},
			{webhookSrcEnv, "webhook.fixed_workers（环境变量 WEBHOOK_FIXED_WORKERS）"},
			{webhookSrcDefault, "webhook.fixed_workers（内置默认值）"},
			{"", "webhook.fixed_workers"},
		} {
			if got := webhookSourceLabel("webhook.fixed_workers", c.src); got != c.want {
				t.Fatalf("src=%q: got %q want %q", c.src, got, c.want)
			}
		}
	})
}
