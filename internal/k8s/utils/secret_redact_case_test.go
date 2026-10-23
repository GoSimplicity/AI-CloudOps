package utils

import (
	"encoding/base64"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SEC-26 P2：字段名大小写变体、折行 base64、多文档、重复键、错误回显。
// 本文件只调用公开函数，可在修复前的代码上编译运行（先红后绿）。

const p2Head = "apiVersion: v1\nkind: Secret\nmetadata:\n  name: app\n  namespace: ops\n"

// "<redacted>" 的 base64 为 PHJlZGFjdGVkPg==
const p2PlaceholderB64 = "PHJlZGFjdGVkPg=="

func p2Existing() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ops"},
		Data:       map[string][]byte{"k": []byte(redactFixtureData)},
	}
}

// 模拟 service 层 UpdateSecretByYaml：先恢复占位符，再解析。
func p2Update(y string, existing *corev1.Secret) (*corev1.Secret, error) {
	restored, err := RestoreRedactedSecretYAML(y, existing)
	if err != nil {
		return nil, err
	}
	return YAMLToSecret(restored)
}

// 模拟 service 层 CreateSecretByYaml：先检查占位符，再解析。
func p2Create(y string) (*corev1.Secret, error) {
	has, err := SecretYAMLHasRedactedPlaceholder(y)
	if err != nil {
		return nil, err
	}
	if has {
		return nil, errPlaceholderForTest
	}
	return YAMLToSecret(y)
}

type p2Err string

func (e p2Err) Error() string { return string(e) }

const errPlaceholderForTest = p2Err("YAML 中包含脱敏占位符")

// 最终要写入集群的对象里不得出现占位符（含空白变体）。
func p2AssertNoPlaceholder(t *testing.T, sec *corev1.Secret) {
	t.Helper()
	for k, v := range sec.Data {
		if strings.TrimSpace(string(v)) == SecretRedactedPlaceholder {
			t.Fatalf("data[%s] 将以占位符写入集群", k)
		}
	}
	for k, v := range sec.StringData {
		if strings.TrimSpace(v) == SecretRedactedPlaceholder {
			t.Fatalf("stringData[%s] 将以占位符写入集群", k)
		}
	}
}

// 错误信息不得包含任何测试假值（明文或 base64）。只报告是否泄露，不打印内容。
func p2AssertNoEcho(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	msg := err.Error()
	for i, v := range []string{redactFixtureData, redactFixtureString, redactFixtureNew} {
		if strings.Contains(msg, v) || strings.Contains(msg, base64.StdEncoding.EncodeToString([]byte(v))) {
			t.Fatalf("错误信息回显了第 %d 个测试假值", i)
		}
	}
}

// 应被拒绝：更新和新建两条路径都必须报错，且不回显值。
func p2MustReject(t *testing.T, y string) {
	t.Helper()
	sec, err := p2Update(y, p2Existing())
	if err == nil {
		p2AssertNoPlaceholder(t, sec)
		t.Fatalf("更新路径应拒绝，但被接受")
	}
	p2AssertNoEcho(t, err)
	sec, err = p2Create(y)
	if err == nil {
		p2AssertNoPlaceholder(t, sec)
		t.Fatalf("新建路径应拒绝，但被接受")
	}
	p2AssertNoEcho(t, err)
}

// 1) stringData 字段名大小写变体（e01/e02/e03/e10）：更新应恢复原值，新建应拒绝。
func TestP2StringDataKeyVariants(t *testing.T) {
	for _, field := range []string{"stringData", "StringData", "stringdata", "STRINGDATA", "ſtringData"} {
		t.Run(field, func(t *testing.T) {
			y := p2Head + field + ":\n  k: " + SecretRedactedPlaceholder + "\n"
			sec, err := p2Update(y, p2Existing())
			if err != nil {
				t.Fatalf("更新路径失败: %v", err)
			}
			p2AssertNoPlaceholder(t, sec)
			if string(sec.Data["k"]) != redactFixtureData {
				t.Fatal("更新路径未从现有 Secret 恢复 k")
			}
			if _, ok := sec.StringData["k"]; ok {
				t.Fatal("占位符键应从 stringData 移除")
			}
			if _, err := p2Create(y); err == nil {
				t.Fatal("新建路径应拒绝占位符")
			} else {
				p2AssertNoEcho(t, err)
			}
		})
	}
}

