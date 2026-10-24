package utils

import (
	"encoding/base64"
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// 测试用的明文（非真实凭据）
const (
	redactFixtureData   = "plain-redact-data-fixture-01"
	redactFixtureString = "plain-redact-string-fixture-02"
	redactFixtureNew    = "plain-redact-new-fixture-03"
)

func parseSecretYAML(t *testing.T, y string) *corev1.Secret {
	t.Helper()
	var sec corev1.Secret
	if err := yaml.Unmarshal([]byte(y), &sec); err != nil {
		t.Fatalf("解析恢复后的 YAML 失败: %v", err)
	}
	return &sec
}

// stringData 中的占位符在现有 Secret 的 stringData 中不存在、只在 data 中存在时，
// 应从 stringData 移除并以 data 形式恢复原值（secret_redact.go 中 stringData→data 分支）。
func TestRestoreRedactedSecretYAMLStringDataPlaceholderFallsBackToData(t *testing.T) {
	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ops"},
		Data:       map[string][]byte{"token": []byte(redactFixtureData)},
	}

	t.Run("data 中未出现该键时补入 data", func(t *testing.T) {
		in := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: app\n  namespace: ops\nstringData:\n  token: " + SecretRedactedPlaceholder + "\n"
		out, err := RestoreRedactedSecretYAML(in, existing)
		if err != nil {
			t.Fatal(err)
		}
		sec := parseSecretYAML(t, out)
		if _, ok := sec.StringData["token"]; ok {
			t.Fatal("占位符键应从 stringData 中移除")
		}
		if string(sec.Data["token"]) != redactFixtureData {
			t.Fatalf("应以 data 形式恢复原值, got %d bytes", len(sec.Data["token"]))
		}
		if strings.Contains(out, SecretRedactedPlaceholder) {
			t.Fatal("恢复后的 YAML 不应再含占位符")
		}
	})

	t.Run("data 中已有同名新值时以 data 为准", func(t *testing.T) {
		in := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: app\n  namespace: ops\ndata:\n  token: " +
			base64.StdEncoding.EncodeToString([]byte(redactFixtureNew)) + "\nstringData:\n  token: " + SecretRedactedPlaceholder + "\n"
		out, err := RestoreRedactedSecretYAML(in, existing)
		if err != nil {
			t.Fatal(err)
		}
		sec := parseSecretYAML(t, out)
		if _, ok := sec.StringData["token"]; ok {
			t.Fatal("占位符键应从 stringData 中移除，否则 K8s 合并时会覆盖 data")
		}
		if string(sec.Data["token"]) != redactFixtureNew {
			t.Fatal("data 中用户填写的新值不应被原值覆盖")
		}
	})

	t.Run("data 与 stringData 均不存在时拒绝", func(t *testing.T) {
		in := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: app\n  namespace: ops\nstringData:\n  missing: " + SecretRedactedPlaceholder + "\n"
		if _, err := RestoreRedactedSecretYAML(in, existing); err == nil {
			t.Fatal("未知键的占位符应报错")
		}
	})
}

// data 与 stringData 存在同名键时：脱敏 YAML 两处都替换为占位符；原样回写后两处各自恢复原值。
func TestRedactAndRestoreSameKeyInDataAndStringData(t *testing.T) {
	existing := &corev1.Secret{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ops"},
		Data:       map[string][]byte{"shared": []byte(redactFixtureData)},
		StringData: map[string]string{"shared": redactFixtureString},
	}
	y, err := SecretToYAML(existing)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{redactFixtureData, redactFixtureString, base64.StdEncoding.EncodeToString([]byte(redactFixtureData))} {
		if strings.Contains(y, v) {
			t.Fatal("脱敏 YAML 泄露了明文")
		}
	}
	if strings.Count(y, "shared: "+SecretRedactedPlaceholder) != 2 {
		t.Fatalf("data 与 stringData 的同名键都应为占位符:\n%s", y)
	}

	out, err := RestoreRedactedSecretYAML(y, existing)
	if err != nil {
		t.Fatal(err)
	}
	sec := parseSecretYAML(t, out)
	if string(sec.Data["shared"]) != redactFixtureData || sec.StringData["shared"] != redactFixtureString {
		t.Fatal("同名键原样回写后 data 与 stringData 应各自恢复原值")
	}

	if got := RedactSecretData(existing.Data); len(got["shared"]) != 0 {
		t.Fatal("data 值应被清空")
	}
	if got := RedactSecretStringData(existing.StringData); got["shared"] != "" {
		t.Fatal("stringData 值应被清空")
	}
	if k := SecretDataKeyInfos(existing.Data); len(k) != 1 || k[0].Size != len(redactFixtureData) {
		t.Fatalf("data_keys 不符: %+v", k)
	}
	if k := SecretStringDataKeyInfos(existing.StringData); len(k) != 1 || k[0].Size != len(redactFixtureString) {
		t.Fatalf("string_data_keys 不符: %+v", k)
	}
}

