package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// 第三轮 P3-1(a)：键上写了 !!str 以外的显式 tag（!!binary/!!int/!!bool/!!float/!!null 等）一律拒绝；
// 否则 !!binary 键会被解码成真实键名写进 viper，而结构扫描只看到原文，从而绕过原文校验
func TestLoadWebhookRejectsTaggedKeys(t *testing.T) {
	for _, c := range []struct{ name, body string }{
		{"B1-binary-leaf", strictHead + "  !!binary Zml4ZWRfd29ya2Vycw==: 0x10\n"},
		{"B2-binary-nested-leaf", strictHead + "  im_feishu:\n    !!binary cmVxdWVzdF90aW1lb3V0X3NlY29uZHM=: 0x10\n"},
		{"B3-binary-parent", strictHead + "  !!binary aW1fZmVpc2h1:\n    request_timeout_seconds: 0x10\n"},
		{"B4-binary-root", "!!binary d2ViaG9vaw==:\n  port: \"8889\"\n  fixed_workers: 0x10\n"},
		{"B5-binary-flow", "webhook: {port: \"8889\", !!binary Zml4ZWRfd29ya2Vycw==: 0x10}\n"},
		{"int-tag-key", strictHead + "  !!int 1: 2\n"},
		{"bool-tag-key", strictHead + "  !!bool true: 2\n"},
		{"custom-tag-key", strictHead + "  !foo fixed_workers: 3\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err, _ := strictLoad(t, "webhook.yaml", c.body, nil)
			expectStartupFailure(t, err, "键使用了不允许的 YAML tag")
		})
	}
	t.Run("str-tag-key-allowed", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.yaml", strictHead+"  !!str fixed_workers: 7\n", nil)
		expectLoaded(t, cfg, err)
		if cfg.FixedWorkers != 7 {
			t.Fatalf("!!str 键应正常生效, got %d", cfg.FixedWorkers)
		}
	})
}

// 第三轮 P3-1(b)：默认值分支里 viper 认为值来自配置文件、扫描却没拿到原文时，不能按内置默认值放行
func TestCollectWebhookLeafValuesInConfigWithoutRaw(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader("webhook:\n  fixed_workers: 0x10\n")); err != nil {
		t.Fatal(err)
	}
	setWebhookDefaults(v)
	_, err := collectWebhookLeafValues(v, webhookFileScan{})
	if err == nil || !strings.Contains(err.Error(), "配置文件中的 webhook.fixed_workers 无法按原文校验") {
		t.Fatalf("viper 有文件值而扫描没有原文时应报错, got %v", err)
	}
}

// 第三轮 P3-6：Y01/Y02 —— IsSet=false 只报“取不到值”，不能再落到类型校验
func TestCollectWebhookLeafValuesUnsetExactMessage(t *testing.T) {
	v := viper.New()
	setWebhookDefaults(v)
	v.Set("webhook.im_feishu", "x")
	_, err := collectWebhookLeafValues(v, webhookFileScan{})
	if err == nil {
		t.Fatal("父键被遮蔽时应报错")
	}
	msg := err.Error()
	for _, leaf := range []string{"request_timeout_seconds", "group_message_api", "tenant_access_token_api", "private_robot_app_id", "private_robot_app_secret"} {
		if !strings.Contains(msg, "webhook.im_feishu."+leaf+" 取不到值") {
			t.Fatalf("应报 webhook.im_feishu.%s 取不到值: %v", leaf, err)
		}
	}
	if strings.Contains(msg, "应为") || strings.Contains(msg, "内置默认值") {
		t.Fatalf("取不到值的叶子不应再报类型错误: %v", err)
	}
}

// 第三轮 P3-6：Y03 —— 父键环境变量只要非空（含纯空白）就按父键变量拒绝，报错文字固定
func TestLoadWebhookParentEnvVarWhitespaceOnly(t *testing.T) {
	_, err, _ := strictLoad(t, "webhook.yaml", strictHead, map[string]string{"WEBHOOK": "  "})
	expectStartupFailure(t, err)
	const want = "环境变量 WEBHOOK 对应的是父键（整棵子树），非空时会遮蔽其下全部叶子键、使其静默变成零值，不支持；请删除或置空，改用叶子变量（如 WEBHOOK_PORT、WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS）"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("纯空白的 WEBHOOK 也应按父键变量拒绝: %v", err)
	}
}