// 2) data 字段名大小写变体 + 裸占位符（e04）：更新应恢复原值，新建应拒绝。
func TestP2DataKeyVariantsBarePlaceholder(t *testing.T) {
	for _, field := range []string{"data", "Data", "DATA", "dAtA"} {
		t.Run(field, func(t *testing.T) {
			y := p2Head + field + ":\n  k: " + SecretRedactedPlaceholder + "\n  s: " + SecretRedactedPlaceholder + "\n"
			existing := p2Existing()
			existing.StringData = map[string]string{"s": redactFixtureString}
			sec, err := p2Update(y, existing)
			if err != nil {
				t.Fatalf("更新路径失败: %v", err)
			}
			p2AssertNoPlaceholder(t, sec)
			if string(sec.Data["k"]) != redactFixtureData {
				t.Fatal("更新路径未从现有 Secret 的 data 恢复 k")
			}
			if string(sec.Data["s"]) != redactFixtureString {
				t.Fatal("更新路径未从现有 Secret 的 stringData 恢复 s")
			}
			if _, err := p2Create(y); err == nil {
				t.Fatal("新建路径应拒绝占位符")
			} else {
				p2AssertNoEcho(t, err)
			}
		})
	}
}

// 3) data 下 base64 编码的占位符：折行（e05/e08）、转义（d01-d03）、单行。两条路径都应拒绝。
func TestP2DataEncodedPlaceholderRejected(t *testing.T) {
	cases := map[string]string{
		"Data_folded":          "Data:\n  k: |\n    PHJlZGFj\n    dGVkPg==\n",
		"data_folded":          "data:\n  k: |\n    PHJlZGFj\n    dGVkPg==\n",
		"Data_folded_trailing": "Data:\n  k: |\n    PHJlZGFj\n    dGVkPg==\n---\n",
		"DATA_escaped":         "DATA:\n  k: \"\\x50HJlZGFjdGVkPg==\"\n",
		"Data_plain_b64":       "Data:\n  k: " + p2PlaceholderB64 + "\n",
		"data_plain_b64":       "data:\n  k: " + p2PlaceholderB64 + "\n",
		"data_b64_with_space":  "data:\n  k: " + base64.StdEncoding.EncodeToString([]byte(SecretRedactedPlaceholder+" ")) + "\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) { p2MustReject(t, p2Head+body) })
	}
}

// 4) stringData 带空白的占位符变体（d09/d10）：两条路径都应拒绝。
func TestP2StringDataWhitespacePlaceholderRejected(t *testing.T) {
	cases := map[string]string{
		"trailing_space":   "stringData:\n  k: \"" + SecretRedactedPlaceholder + " \"\n",
		"trailing_newline": "stringData:\n  k: \"" + SecretRedactedPlaceholder + "\\n\"\n",
		"StringData_space": "StringData:\n  k: \" " + SecretRedactedPlaceholder + "\"\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) { p2MustReject(t, p2Head+body) })
	}
}

// 5) 多文档 YAML（p02）：两条路径都应拒绝。
func TestP2MultiDocumentRejected(t *testing.T) {
	second := "---\n" + p2Head + "stringData:\n  k: " + SecretRedactedPlaceholder + "\n"
	cases := map[string]string{
		"two_docs":         p2Head + "stringData:\n  k: " + redactFixtureNew + "\n" + second,
		"second_has_token": p2Head + "type: Opaque\n" + second,
		"trailing_sep":     p2Head + "type: Opaque\n---\n",
		"doc_end_then_sep": p2Head + "type: Opaque\n...\n---\n" + p2Head,
		"second_broken":    p2Head + "type: Opaque\n---\n: : [\n",
	}
	for name, y := range cases {
		t.Run(name, func(t *testing.T) { p2MustReject(t, y) })
	}
}