// secretYAMLDocument 必须与 corev1.Secret 的字段一一对应（名称、JSON 标签、类型；仅 data 的值类型不同），
// 否则 k8s 升级新增字段时会被静默丢弃。
func TestSecretYAMLDocumentMatchesCoreV1Secret(t *testing.T) {
	want := reflect.TypeOf(corev1.Secret{})
	got := reflect.TypeOf(secretYAMLDocument{})
	if want.NumField() != got.NumField() {
		t.Fatalf("字段数不一致: corev1.Secret=%d secretYAMLDocument=%d", want.NumField(), got.NumField())
	}
	for i := 0; i < want.NumField(); i++ {
		w, g := want.Field(i), got.Field(i)
		if w.Name != g.Name || w.Anonymous != g.Anonymous || w.Tag.Get("json") != g.Tag.Get("json") {
			t.Fatalf("第 %d 个字段不一致: %s %q vs %s %q", i, w.Name, w.Tag.Get("json"), g.Name, g.Tag.Get("json"))
		}
		if w.Name == "Data" {
			continue
		}
		if w.Type != g.Type {
			t.Fatalf("字段 %s 类型不一致: %s vs %s", w.Name, w.Type, g.Type)
		}
	}
}

// 对合法的单文档 YAML，类型化解析结果必须与直接 yaml.Unmarshal 到 corev1.Secret 完全一致。
func TestYAMLToSecretMatchesDirectUnmarshal(t *testing.T) {
	newB64 := base64.StdEncoding.EncodeToString([]byte(redactFixtureNew))
	inputs := []string{
		"apiVersion: v1\nkind: Secret\nmetadata:\n  name: app\n  namespace: ops\n  labels:\n    port: 8080\n    on: yes\n  annotations:\n    a: \"1\"\ntype: Opaque\nimmutable: true\ndata:\n  k: " + newB64 + "\n  n: null\n  e: \"\"\nstringData:\n  s: " + redactFixtureString + "\n  i: 12\n",
		"apiVersion: v1\nkind: Secret\nmetadata:\n  name: app\nData:\n  k: |\n    " + newB64[:8] + "\n    " + newB64[8:] + "\nStringData:\n  s: x\n",
		"---\nkind: Secret\nmetadata: {name: app, creationTimestamp: null}\ndata: {}\n",
		"kind: Secret\n",
	}
	for i, y := range inputs {
		var direct corev1.Secret
		if err := yaml.Unmarshal([]byte(y), &direct); err != nil {
			t.Fatalf("输入 %d 直接解析失败", i)
		}
		typed, err := YAMLToSecret(y)
		if err != nil {
			t.Fatalf("输入 %d 类型化解析失败: %v", i, err)
		}
		if !reflect.DeepEqual(&direct, typed) {
			t.Fatalf("输入 %d 解析结果不一致", i)
		}
	}
}

// 恢复后重新生成的 YAML 必须能被 YAMLToSecret 解析，且不含占位符。
func TestRestoreRedactedSecretYAMLOutputReparses(t *testing.T) {
	existing := &corev1.Secret{Data: map[string][]byte{"k": []byte(redactFixtureData)}}
	in := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: app\nDATA:\n  k: " + SecretRedactedPlaceholder + "\n"
	out, err := RestoreRedactedSecretYAML(in, existing)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, SecretRedactedPlaceholder) {
		t.Fatal("恢复结果仍含占位符")
	}
	sec, err := YAMLToSecret(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(sec.Data["k"]) != redactFixtureData || sec.Name != "app" {
		t.Fatal("恢复结果不对")
	}
}

func TestSanitizeSecretYAMLError(t *testing.T) {
	err := sanitizeSecretYAMLError(errString("line 3: cannot decode !!str `" + redactFixtureNew + "` as a !!int; unsupported map key of type: <nil>, key: <nil>, value: \"" + redactFixtureNew + "\""))
	if strings.Contains(err.Error(), redactFixtureNew) {
		t.Fatal("未抹掉回显的值")
	}
	if !strings.Contains(err.Error(), "line 3") {
		t.Fatal("应保留位置信息")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