// 第三轮 J-1：<< 合并链超过 64 层不再静默截断，而是报校验错误；64 层以内正常加载
func webhookMergeChain(hops int, val string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "a0: &a0 {fixed_workers: %s}\n", val)
	for i := 1; i < hops; i++ {
		fmt.Fprintf(&b, "a%d: &a%d {<<: *a%d}\n", i, i, i-1)
	}
	fmt.Fprintf(&b, "webhook:\n  port: \"8889\"\n  <<: *a%d\n", hops-1)
	return b.String()
}

func TestLoadWebhookMergeChainDepthLimit(t *testing.T) {
	for _, hops := range []int{63, 64} {
		t.Run(fmt.Sprintf("hops-%d-ok", hops), func(t *testing.T) {
			cfg, err, _ := strictLoad(t, "webhook.yaml", webhookMergeChain(hops, "7"), nil)
			expectLoaded(t, cfg, err)
			if cfg.FixedWorkers != 7 {
				t.Fatalf("got %d", cfg.FixedWorkers)
			}
		})
	}
	for _, c := range []struct {
		hops int
		val  string
	}{{65, "7"}, {66, "7"}, {66, "010"}} {
		t.Run(fmt.Sprintf("hops-%d-%s-rejected", c.hops, c.val), func(t *testing.T) {
			_, err, _ := strictLoad(t, "webhook.yaml", webhookMergeChain(c.hops, c.val), nil)
			expectStartupFailure(t, err, "<< 合并链超过 64 层")
		})
	}
	t.Run("nesting-over-64-rejected", func(t *testing.T) {
		deep := "deep: " + strings.Repeat("{a: ", 70) + "1" + strings.Repeat("}", 70) + "\n"
		_, err, _ := strictLoad(t, "webhook.yaml", deep+strictHead, nil)
		expectStartupFailure(t, err, "嵌套超过 64 层")
	})
}

// 第三轮 F1：多文档 YAML（---）一律拒绝（viper 只读第一个文档，其余文档会被静默忽略）
func TestLoadWebhookRejectsMultiDocument(t *testing.T) {
	for _, c := range []struct{ name, body string }{
		{"two-docs", strictHead + "---\nwebhook:\n  fixed_workers: 0x10\n"},
		{"second-doc-scalar", strictHead + "---\nfake-leak-x\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err, _ := strictLoad(t, "webhook.yaml", c.body, nil)
			expectStartupFailure(t, err, "配置文件包含多个 YAML 文档")
		})
	}
	// 第二个文档有语法错误：viper 只读第一个文档不报错，扫描读第二个文档时报错，同样只输出白名单原因
	t.Run("second-doc-syntax-error-sanitized", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", strictHead+"---\nx: \"zz-unknown-zz\n", nil)
		expectStartupFailure(t, err, "读取Webhook配置文件失败: 配置文件第 4 行: ")
		if strings.Contains(err.Error(), "zz-unknown-zz") || strings.Contains(err.Error(), "yaml:") {
			t.Fatalf("第二个文档的解析错误不应回显原文")
		}
	})
	t.Run("leading-marker-single-doc-ok", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.yaml", "---\n"+strictHead+"  fixed_workers: 3\n", nil)
		expectLoaded(t, cfg, err)
		if cfg.FixedWorkers != 3 {
			t.Fatalf("got %d", cfg.FixedWorkers)
		}
	})
}