// 6) 重复键（p01），含只差大小写：两条路径都应拒绝。
func TestP2DuplicateKeysRejected(t *testing.T) {
	cases := map[string]string{
		"exact_stringData":   p2Head + "stringData:\n  k: " + redactFixtureNew + "\nstringData:\n  k: " + SecretRedactedPlaceholder + "\n",
		"exact_data":         p2Head + "data:\n  k: " + p2PlaceholderB64 + "\ndata:\n  j: " + p2PlaceholderB64 + "\n",
		"case_stringData":    p2Head + "stringData:\n  k: " + redactFixtureNew + "\nStringData:\n  k: " + SecretRedactedPlaceholder + "\n",
		"case_data":          p2Head + "data:\n  k: " + SecretRedactedPlaceholder + "\nData:\n  j: " + p2PlaceholderB64 + "\n",
		"case_data_upper":    p2Head + "data:\n  k: " + SecretRedactedPlaceholder + "\nDATA:\n  j: " + SecretRedactedPlaceholder + "\n",
		"inner_key_exact":    p2Head + "stringData:\n  k: " + redactFixtureNew + "\n  k: " + SecretRedactedPlaceholder + "\n",
		"case_kind":          p2Head + "Kind: Secret\nstringData:\n  k: " + redactFixtureNew + "\n",
		"case_metadata":      p2Head + "Metadata:\n  name: other\nstringData:\n  k: " + redactFixtureNew + "\n",
		"case_metadata_name": "apiVersion: v1\nkind: Secret\nmetadata:\n  name: app\n  Name: other\n  namespace: ops\nstringData:\n  k: " + redactFixtureNew + "\n",
	}
	for name, y := range cases {
		t.Run(name, func(t *testing.T) { p2MustReject(t, y) })
	}
}

// 7) 解析错误不得回显值：非字符串键、类型标签错误、非法 base64。
func TestP2ErrorsDoNotEchoValues(t *testing.T) {
	newB64 := base64.StdEncoding.EncodeToString([]byte(redactFixtureNew))
	cases := map[string]string{
		"null_key_data":       p2Head + "data:\n  ~: " + newB64 + "\n",
		"null_key_stringData": p2Head + "stringData:\n  ~: " + redactFixtureNew + "\n",
		"int_key_data":        p2Head + "data:\n  ? [a]\n  : " + newB64 + "\n",
		"bad_tag":             p2Head + "stringData:\n  k: !!int " + redactFixtureNew + "\n",
		"bad_base64":          p2Head + "data:\n  k: " + redactFixtureNew + "\n",
	}
	for name, y := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := p2Update(y, p2Existing())
			if err == nil {
				t.Fatal("更新路径应报错")
			}
			p2AssertNoEcho(t, err)
			_, err = p2Create(y)
			if err == nil {
				t.Fatal("新建路径应报错")
			}
			p2AssertNoEcho(t, err)
		})
	}
}

