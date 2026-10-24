package config

import (
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

// webhookLeafSpec 测试侧独立维护的 WebhookConfig 叶子键清单（与反射生成的表互相校验）
type webhookLeafSpec struct {
	key   string
	env   string
	isInt bool
}

var webhookLeafSpecs = []webhookLeafSpec{
	{"webhook.alert_manager_api", "WEBHOOK_ALERT_MANAGER_API", false},
	{"webhook.backend_domain", "WEBHOOK_BACKEND_DOMAIN", false},
	{"webhook.common_map_renew_interval_seconds", "WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS", true},
	{"webhook.default_upgrade_minutes", "WEBHOOK_DEFAULT_UPGRADE_MINUTES", true},
	{"webhook.fixed_workers", "WEBHOOK_FIXED_WORKERS", true},
	{"webhook.front_domain", "WEBHOOK_FRONT_DOMAIN", false},
	{"webhook.im_feishu.group_message_api", "WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API", false},
	{"webhook.im_feishu.private_robot_app_id", "WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_ID", false},
	{"webhook.im_feishu.private_robot_app_secret", "WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_SECRET", false},
	{"webhook.im_feishu.request_timeout_seconds", "WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS", true},
	{"webhook.im_feishu.tenant_access_token_api", "WEBHOOK_IM_FEISHU_TENANT_ACCESS_TOKEN_API", false},
	{"webhook.port", "WEBHOOK_PORT", false},
}

// leakMarkers 测试里写进配置的“值”都带这些片段，报错/告警中出现即视为回显了原值
var leakMarkers = []string{"fake-leak", "7391"}

func assertNoLeak(t *testing.T, msg string) {
	t.Helper()
	for _, m := range leakMarkers {
		if strings.Contains(msg, m) {
			t.Fatalf("输出不应回显配置值（含 %q）", m) // 不打印 msg 本身
		}
	}
}

// clearAllWebhookEnv 把全部 WEBHOOK_* 叶子变量和父键变量置空（视为未设置）并注入假 DSN
func clearAllWebhookEnv(t *testing.T) {
	t.Helper()
	t.Setenv(WebhookMySQLAddrEnv, fakeWebhookDSN)
	for _, s := range webhookLeafSpecs {
		t.Setenv(s.env, "")
	}
	// 父键变量非空会直接启动失败，测试默认置空
	t.Setenv("WEBHOOK", "")
	t.Setenv("WEBHOOK_IM_FEISHU", "")
}

// nestedWebhookYAML 把 webhook.a.b 形式的键写成嵌套 YAML：webhook:\n  a:\n    b: val
func nestedWebhookYAML(key, val string) string {
	parts := strings.Split(key, ".")
	var b strings.Builder
	for i, p := range parts {
		b.WriteString(strings.Repeat("  ", i) + p + ":")
		if i == len(parts)-1 {
			b.WriteString(" " + val)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// 断言加载失败（不可继续）、报错只含键名（含“类型错误”），不回显值，且（在空白全局上）不发布任何键
func assertWebhookTypeError(t *testing.T, err error, key string) {
	t.Helper()
	if err == nil || IsNonFatal(err) {
		t.Fatalf("%s 写错类型应启动失败, got err=%v", key, err != nil)
	}
	msg := err.Error()
	if !strings.Contains(msg, "类型错误") || !strings.Contains(msg, key) {
		t.Fatalf("报错应是类型错误并列出键名 %s: %s", key, msg)
	}
	assertNoLeak(t, msg)
	if len(viper.AllKeys()) != 0 {
		t.Fatal("失败时不应发布任何键到全局 viper")
	}
}

// P2-1：标量叶子写成 map（{}、非空 map）或列表时启动失败，报错只写键名；即使环境变量覆盖了该键，文件写错仍失败
func TestLoadWebhookRejectsLeafWrittenAsMapOrList(t *testing.T) {
	for _, s := range webhookLeafSpecs {
		// 子测试名只用标签，不带取值（-v 日志里不出现测试假值）
		for _, bc := range []struct{ name, val string }{
			{"empty-map", "{}"},
			{"map", "{x: fake-leak-7391}"},
			{"map-kv", "{k: v}"},
			{"flow-list", "[fake-leak-7391]"},
			{"empty-list", "[]"},
			{"block-list", "\n" + strings.Repeat("  ", strings.Count(s.key, ".")+1) + "- fake-leak-7391"},
		} {
			bad := bc.val
			for _, withEnv := range []bool{false, true} {
				t.Run(s.key+"/"+bc.name+"/env="+strconv.FormatBool(withEnv), func(t *testing.T) {
					viper.Reset()
					t.Cleanup(viper.Reset)
					clearAllWebhookEnv(t)
					if withEnv {
						v := "env-value"
						if s.isInt || s.key == webhookPortKey {
							v = "7"
						}
						t.Setenv(s.env, v)
					}
					_, _, err := loadWebhookFromFile(writeWebhookYAML(t, nestedWebhookYAML(s.key, bad)))
					assertWebhookTypeError(t, err, s.key)
				})
			}
		}
	}
}

// TL 给的三个具体例子：以前读成零值且启动成功
func TestLoadWebhookLeafMapExamplesFromReview(t *testing.T) {
	for _, tc := range []struct{ key, body string }{
		{"webhook.front_domain", "webhook:\n  front_domain: {}\n"},
		{"webhook.im_feishu.request_timeout_seconds", "webhook:\n  im_feishu:\n    request_timeout_seconds: {x: -1}\n"},
		{"webhook.im_feishu.private_robot_app_secret", "webhook:\n  im_feishu:\n    private_robot_app_secret: {k: fake-leak-v}\n"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			clearAllWebhookEnv(t)
			_, _, err := loadWebhookFromFile(writeWebhookYAML(t, tc.body))
			assertWebhookTypeError(t, err, tc.key)
		})
	}
}

// 父键（webhook.im_feishu、根节点 webhook）写成标量或列表时启动失败（bb410eb 之前就存在的静默忽略）
func TestLoadWebhookRejectsParentWrittenAsScalarOrList(t *testing.T) {
	for _, tc := range []struct{ name, key, body string }{
		{"feishu-int", "webhook.im_feishu", "webhook:\n  im_feishu: 7391\n"},
		{"feishu-str", "webhook.im_feishu", "webhook:\n  im_feishu: fake-leak-str\n"},
		{"feishu-quoted", "webhook.im_feishu", "webhook:\n  im_feishu: \"fake-leak-q\"\n"},
		{"feishu-bool", "webhook.im_feishu", "webhook:\n  im_feishu: true\n"},
		{"feishu-list", "webhook.im_feishu", "webhook:\n  im_feishu: [fake-leak-a, b]\n"},
		{"feishu-empty-list", "webhook.im_feishu", "webhook:\n  im_feishu: []\n"},
		{"feishu-block-list", "webhook.im_feishu", "webhook:\n  im_feishu:\n    - request_timeout_seconds: 7391\n"},
		{"root-scalar", "webhook", "webhook: fake-leak-root\n"},
		{"root-int", "webhook", "webhook: 7391\n"},
		{"root-list", "webhook", "webhook:\n  - port: \"7391\"\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			clearAllWebhookEnv(t)
			_, _, err := loadWebhookFromFile(writeWebhookYAML(t, tc.body))
			assertWebhookTypeError(t, err, tc.key)
		})
	}
}

// 父键写成 ~ / 空 / {}：合法，子键取环境变量/默认值，cfg 与全局一致
func TestLoadWebhookParentNullOrEmptyMapAccepted(t *testing.T) {
	for _, body := range []string{
		"webhook:\n  im_feishu: ~\n",
		"webhook:\n  im_feishu: null\n",
		"webhook:\n  im_feishu:\n",
		"webhook:\n  im_feishu: {}\n",
	} {
		t.Run(strconv.Quote(body), func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			clearAllWebhookEnv(t)
			t.Setenv("WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS", "7")
			cfg, _, err := loadWebhookFromFile(writeWebhookYAML(t, body))
			if err != nil {
				t.Fatalf("父键为空应正常加载: %v", err)
			}
			if cfg.ImFeishu.RequestTimeoutSeconds != 7 || cfg.ImFeishu.GroupMessageAPI != "https://open.feishu.cn/open-apis/bot/v2/hook" {
				t.Fatal("子键应取环境变量/默认值")
			}
			if n := compareWebhookCfgWithGlobal(t, reflect.ValueOf(*cfg), "webhook"); n != 12 {
				t.Fatalf("应比对 12 个字段, got %d", n)
			}
		})
	}
}

// 叶子写成 ~：等同未写，取默认值；环境变量仍可覆盖
func TestLoadWebhookLeafNullUsesDefaultOrEnv(t *testing.T) {
	defaults := viper.New()
	setWebhookDefaults(defaults)
	for _, s := range webhookLeafSpecs {
		for _, withEnv := range []bool{false, true} {
			t.Run(s.key+"/env="+strconv.FormatBool(withEnv), func(t *testing.T) {
				viper.Reset()
				t.Cleanup(viper.Reset)
				clearAllWebhookEnv(t)
				want := defaults.GetString(s.key)
				if withEnv {
					want = "env-value"
					if s.isInt || s.key == webhookPortKey {
						want = "7"
					}
					t.Setenv(s.env, want)
				}
				cfg, _, err := loadWebhookFromFile(writeWebhookYAML(t, nestedWebhookYAML(s.key, "~")))
				if err != nil {
					t.Fatalf("叶子为 ~ 应正常加载: %v", err)
				}
				if viper.GetString(s.key) != want {
					t.Fatalf("%s 应取默认值/环境变量", s.key) // 不输出值
				}
				compareWebhookCfgWithGlobal(t, reflect.ValueOf(*cfg), "webhook")
			})
		}
	}
}

// 同前缀键（front_domain 与 front_domain_x 等）：真正的叶子正常生效，同前缀的未知键不发布、只告警一次
func TestLoadWebhookSamePrefixKeys(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	clearAllWebhookEnv(t)
	warns := captureConfigWarn(t)
	body := "webhook:\n" +
		"  port: \"8889\"\n" +
		"  port_x: fake-leak-p\n" +
		"  front_domain: file-front.example\n" +
		"  front_domain_x: fake-leak-x\n" +
		"  im_feishu:\n" +
		"    request_timeout_seconds: 7\n" +
		"    request_timeout_seconds_x: {a: fake-leak-n}\n" +
		"  im_feishu_x: {request_timeout_seconds: 7391}\n"
	cfg, _, err := loadWebhookFromFile(writeWebhookYAML(t, body))
	if err != nil {
		t.Fatalf("同前缀的未知键不应导致启动失败: %v", err)
	}
	if cfg.FrontDomain != "file-front.example" || viper.GetString("webhook.front_domain") != "file-front.example" {
		t.Fatal("front_domain 不应因存在 front_domain_x 而被跳过")
	}
	if cfg.ImFeishu.RequestTimeoutSeconds != 7 || viper.GetInt("webhook.im_feishu.request_timeout_seconds") != 7 {
		t.Fatal("request_timeout_seconds 不应因存在同前缀键而被跳过")
	}
	for _, k := range []string{"webhook.port_x", "webhook.front_domain_x", "webhook.im_feishu.request_timeout_seconds_x", "webhook.im_feishu_x", "webhook.im_feishu_x.request_timeout_seconds"} {
		if viper.IsSet(k) {
			t.Fatalf("未知键 %s 不应发布到全局", k)
		}
	}
	if len(*warns) != 1 {
		t.Fatalf("未知键应只告警一次, got %d", len(*warns))
	}
	w := (*warns)[0]
	if !strings.Contains(w, "webhook.front_domain_x, webhook.im_feishu.request_timeout_seconds_x, webhook.im_feishu_x, webhook.port_x") {
		t.Fatalf("告警应按键名排序列出全部未知键: %s", w)
	}
	assertNoLeak(t, w)
}

// 未知键：不发布到全局、stderr 只告警一次（只写键名）、不拒绝启动；没有未知键时不告警
func TestLoadWebhookUnknownKeysWarnOnceNotPublished(t *testing.T) {
	// 第二轮：仓库 webhook.yaml 已删掉 7 个没有代码读取的键（alert_receive_queue_size、http_request_global_timeout_seconds、
	// min_workers、max_workers、scale_threshold、scale_interval、im_feishu.private_message_api），不再告警
	t.Run("repo-file", func(t *testing.T) {
		viper.Reset()
		t.Cleanup(viper.Reset)
		clearAllWebhookEnv(t)
		warns := captureConfigWarn(t)
		if _, _, err := loadWebhookFromFile(webhookConfigFile); err != nil {
			t.Fatalf("仓库 webhook.yaml 应正常加载: %v", err)
		}
		if len(*warns) != 0 {
			t.Fatalf("仓库 webhook.yaml 不应再有未知键告警, got %d", len(*warns))
		}
		var want12 []string
		for _, s := range webhookLeafSpecs {
			want12 = append(want12, s.key)
		}
		got := viper.AllKeys()
		sort.Strings(got)
		if !reflect.DeepEqual(got, want12) {
			t.Fatalf("全局只应发布 12 个已知叶子键, got %v", got)
		}
	})
	t.Run("nested-unknown-map", func(t *testing.T) {
		viper.Reset()
		t.Cleanup(viper.Reset)
		clearAllWebhookEnv(t)
		warns := captureConfigWarn(t)
		if _, _, err := loadWebhookFromFile(writeWebhookYAML(t, "webhook:\n  foo:\n    bar: fake-leak-b\n    baz: [7391]\n  zzz: fake-leak-z\n")); err != nil {
			t.Fatalf("未知键不应拒绝启动: %v", err)
		}
		if len(*warns) != 1 || !strings.Contains((*warns)[0], "webhook.foo, webhook.zzz") || strings.Contains((*warns)[0], "webhook.foo.bar") {
			t.Fatalf("应只告警一次，列出 webhook.foo, webhook.zzz（不展开子键）: %v", *warns)
		}
		assertNoLeak(t, (*warns)[0])
		if viper.IsSet("webhook.foo") || viper.IsSet("webhook.foo.bar") || viper.IsSet("webhook.zzz") {
			t.Fatal("未知键不应发布到全局")
		}
	})
	t.Run("no-unknown", func(t *testing.T) {
		viper.Reset()
		t.Cleanup(viper.Reset)
		clearAllWebhookEnv(t)
		warns := captureConfigWarn(t)
		if _, _, err := loadWebhookFromFile(writeWebhookYAML(t, "webhook:\n  port: \"8889\"\n  im_feishu:\n    request_timeout_seconds: 7\n")); err != nil {
			t.Fatal(err)
		}
		if len(*warns) != 0 {
			t.Fatalf("没有未知键时不应告警: %v", *warns)
		}
	})
}

// webhookIntBound 整数字段的上下限（测试侧独立维护）
var webhookIntBounds = []struct {
	key, env string
	min, max int
	unit     time.Duration
}{
	{"webhook.fixed_workers", "WEBHOOK_FIXED_WORKERS", 1, 1024, 0},
	{"webhook.default_upgrade_minutes", "WEBHOOK_DEFAULT_UPGRADE_MINUTES", 1, 10080, time.Minute},
	{"webhook.common_map_renew_interval_seconds", "WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS", 1, 86400, time.Second},
	{"webhook.im_feishu.request_timeout_seconds", "WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS", 1, 600, time.Second},
}

// rangeErrFor 越界报错：取自环境变量写“KEY（环境变量 ENV）”，取自配置文件写“配置文件中的 KEY ”
func rangeErrFor(src, key, env string, min, max int) string {
	label := key + "（环境变量 " + env + "）"
	if src != "env" {
		label = "配置文件中的 " + key + " "
	}
	return "Webhook 整数配置超出允许范围: " + label + "必须在 " + strconv.Itoa(min) + " 到 " + strconv.Itoa(max) + " 之间"
}

// 整数字段只接受十进制整数：YAML 按节点原始文本校验（不加引号的 10.5 / 010 / 0x10 / +3 等不再被截断或按进制解析），
// 环境变量同样严格（第二轮 N5：先去首尾 ASCII 空白，" 10"、"10 " 改为接受，见 TestLoadWebhookIntEnvTrimsASCIISpace）
func TestLoadWebhookIntRawTextStrict(t *testing.T) {
	yamlBad := []string{"10.5", "-0.5", "0x10", "0X10", "+3", "010", "00", "-0", "1e1", "10.0", "0o10", "0b1", "1_0", "true", ".inf", "\"010\"", "\"0x10\"", "\"+3\"", "\"10.5\"", "\" 10\"", "\"10 \"", "\"\"", "'-0.5'"}
	envBad := []string{"010", "0x10", "+3", "10.5", "-0.5", "1 0", "\v10", "-0", "1_0", "1e1", "abc", "１０"}
	for _, b := range webhookIntBounds {
		for _, bad := range yamlBad {
			t.Run(b.key+"/yaml/"+bad, func(t *testing.T) {
				viper.Reset()
				t.Cleanup(viper.Reset)
				clearAllWebhookEnv(t)
				_, _, err := loadWebhookFromFile(writeWebhookYAML(t, nestedWebhookYAML(b.key, bad)))
				assertWebhookIntSyntaxError(t, err, "yaml", b.key, b.env, bad)
			})
		}
		for _, bad := range envBad {
			t.Run(b.key+"/env/"+strconv.Quote(bad), func(t *testing.T) {
				viper.Reset()
				t.Cleanup(viper.Reset)
				clearAllWebhookEnv(t)
				t.Setenv(b.env, bad)
				_, _, err := loadWebhookFromFile(writeWebhookYAML(t, "webhook:\n  port: \"8889\"\n"))
				assertWebhookIntSyntaxError(t, err, "env", b.key, b.env, bad)
			})
		}
		for _, good := range []struct{ src, raw string }{{"yaml", "7"}, {"yaml", "\"7\""}, {"yaml", "'7'"}, {"env", "7"}} {
			t.Run(b.key+"/ok/"+good.src+"/"+good.raw, func(t *testing.T) {
				viper.Reset()
				t.Cleanup(viper.Reset)
				clearAllWebhookEnv(t)
				body := nestedWebhookYAML(b.key, good.raw)
				if good.src == "env" {
					body = "webhook:\n  port: \"8889\"\n"
					t.Setenv(b.env, good.raw)
				}
				cfg, _, err := loadWebhookFromFile(writeWebhookYAML(t, body))
				if err != nil {
					t.Fatalf("十进制整数应通过: %v", err)
				}
				if viper.GetInt(b.key) != 7 {
					t.Fatalf("全局应为 7, got %d", viper.GetInt(b.key))
				}
				compareWebhookCfgWithGlobal(t, reflect.ValueOf(*cfg), "webhook")
			})
		}
	}
}

func assertWebhookIntSyntaxError(t *testing.T, err error, src, key, env, bad string) {
	t.Helper()
	if err == nil || IsNonFatal(err) {
		t.Fatalf("%s=%q 应启动失败", key, bad)
	}
	msg := err.Error()
	want := key + "（环境变量 " + env + "）应为十进制整数"
	if src != "env" {
		want = "配置文件中的 " + key + " 应为十进制整数"
	}
	if !strings.Contains(msg, want) || (src != "env" && strings.Contains(msg, "环境变量")) {
		t.Fatalf("报错应按来源列出键名（环境变量来源带变量名）并说明需要十进制整数: %s", msg)
	}
	if v := strings.Trim(strings.TrimSpace(bad), `"'`); len(v) >= 2 && strings.Contains(msg, v) {
		t.Fatalf("报错不应包含取值: %s", msg)
	}
	if len(viper.AllKeys()) != 0 {
		t.Fatal("失败时不应发布任何键到全局 viper")
	}
}

// 业务上限：max 通过、max+1 与超出 int64 的值拒绝；报错只含键名、变量名和范围；上限乘以时间单位不溢出
func TestLoadWebhookIntUpperBounds(t *testing.T) {
	for _, b := range webhookIntBounds {
		if b.unit != 0 {
			if d := time.Duration(b.max) * b.unit; d <= 0 || d/b.unit != time.Duration(b.max) {
				t.Fatalf("%s 上限乘以时间单位溢出", b.key)
			}
		}
		for _, src := range []string{"env", "yaml", "yaml-quoted"} {
			load := func(t *testing.T, val string) (*WebhookConfig, error) {
				viper.Reset()
				t.Cleanup(viper.Reset)
				clearAllWebhookEnv(t)
				body := "webhook:\n  port: \"8889\"\n"
				switch src {
				case "env":
					t.Setenv(b.env, val)
				case "yaml":
					body = nestedWebhookYAML(b.key, val)
				case "yaml-quoted":
					body = nestedWebhookYAML(b.key, strconv.Quote(val))
				}
				cfg, _, err := loadWebhookFromFile(writeWebhookYAML(t, body))
				return cfg, err
			}
			t.Run(b.key+"/"+src+"/max", func(t *testing.T) {
				cfg, err := load(t, strconv.Itoa(b.max))
				if err != nil {
					t.Fatalf("上限 %d 应通过: %v", b.max, err)
				}
				if viper.GetInt(b.key) != b.max {
					t.Fatalf("全局应为 %d", b.max)
				}
				compareWebhookCfgWithGlobal(t, reflect.ValueOf(*cfg), "webhook")
			})
			for _, bad := range []string{strconv.Itoa(b.max + 1), strconv.Itoa(b.max + 7391), "9223372036854775807", "99999999999999999999", "-99999999999999999999"} {
				t.Run(b.key+"/"+src+"/"+bad, func(t *testing.T) {
					_, err := load(t, bad)
					if err == nil || IsNonFatal(err) {
						t.Fatalf("%s=%s 应启动失败", b.key, bad)
					}
					if want := rangeErrFor(src, b.key, b.env, b.min, b.max); err.Error() != want {
						t.Fatalf("报错应只描述键名、来源和范围\n got: %s\nwant: %s", err.Error(), want)
					}
					if len(viper.AllKeys()) != 0 {
						t.Fatal("失败时不应发布任何键到全局 viper")
					}
				})
			}
		}
	}
}

// 解码错误脱敏：只报键名和期望类型，不回显原值
func TestDecodeWebhookConfigErrorHidesValue(t *testing.T) {
	m := webhookValues{
		keys: []string{"webhook.fixed_workers", "webhook.front_domain"},
		vals: map[string]any{"webhook.fixed_workers": "fake-leak-7391", "webhook.front_domain": map[string]any{"x": "fake-leak-m"}},
	}
	_, err := decodeWebhookConfig(m)
	if err == nil {
		t.Fatal("类型不符应报错")
	}
	msg := err.Error()
	if !strings.Contains(msg, "webhook.fixed_workers") || !strings.Contains(msg, "int") {
		t.Fatalf("报错应包含键名和期望类型: %s", msg)
	}
	assertNoLeak(t, msg)
}

// P3-5：全局已有旧值时，各种加载失败都不能改动全局。整个用例只在开头 Reset 一次，逐字段断言
func TestLoadWebhookFailureKeepsPreviousGlobal(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	clearAllWebhookEnv(t)

	// 先成功加载一次，每个字段都给出与默认值不同的旧值
	old := map[string]string{
		"WEBHOOK_PORT":                               "7101",
		"WEBHOOK_FIXED_WORKERS":                      "3",
		"WEBHOOK_FRONT_DOMAIN":                       "old-front.example",
		"WEBHOOK_BACKEND_DOMAIN":                     "old-back.example",
		"WEBHOOK_DEFAULT_UPGRADE_MINUTES":            "11",
		"WEBHOOK_ALERT_MANAGER_API":                  "http://old-am.example",
		"WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS":  "12",
		"WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API":        "http://old-group.example",
		"WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS":  "13",
		"WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_ID":     "old-app-id",
		"WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_SECRET": "old-app-secret-value",
		"WEBHOOK_IM_FEISHU_TENANT_ACCESS_TOKEN_API":  "http://old-token.example",
	}
	t.Run("seed", func(t *testing.T) {
		for k, v := range old {
			t.Setenv(k, v)
		}
		cfg, _, err := loadWebhookFromFile(writeWebhookYAML(t, "webhook:\n  port: \"8889\"\n"))
		if err != nil {
			t.Fatalf("预置旧值应加载成功: %v", err)
		}
		if n := compareWebhookCfgWithGlobal(t, reflect.ValueOf(*cfg), "webhook"); n != 12 {
			t.Fatalf("应有 12 个字段, got %d", n)
		}
	})
	snapshot := map[string]any{}
	for _, s := range webhookLeafSpecs {
		if !viper.IsSet(s.key) {
			t.Fatalf("预置后全局应有 %s", s.key)
		}
		snapshot[s.key] = viper.Get(s.key)
	}
	snapKeys := viper.AllKeys()
	sort.Strings(snapKeys)

	cases := []struct {
		name string
		body string
		env  map[string]string
	}{
		{"leaf-empty-map", "webhook:\n  front_domain: {}\n  fixed_workers: 5\n", nil},
		{"leaf-map", "webhook:\n  im_feishu:\n    request_timeout_seconds: {x: -1}\n    private_robot_app_secret: new-secret-value\n", nil},
		{"leaf-list", "webhook:\n  backend_domain: [a]\n  front_domain: new-front.example\n", nil},
		{"parent-scalar", "webhook:\n  im_feishu: 5\n  front_domain: new-front.example\n", nil},
		{"parent-list", "webhook:\n  im_feishu: [a]\n  port: \"7202\"\n", nil},
		{"root-scalar", "webhook: 5\n", nil},
		{"int-octal", "webhook:\n  fixed_workers: 010\n  front_domain: new-front.example\n", nil},
		{"int-float", "webhook:\n  default_upgrade_minutes: 10.5\n", nil},
		{"int-hex-env", "webhook:\n  front_domain: new-front.example\n", map[string]string{"WEBHOOK_FIXED_WORKERS": "0x10"}},
		{"over-max", "webhook:\n  im_feishu:\n    request_timeout_seconds: 601\n  front_domain: new-front.example\n", nil},
		{"negative", "webhook:\n  common_map_renew_interval_seconds: -1\n", nil},
		{"zero-env", "webhook:\n  front_domain: new-front.example\n", map[string]string{"WEBHOOK_DEFAULT_UPGRADE_MINUTES": "0"}},
		{"bad-port", "webhook:\n  port: \"0\"\n  front_domain: new-front.example\n", nil},
		{"unknown-plus-type-error", "webhook:\n  brand_new_key: 1\n  front_domain: {}\n", nil},
		{"flat-format", "port: \"7203\"\nfront_domain: new-front.example\n", nil},
		{"missing-dsn", "webhook:\n  front_domain: new-front.example\n", map[string]string{WebhookMySQLAddrEnv: ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// 不 Reset：全局里是上一次成功加载的旧值
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			if _, _, err := loadWebhookFromFile(writeWebhookYAML(t, tc.body)); err == nil || IsNonFatal(err) {
				t.Fatal("应启动失败")
			}
			for _, s := range webhookLeafSpecs {
				if !viper.IsSet(s.key) || !reflect.DeepEqual(viper.Get(s.key), snapshot[s.key]) {
					t.Errorf("加载失败后全局 %s 被改动", s.key) // 不输出值（含假密钥）
				}
			}
			got := viper.AllKeys()
			sort.Strings(got)
			if !reflect.DeepEqual(got, snapKeys) {
				t.Errorf("加载失败后全局键集合被改动: %v", got)
			}
		})
	}
}

// 反射生成的两张表：父键白名单只有 webhook.im_feishu；叶子表与测试侧清单逐项一致（键名、环境变量、类型）
func TestWebhookKeySchemaFromReflection(t *testing.T) {
	if !reflect.DeepEqual(webhookKeySchema.parents, map[string]bool{"webhook.im_feishu": true}) {
		t.Fatalf("父键白名单应只有 webhook.im_feishu, got %v", webhookKeySchema.parents)
	}
	if len(webhookKeySchema.leafKeys) != len(webhookLeafSpecs) || len(webhookKeySchema.leaves) != len(webhookLeafSpecs) {
		t.Fatalf("叶子数应为 %d, got %d", len(webhookLeafSpecs), len(webhookKeySchema.leafKeys))
	}
	for i, s := range webhookLeafSpecs {
		if webhookKeySchema.leafKeys[i] != s.key {
			t.Fatalf("leafKeys[%d] 应为 %s, got %s", i, s.key, webhookKeySchema.leafKeys[i])
		}
		leaf := webhookKeySchema.leaves[s.key]
		wantKind := reflect.String
		if s.isInt {
			wantKind = reflect.Int
		}
		if leaf.env != s.env || leaf.kind != wantKind {
			t.Fatalf("%s: env/kind 不符: %s %s", s.key, leaf.env, leaf.kind)
		}
	}
	// 整数规则覆盖全部整数叶子
	var ruleKeys, intKeys []string
	for _, r := range webhookIntRules {
		ruleKeys = append(ruleKeys, r.key)
		if webhookKeySchema.leaves[r.key].env != r.env {
			t.Fatalf("%s 规则中的变量名与 env 标签不一致", r.key)
		}
	}
	for _, s := range webhookLeafSpecs {
		if s.isInt {
			intKeys = append(intKeys, s.key)
		}
	}
	sort.Strings(ruleKeys)
	if !reflect.DeepEqual(ruleKeys, intKeys) {
		t.Fatalf("整数规则应覆盖全部整数字段: %v vs %v", ruleKeys, intKeys)
	}
}

// 不支持的字段类型在建表时直接 panic，防止新增字段被静默忽略
func TestBuildWebhookSchemaPanicsOnUnsupportedKind(t *testing.T) {
	type bad struct {
		L []string `mapstructure:"l"`
	}
	defer func() {
		if recover() == nil {
			t.Fatal("应 panic")
		}
	}()
	buildWebhookSchema(reflect.TypeOf(bad{}), "webhook")
}

// YAML 别名与 << 合并键：按展开后的结构校验
func TestLoadWebhookYAMLAliasAndMerge(t *testing.T) {
	t.Run("alias-map-into-leaf", func(t *testing.T) {
		viper.Reset()
		t.Cleanup(viper.Reset)
		clearAllWebhookEnv(t)
		body := "x: &m {k: fake-leak-a}\nwebhook:\n  front_domain: *m\n"
		_, _, err := loadWebhookFromFile(writeWebhookYAML(t, body))
		assertWebhookTypeError(t, err, "webhook.front_domain")
	})
	t.Run("merge-bad-int", func(t *testing.T) {
		viper.Reset()
		t.Cleanup(viper.Reset)
		clearAllWebhookEnv(t)
		body := "base: &b\n  fixed_workers: 010\nwebhook:\n  <<: *b\n"
		_, _, err := loadWebhookFromFile(writeWebhookYAML(t, body))
		if err == nil || !strings.Contains(err.Error(), "webhook.fixed_workers") || !strings.Contains(err.Error(), "十进制整数") {
			t.Fatalf("合并键带入的 010 应被拒绝: %v", err)
		}
	})
	t.Run("merge-ok", func(t *testing.T) {
		viper.Reset()
		t.Cleanup(viper.Reset)
		clearAllWebhookEnv(t)
		body := "base: &b\n  fixed_workers: 9\n  im_feishu: {request_timeout_seconds: 11}\nwebhook:\n  <<: *b\n"
		cfg, _, err := loadWebhookFromFile(writeWebhookYAML(t, body))
		if err != nil || cfg.FixedWorkers != 9 || cfg.ImFeishu.RequestTimeoutSeconds != 11 {
			t.Fatalf("合并键应正常生效: %v", err)
		}
		compareWebhookCfgWithGlobal(t, reflect.ValueOf(*cfg), "webhook")
	})
}

// 字符串叶子写成未加引号的数字/布尔：按文件原始文本取值（0x50 不会变成 80，007 不会丢前导零）
func TestLoadWebhookStringLeafUsesRawText(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	clearAllWebhookEnv(t)
	body := "webhook:\n  port: 8081\n  backend_domain: 007\n  front_domain: true\n  im_feishu:\n    private_robot_app_id: 1e3\n"
	cfg, _, err := loadWebhookFromFile(writeWebhookYAML(t, body))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8081" || cfg.BackendDomain != "007" || cfg.FrontDomain != "true" || cfg.ImFeishu.PrivateRobotAppID != "1e3" {
		t.Fatal("字符串叶子应保留原始文本")
	}
	compareWebhookCfgWithGlobal(t, reflect.ValueOf(*cfg), "webhook")

	viper.Reset()
	if _, _, err := loadWebhookFromFile(writeWebhookYAML(t, "webhook:\n  port: 0x50\n")); err == nil {
		t.Fatal("port: 0x50 应按原始文本判为非法端口")
	}
}

// 第二轮起配置文件只接受 .yaml/.yml/.json：原“非 YAML 格式只靠取值层兜底”的用例改为启动失败，
// 见 webhook_strict_scan_test.go 的 TestLoadWebhookRejectsNonYAMLFormats