// 第三轮 P3-5：解析错误只输出白名单原因 + 锚定位置的行号，不回显任何原文
func TestLoadWebhookParseErrorWhitelist(t *testing.T) {
	for _, c := range []struct{ name, file, body string }{
		{"J6-json-overflow", "webhook.json", `{"webhook":{"port":"8889","fixed_workers":1e999}}`},
		{"P7-backtick", "webhook.yaml", "a`fake-leak-root\n"},
		{"invalid-utf8", "webhook.yaml", strictHead + "  front_domain: \"a\xffb\"\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err, _ := strictLoad(t, c.file, c.body, nil)
			expectStartupFailure(t, err, "读取Webhook配置文件失败")
			for _, bad := range []string{"1e999", "fixed_workers", "fake-", "`", "Go struct", "unmarshal"} {
				if strings.Contains(err.Error(), bad) {
					t.Fatalf("解析错误不应回显原文或内部细节（含 %q）", bad)
				}
			}
		})
	}
	for _, c := range []struct{ name, in, want string }{
		{"line-and-reason", "yaml: line 3: did not find expected key", "配置文件第 3 行: 结构不完整（缺少预期的键、值、结束引号或括号）"},
		{"unknown-reason-not-echoed", "yaml: line 2: something zz-unknown-zz happened", "配置文件第 2 行: 无法解析（YAML/JSON 语法错误）"},
		{"line-in-field-name-ignored", "json: cannot unmarshal number 1e999 into Go struct field .webhook.line 7: of type float64", "配置文件解析失败: 取值类型不符或数值超出范围"},
		{"no-line", "unexpected end of JSON input", "配置文件解析失败: 内容意外结束（缺少结束引号或括号）"},
		{"viper-wrapper", "While parsing config: yaml: unmarshal errors:\n  line 7: mapping key \"fake-leak-k\" already defined at line 6", "配置文件第 7 行: 同一层有重复的键"},
	} {
		t.Run("unit-"+c.name, func(t *testing.T) {
			if got := sanitizeWebhookParseError(errors.New(c.in)); got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

// 第三轮 P3-5：JSON 里合法的 UTF-16 代理对转义正常接受；\u0000 一律拒绝（YAML 的 \0 同样拒绝）
func TestLoadWebhookJSONEscapes(t *testing.T) {
	t.Run("J5-surrogate-pair-ok", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.json", `{"webhook":{"port":"8889","front_domain":"\ud83d\ude00\/x"}}`, nil)
		expectLoaded(t, cfg, err)
		if cfg.FrontDomain != "\U0001F600/x" {
			t.Fatalf("got %q", cfg.FrontDomain)
		}
	})
	t.Run("escaped-slash-only", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.json", `{"webhook":{"port":"8889","front_domain":"http:\/\/f.example"}}`, nil)
		expectLoaded(t, cfg, err)
		if cfg.FrontDomain != "http://f.example" {
			t.Fatalf("got %q", cfg.FrontDomain)
		}
	})
	t.Run("escaped-backslash-not-pair", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.json", `{"webhook":{"port":"8889","front_domain":"\\ud83d\\ude00"}}`, nil)
		expectLoaded(t, cfg, err)
		if cfg.FrontDomain != `\ud83d\ude00` {
			t.Fatalf("got %q", cfg.FrontDomain)
		}
	})
	t.Run("lone-surrogate-rejected", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.json", `{"webhook":{"port":"8889","front_domain":"\ud83dx"}}`, nil)
		expectStartupFailure(t, err)
	})
	for _, c := range []struct{ name, file, body string }{
		{"J10-json-nul-value", "webhook.json", `{"webhook":{"port":"8889","front_domain":"a\u0000b"}}`},
		{"json-nul-key", "webhook.json", `{"webhook":{"port":"8889","x\u0000":"1"}}`},
		{"yaml-nul-value", "webhook.yaml", strictHead + "  front_domain: \"a\\0b\"\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err, _ := strictLoad(t, c.file, c.body, nil)
			expectStartupFailure(t, err, "包含 NUL 字符")
		})
	}
}