// 8) 正常输入保持兼容（对照组）：规范键、带 --- 前导、数值转字符串、合法折行 base64。
func TestP2CanonicalInputsStillAccepted(t *testing.T) {
	newB64 := base64.StdEncoding.EncodeToString([]byte(redactFixtureNew))
	folded := newB64[:8] + "\n    " + newB64[8:]
	t.Run("leading_separator", func(t *testing.T) {
		y := "---\n" + p2Head + "stringData:\n  k: " + SecretRedactedPlaceholder + "\n"
		sec, err := p2Update(y, p2Existing())
		if err != nil {
			t.Fatal(err)
		}
		if string(sec.Data["k"]) != redactFixtureData {
			t.Fatal("未恢复")
		}
	})
	t.Run("doc_end_marker", func(t *testing.T) {
		y := p2Head + "stringData:\n  k: " + redactFixtureNew + "\n...\n"
		if _, err := p2Create(y); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("number_coercion", func(t *testing.T) {
		y := p2Head + "  labels:\n    port: 8080\nstringData:\n  port: 8080\n  flag: true\n"
		sec, err := p2Create(y)
		if err != nil {
			t.Fatal(err)
		}
		if sec.StringData["port"] != "8080" || sec.Labels["port"] != "8080" || sec.StringData["flag"] != "true" {
			t.Fatal("数值/布尔未按原逻辑转为字符串")
		}
	})
	t.Run("folded_real_base64", func(t *testing.T) {
		y := p2Head + "data:\n  k: |\n    " + folded + "\n  j: " + SecretRedactedPlaceholder + "\n"
		existing := p2Existing()
		existing.Data["j"] = []byte(redactFixtureString)
		sec, err := p2Update(y, existing)
		if err != nil {
			t.Fatal(err)
		}
		if string(sec.Data["k"]) != redactFixtureNew || string(sec.Data["j"]) != redactFixtureString {
			t.Fatal("合法折行 base64 或占位符恢复结果不对")
		}
	})
	t.Run("no_placeholder_unchanged", func(t *testing.T) {
		y := p2Head + "data:\n  k: " + newB64 + "\n"
		out, err := RestoreRedactedSecretYAML(y, p2Existing())
		if err != nil {
			t.Fatal(err)
		}
		if out != y {
			t.Fatal("无占位符时应原样返回")
		}
	})
	t.Run("missing_key_error_names_key_only", func(t *testing.T) {
		y := p2Head + "Data:\n  absent: " + SecretRedactedPlaceholder + "\n"
		_, err := p2Update(y, p2Existing())
		if err == nil || !strings.Contains(err.Error(), "absent") {
			t.Fatalf("应报出缺失的键名, got err=%v", err != nil)
		}
		p2AssertNoEcho(t, err)
	})
}

// 9) YAML 合并键把占位符带进 StringData：更新恢复，新建拒绝。
func TestP2MergeKeyStringData(t *testing.T) {
	y := p2Head + "x: &a\n  k: " + SecretRedactedPlaceholder + "\nStringData:\n  <<: *a\n"
	sec, err := p2Update(y, p2Existing())
	if err != nil {
		t.Fatalf("更新路径失败: %v", err)
	}
	p2AssertNoPlaceholder(t, sec)
	if _, err := p2Create(y); err == nil {
		t.Fatal("新建路径应拒绝占位符")
	}
}

// 10) 每一层单独都要拦住（纵深防御）：Has 判定、Restore 残留检查、YAMLToSecret 兜底各自独立生效。
func TestP2EachLayerRejectsIndependently(t *testing.T) {
	inputs := map[string]string{
		"Data_folded":      p2Head + "Data:\n  k: |\n    PHJlZGFj\n    dGVkPg==\n",
		"stringData_space": p2Head + "stringData:\n  k: \"" + SecretRedactedPlaceholder + " \"\n",
		"STRINGDATA_nl":    p2Head + "STRINGDATA:\n  k: \"" + SecretRedactedPlaceholder + "\\n\"\n",
	}
	for name, y := range inputs {
		t.Run(name, func(t *testing.T) {
			if has, err := SecretYAMLHasRedactedPlaceholder(y); err == nil && !has {
				t.Fatal("SecretYAMLHasRedactedPlaceholder 未识别")
			}
			if _, err := RestoreRedactedSecretYAML(y, p2Existing()); err == nil {
				t.Fatal("RestoreRedactedSecretYAML 未拒绝残留占位符")
			}
			if _, err := YAMLToSecret(y); err == nil {
				t.Fatal("YAMLToSecret 未拒绝占位符")
			}
		})
	}
	// data 裸占位符：Has 必须为 true，YAMLToSecret 必须拒绝（Restore 会恢复，不在此列）
	for _, field := range []string{"data", "Data"} {
		y := p2Head + field + ":\n  k: " + SecretRedactedPlaceholder + "\n"
		if has, err := SecretYAMLHasRedactedPlaceholder(y); err != nil || !has {
			t.Fatalf("%s: SecretYAMLHasRedactedPlaceholder 应返回 true", field)
		}
		if _, err := YAMLToSecret(y); err == nil {
			t.Fatalf("%s: YAMLToSecret 应拒绝", field)
		}
	}
	// 结构问题：YAMLToSecret 自身也要拒绝（不依赖前置检查）
	structural := map[string]string{
		"multi_doc":  p2Head + "type: Opaque\n---\n" + p2Head,
		"dup_exact":  p2Head + "type: Opaque\ntype: Opaque\n",
		"dup_case":   p2Head + "stringData:\n  a: b\nStringData:\n  c: d\n",
		"dup_meta":   p2Head + "  Name: other\n",
		"dup_unicod": p2Head + "stringData:\n  a: b\nſtringData:\n  c: d\n",
	}
	for name, y := range structural {
		if _, err := YAMLToSecret(y); err == nil {
			t.Fatalf("%s: YAMLToSecret 应拒绝", name)
		}
		if _, err := SecretYAMLHasRedactedPlaceholder(y); err == nil {
			t.Fatalf("%s: SecretYAMLHasRedactedPlaceholder 应报错", name)
		}
	}
}

// 11) 只有注释/空文档时保持原行为：Restore 原样返回，不当作多文档。
func TestP2EmptyDocumentCompat(t *testing.T) {
	for _, y := range []string{"# only comment\n", "---\n"} {
		out, err := RestoreRedactedSecretYAML(y, p2Existing())
		if err != nil || out != y {
			t.Fatalf("空文档应原样返回: %q", y)
		}
	}
}
