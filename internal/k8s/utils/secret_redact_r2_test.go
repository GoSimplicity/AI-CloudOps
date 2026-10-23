package utils

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

// SEC-26 P2 第一层 Review 跟进（P2-1、P3-1、P3-2、P3-4、P3-6）。
// 只调用 f763601 上已存在的函数，可在修复前的代码上编译运行（先红后绿）。

// 哨兵前缀：放在值的位置，报错里出现它即视为回显。
const r2Sentinel = "ZQXSNTL"

func r2AssertNoSentinel(t *testing.T, err error) {
	t.Helper()
	if err != nil && strings.Contains(err.Error(), r2Sentinel) {
		t.Fatal("错误信息回显了输入中的值")
	}
}

// 两条路径都必须报错，且报错不含哨兵。
func r2MustErrorNoEcho(t *testing.T, y string) {
	t.Helper()
	_, err := p2Update(y, p2Existing())
	if err == nil {
		t.Fatal("更新路径应报错")
	}
	r2AssertNoSentinel(t, err)
	_, err = p2Create(y)
	if err == nil {
		t.Fatal("新建路径应报错")
	}
	r2AssertNoSentinel(t, err)
}

// P2-1：invalid map key（复杂键）报错不得回显值。
func TestR2InvalidMapKeyDoesNotEcho(t *testing.T) {
	cases := map[string]string{
		"flow_double_brace_stringData": "stringData: {{password: " + r2Sentinel + "01}}\n",
		"flow_double_brace_data":       "data: {{password: " + r2Sentinel + "02}}\n",
		"seq_key_stringData":           "stringData:\n  ? [" + r2Sentinel + "03]\n  : x\n",
		"map_key_stringData":           "stringData:\n  ? {a: " + r2Sentinel + "04}\n  : x\n",
		"seq_key_value_side":           "stringData:\n  ? [a]\n  : " + r2Sentinel + "05\n",
		"top_level_brace":              "{{password: " + r2Sentinel + "06}}\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) { r2MustErrorNoEcho(t, p2Head+body) })
	}
}

// P2-1：直接构造 yaml/json 库可能给出的报错形态，确认过滤器不放行值。
func TestR2SanitizeKnownLibraryMessages(t *testing.T) {
	msgs := []string{
		`yaml: invalid map key: map[interface {}]interface {}{"password":"` + r2Sentinel + `"}`,
		`yaml: line 7: invalid map key: []interface {}{"` + r2Sentinel + `"}`,
		`unsupported map key of type: <nil>, key: <nil>, value: "` + r2Sentinel + `"`,
		"yaml: line 3: cannot decode !!str `" + r2Sentinel + "` as a !!int",
		`parsing time "` + r2Sentinel + `" as "2006-01-02T15:04:05Z07:00": cannot parse`,
		"yaml: unknown anchor '" + r2Sentinel + "' referenced",
		`error converting YAML to JSON: yaml: line 4: key "` + r2Sentinel + `" already set in map`,
		`json: cannot unmarshal string into Go struct field secretYAMLDocument.metadata.name of type ` + r2Sentinel,
		"yaml: line 2: found character that cannot start any token " + r2Sentinel,
	}
	for i, m := range msgs {
		err := sanitizeSecretYAMLError(errors.New(m))
		if strings.Contains(err.Error(), r2Sentinel) {
			t.Fatalf("第 %d 条报错未过滤", i)
		}
	}
	// 行号要保留
	if err := sanitizeSecretYAMLError(errors.New(`yaml: line 7: invalid map key: []interface {}{"x"}`)); !strings.Contains(err.Error(), "7") {
		t.Fatal("应保留行号")
	}
}