// 第三轮 P3-7/J-2：端口与整数只去 ASCII 空白（空格、\t、\r、\n）；NBSP、\v、\f、全角空格等报错
func TestLoadWebhookPortTrimsOnlyASCIISpace(t *testing.T) {
	for _, c := range []struct{ name, val string }{{"space-crlf", " 8890\r\n"}, {"tab", "\t8890\t"}} {
		t.Run("ok-"+c.name, func(t *testing.T) {
			cfg, err, _ := strictLoad(t, "webhook.yaml", strictHead, map[string]string{"WEBHOOK_PORT": c.val})
			expectLoaded(t, cfg, err)
			if cfg.Port != "8890" || viper.GetString("webhook.port") != "8890" {
				t.Fatalf("got %q", cfg.Port)
			}
		})
	}
	for _, c := range []struct{ name, val string }{
		{"nbsp-lead", "\u00a08890"}, {"nbsp-trail", "8890\u00a0"}, {"vtab", "\v8890"}, {"formfeed", "8890\f"}, {"ideographic", "\u30008890"}, {"nel", "8890\u0085"},
	} {
		t.Run("reject-env-"+c.name, func(t *testing.T) {
			_, err, _ := strictLoad(t, "webhook.yaml", strictHead, map[string]string{"WEBHOOK_PORT": c.val})
			expectStartupFailure(t, err, "WEBHOOK_PORT")
		})
	}
	t.Run("reject-file-nbsp", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", "webhook:\n  port: \"\\u00a08890\"\n", nil)
		expectStartupFailure(t, err, "webhook.port")
	})
	t.Run("reject-int-ideographic", func(t *testing.T) {
		_, err, _ := strictLoad(t, "webhook.yaml", strictHead, map[string]string{"WEBHOOK_FIXED_WORKERS": "\u300010"})
		expectStartupFailure(t, err, "WEBHOOK_FIXED_WORKERS")
	})
}

// 第三轮 P3-7/J-2：URL 类字段去首尾空白；作为拼接前缀的 URL 再去尾斜杠，避免拼出 hook/ /<token>
func TestLoadWebhookURLFieldsTrimmed(t *testing.T) {
	t.Run("env-group-message-api-trailing-space", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.yaml", strictHead, map[string]string{"WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API": "https://open.feishu.cn/open-apis/bot/v2/hook/ "})
		expectLoaded(t, cfg, err)
		const want = "https://open.feishu.cn/open-apis/bot/v2/hook"
		if cfg.ImFeishu.GroupMessageAPI != want || viper.GetString("webhook.im_feishu.group_message_api") != want {
			t.Fatalf("got %q / %q", cfg.ImFeishu.GroupMessageAPI, viper.GetString("webhook.im_feishu.group_message_api"))
		}
	})
	t.Run("file-base-urls", func(t *testing.T) {
		body := strictHead + "  front_domain: \" http://f.example/ \"\n  backend_domain: \"http://b.example/\\t\"\n  alert_manager_api: \"http://am.example:9093/ \"\n"
		cfg, err, _ := strictLoad(t, "webhook.yaml", body, nil)
		expectLoaded(t, cfg, err)
		if cfg.FrontDomain != "http://f.example" || cfg.BackendDomain != "http://b.example" || cfg.AlertManagerAPI != "http://am.example:9093" {
			t.Fatalf("got %q %q %q", cfg.FrontDomain, cfg.BackendDomain, cfg.AlertManagerAPI)
		}
		if viper.GetString("webhook.alert_manager_api") != "http://am.example:9093" {
			t.Fatalf("全局 viper 也应是去空白后的值")
		}
	})
	t.Run("env-token-api-keeps-trailing-slash", func(t *testing.T) {
		cfg, err, _ := strictLoad(t, "webhook.yaml", strictHead, map[string]string{"WEBHOOK_IM_FEISHU_TENANT_ACCESS_TOKEN_API": "\thttps://t.example/auth/ \n"})
		expectLoaded(t, cfg, err)
		if cfg.ImFeishu.TenantAccessTokenAPI != "https://t.example/auth/" {
			t.Fatalf("got %q", cfg.ImFeishu.TenantAccessTokenAPI)
		}
	})
}

// 第三轮 P3-5：白名单按小写匹配（yaml 报错里的 UTF-8 是大写）
func TestLoadWebhookParseErrorInvalidUTF8Reason(t *testing.T) {
	_, err, _ := strictLoad(t, "webhook.yaml", strictHead+"  front_domain: \"a\xffb\"\n", nil)
	expectStartupFailure(t, err, "读取Webhook配置文件失败: 配置文件解析失败: 不是合法的 UTF-8 编码")
}

// 第三轮 J-3：配置文件存在但读不出来（这里是同名目录）时启动失败，不当作文件缺失
func TestLoadWebhookUnreadableConfigFails(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "webhook.yaml")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	viper.Reset()
	t.Cleanup(viper.Reset)
	clearAllWebhookEnv(t)
	_, _, err := loadWebhookFromFile(dir)
	expectStartupFailure(t, err, "读取Webhook配置文件失败: 无法读取文件")
}