// P2-1：表驱动覆盖各类 yaml/json 报错，值的位置放哨兵。
func TestR2YAMLErrorClassesDoNotEcho(t *testing.T) {
	s := r2Sentinel
	cases := map[string]string{
		"unclosed_dquote":      p2Head + "stringData:\n  k: \"" + s + "a\n",
		"unclosed_squote":      p2Head + "stringData:\n  k: '" + s + "b\n",
		"unclosed_flow":        p2Head + "stringData: {k: " + s + "c\n",
		"bad_escape":           p2Head + "stringData:\n  k: \"\\q" + s + "d\"\n",
		"tab_indent":           p2Head + "stringData:\n\tk: " + s + "e\n",
		"mapping_in_value":     p2Head + "stringData:\n  k: " + s + "f: g\n",
		"tag_int":              p2Head + "stringData:\n  k: !!int " + s + "g\n",
		"tag_float":            p2Head + "stringData:\n  k: !!float " + s + "h\n",
		"tag_bool":             p2Head + "stringData:\n  k: !!bool " + s + "i\n",
		"tag_binary":           p2Head + "stringData:\n  k: !!binary " + s + "j!!\n",
		"tag_timestamp":        p2Head + "stringData:\n  k: !!timestamp " + s + "k\n",
		"unknown_anchor":       p2Head + "stringData:\n  k: *" + s + "l\n",
		"creation_timestamp":   "apiVersion: v1\nkind: Secret\nmetadata:\n  name: app\n  creationTimestamp: " + s + "m\n",
		"data_is_scalar":       p2Head + "data: " + s + "n\n",
		"stringData_is_list":   p2Head + "stringData: [" + s + "o]\n",
		"stringData_val_map":   p2Head + "stringData:\n  k: {a: " + s + "p}\n",
		"immutable_string":     p2Head + "immutable: " + s + "q\n",
		"name_is_list":         "apiVersion: v1\nkind: Secret\nmetadata:\n  name: [" + s + "r]\n",
		"labels_val_list":      p2Head + "  labels:\n    a: [" + s + "s]\n",
		"top_level_list":       "- " + s + "t\n",
		"top_level_scalar":     s + "u\n",
		"bad_base64":           p2Head + "data:\n  k: " + s + "v!\n",
		"null_key":             p2Head + "stringData:\n  ~: " + s + "w\n",
		"dup_key_same_value":   p2Head + "stringData:\n  k: " + s + "x\n  k: " + s + "x\n",
		"second_doc_broken":    p2Head + "stringData:\n  k: a\n---\nk: \"" + s + "y\n",
		"control_char":         p2Head + "stringData:\n  k: " + s + "z\x01\n",
		"merge_non_map":        p2Head + "stringData:\n  <<: " + s + "A\n",
		"directive_bad":        "%YAML 9.9 " + s + "B\n" + p2Head,
		"anchor_then_override": p2Head + "x: &a {k: " + s + "C}\nstringData:\n  <<: *a\n  k: " + s + "D\n",
	}
	for name, y := range cases {
		t.Run(name, func(t *testing.T) { r2MustErrorNoEcho(t, y) })
	}
}

// P2-1：模糊测试，把任意片段放到多个值位置，报错不得出现哨兵。
func FuzzSecretYAMLErrorsDoNotEcho(f *testing.F) {
	for _, seed := range []string{"x", "}}", "]", "\"", "'", "\t", ": y", "\n  - z", "!!int", "*a", "{", "\\", "\x00", "{{a: b}}"} {
		f.Add(seed)
	}
	templates := []string{
		p2Head + "stringData:\n  k: %s\n",
		p2Head + "stringData: {{password: %s}}\n",
		p2Head + "stringData:\n  ? [%s]\n  : x\n",
		p2Head + "data:\n  k: %s\n",
		p2Head + "stringData:\n  k: \"%s\n",
		p2Head + "stringData:\n  k: !!int %s\n",
		"apiVersion: v1\nkind: Secret\nmetadata:\n  name: app\n  creationTimestamp: %s\n",
	}
	f.Fuzz(func(t *testing.T, frag string) {
		for _, tpl := range templates {
			y := strings.Replace(tpl, "%s", r2Sentinel+frag, 1)
			if _, err := p2Update(y, p2Existing()); err != nil {
				r2AssertNoSentinel(t, err)
			}
			if _, err := p2Create(y); err != nil {
				r2AssertNoSentinel(t, err)
			}
		}
	})
}

// P3-1：首尾的 NUL 与零宽字符（ZWSP/ZWNJ/ZWJ/BOM）也要去掉后再判断占位符。
func TestR2PlaceholderWithInvisibleChars(t *testing.T) {
	variants := map[string]string{
		"nul_suffix":  SecretRedactedPlaceholder + "\x00",
		"nul_prefix":  "\x00" + SecretRedactedPlaceholder,
		"zwsp":        "\u200b" + SecretRedactedPlaceholder,
		"zwnj":        SecretRedactedPlaceholder + "\u200c",
		"zwj":         "\u200d" + SecretRedactedPlaceholder + "\u200d",
		"bom":         "\ufeff" + SecretRedactedPlaceholder,
		"mixed":       "\ufeff \u200b" + SecretRedactedPlaceholder + "\x00\n\u200c",
		"nul_nul_ws":  "\x00\x00 " + SecretRedactedPlaceholder + " \x00",
		"zwsp_suffix": SecretRedactedPlaceholder + "\u200b",
	}
	for name, v := range variants {
		t.Run("data_"+name, func(t *testing.T) {
			p2MustReject(t, p2Head+"data:\n  k: "+base64.StdEncoding.EncodeToString([]byte(v))+"\n")
		})
		t.Run("stringData_"+name, func(t *testing.T) {
			p2MustReject(t, p2Head+"stringData:\n  k: \""+r2YAMLEscape(v)+"\"\n")
		})
	}
	// 对照：零宽字符在中间时不是占位符，按普通值接受
	t.Run("interior_not_placeholder", func(t *testing.T) {
		y := p2Head + "stringData:\n  k: \"<re\\u200bdacted>\"\n"
		if _, err := p2Create(y); err != nil {
			t.Fatal(err)
		}
	})
}

// 把字符串转成 YAML 双引号字面量（不可见字符用转义写出）
func r2YAMLEscape(v string) string {
	var b strings.Builder
	for _, r := range v {
		switch {
		case r == 0:
			b.WriteString(`\0`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '"' || r == '\\':
			b.WriteRune('\\')
			b.WriteRune(r)
		case r > 0x7e:
			b.WriteString(`\u` + hex4(r))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func hex4(r rune) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[(r>>12)&0xf], digits[(r>>8)&0xf], digits[(r>>4)&0xf], digits[r&0xf]})
}

// P3-2：字符串化后相同的键（1 与 "1"、true 与 "true"、1.5 与 "1.5"）按重复键拒绝，多次运行结果稳定。
func TestR2StringifiedDuplicateKeysRejectedStably(t *testing.T) {
	newB64 := base64.StdEncoding.EncodeToString([]byte(redactFixtureNew))
	cases := map[string]string{
		"stringData_int":   p2Head + "stringData:\n  1: \"" + SecretRedactedPlaceholder + "\"\n  \"1\": " + redactFixtureNew + "\n",
		"data_int":         p2Head + "data:\n  1: \"" + SecretRedactedPlaceholder + "\"\n  \"1\": " + newB64 + "\n",
		"stringData_bool":  p2Head + "stringData:\n  true: \"" + SecretRedactedPlaceholder + "\"\n  \"true\": " + redactFixtureNew + "\n",
		"stringData_yes":   p2Head + "stringData:\n  yes: \"" + SecretRedactedPlaceholder + "\"\n  \"true\": " + redactFixtureNew + "\n",
		"stringData_float": p2Head + "stringData:\n  1.5: \"" + SecretRedactedPlaceholder + "\"\n  \"1.5\": " + redactFixtureNew + "\n",
		"labels_int":       p2Head + "  labels:\n    1: a\n    \"1\": b\n",
		"nested_in_list":   p2Head + "extra:\n  - 1: a\n    \"1\": b\n",
	}
	for name, y := range cases {
		t.Run(name, func(t *testing.T) {
			for i := 0; i < 50; i++ {
				if _, err := RestoreRedactedSecretYAML(y, p2Existing()); err == nil {
					t.Fatalf("第 %d 次：RestoreRedactedSecretYAML 应拒绝", i)
				}
				if _, err := SecretYAMLHasRedactedPlaceholder(y); err == nil {
					t.Fatalf("第 %d 次：SecretYAMLHasRedactedPlaceholder 应报错", i)
				}
				if _, err := YAMLToSecret(y); err == nil {
					t.Fatalf("第 %d 次：YAMLToSecret 应拒绝", i)
				}
			}
		})
	}
	// 对照：类型不同但字符串化后不同的键照常接受
	t.Run("distinct_ok", func(t *testing.T) {
		y := p2Head + "stringData:\n  1: a\n  \"2\": b\n"
		sec, err := YAMLToSecret(y)
		if err != nil || sec.StringData["1"] != "a" || sec.StringData["2"] != "b" {
			t.Fatal("不同键应照常接受")
		}
	})
}

// P3-4：metadata 字段名本身为非标准大小写时，其下只差大小写的键也要拒绝。
func TestR2NonCanonicalMetadataFieldDuplicates(t *testing.T) {
	for _, field := range []string{"Metadata", "METADATA", "metaData"} {
		t.Run(field, func(t *testing.T) {
			y := "apiVersion: v1\nkind: Secret\n" + field + ":\n  name: app\n  Name: other\n  namespace: ops\nstringData:\n  k: " + redactFixtureNew + "\n"
			p2MustReject(t, y)
			if _, err := YAMLToSecret(y); err == nil {
				t.Fatal("YAMLToSecret 应拒绝")
			}
		})
	}
	// 对照：只有非标准写法、没有重复时照常接受（按 encoding/json 不区分大小写的规则归并）
	t.Run("Metadata_alone_ok", func(t *testing.T) {
		sec, err := YAMLToSecret("apiVersion: v1\nkind: Secret\nMetadata:\n  name: app\n")
		if err != nil || sec.Name != "app" {
			t.Fatal("单独的 Metadata 应照常解析")
		}
	})
}

// P3-6：入口大小上限 1 MiB（1048576 字节），超限在任何解析前报通用错误。
func TestR2SizeLimit(t *testing.T) {
	const limit = 1048576
	build := func(total int) string {
		head := p2Head + "stringData:\n  k: "
		pad := total - len(head) - 1
		return head + strings.Repeat("a", pad) + "\n"
	}
	at := build(limit)
	over := build(limit + 1)
	if len(at) != limit || len(over) != limit+1 {
		t.Fatal("构造长度不对")
	}
	t.Run("exactly_limit_accepted", func(t *testing.T) {
		if _, err := p2Create(at); err != nil {
			t.Fatalf("新建路径: %v", err)
		}
		if _, err := p2Update(at, p2Existing()); err != nil {
			t.Fatalf("更新路径: %v", err)
		}
	})
	t.Run("limit_plus_one_rejected", func(t *testing.T) {
		if _, err := p2Create(over); err == nil {
			t.Fatal("新建路径应拒绝")
		}
		if _, err := p2Update(over, p2Existing()); err == nil {
			t.Fatal("更新路径应拒绝")
		}
		if _, err := YAMLToSecret(over); err == nil {
			t.Fatal("YAMLToSecret 应拒绝")
		}
	})
	t.Run("checked_before_parse", func(t *testing.T) {
		// 超限且语法错误、含哨兵：应报大小错误（与语法错误文案不同），不回显
		broken := "stringData: {{k: " + r2Sentinel + strings.Repeat("b", limit) + "\n"
		_, errSize := SecretYAMLHasRedactedPlaceholder(broken)
		_, errSyntax := SecretYAMLHasRedactedPlaceholder("stringData: {{k: " + r2Sentinel + "\n")
		if errSize == nil || errSyntax == nil {
			t.Fatal("两者都应报错")
		}
		r2AssertNoSentinel(t, errSize)
		if errSize.Error() == errSyntax.Error() {
			t.Fatal("超限输入应在解析前以大小错误拒绝")
		}
		if !strings.Contains(errSize.Error(), "1 MiB") {
			t.Fatal("应提示 1 MiB 上限")
		}
	})
}
