/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package config

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/pkg/runenv"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NonFatalError 用于标记“可继续运行”的告警类错误（例如配置文件不存在）。
type NonFatalError struct {
	err error
}

func (e NonFatalError) Error() string {
	return e.err.Error()
}

func (e NonFatalError) Unwrap() error {
	return e.err
}

func IsNonFatal(err error) bool {
	var nf NonFatalError
	return errors.As(err, &nf)
}

// Load 加载主程序配置，支持优先级：环境变量 > 配置文件 > 默认值。
func Load() (*Config, *ExternalConfig, error) {
	configFile := pflag.String("config", "", "配置文件路径")
	pflag.Parse()

	if *configFile == "" {
		// 与 Config.Validate 使用同一判断（去首尾空白、忽略大小写），避免选了开发配置却按生产规则校验
		if runenv.IsProduction() {
			*configFile = "config/config.production.yaml"
		} else {
			*configFile = "config/config.development.yaml"
		}
	}

	return loadFromFile(*configFile)
}

// loadFromFile 按“环境变量 > 配置文件 > 默认值”加载指定配置文件（不解析命令行参数，便于测试）
func loadFromFile(configFile string) (*Config, *ExternalConfig, error) {
	viper.SetConfigFile(configFile)
	setDefaults()
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	readErr := viper.ReadInConfig()
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return nil, nil, fmt.Errorf("读取配置文件失败: %w", readErr)
	}

	bindEnvVars()

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, nil, fmt.Errorf("解析配置失败: %w", err)
	}

	ext := &ExternalConfig{}
	loadExternalConfig(ext)

	if err := cfg.Validate(); err != nil {
		return cfg, ext, fmt.Errorf("配置校验失败: %w", err)
	}

	if readErr != nil {
		return cfg, ext, NonFatalError{err: fmt.Errorf("读取配置文件失败: %w", readErr)}
	}

	return cfg, ext, nil
}

// WebhookSecrets Webhook 进程的敏感配置，仅保存在返回值中，不写入全局 viper
type WebhookSecrets struct {
	// MySQLAddr 来自 WEBHOOK_MYSQL_ADDR 的 MySQL DSN（含账号密码）
	MySQLAddr string
}

// LoadWebhook 加载 Webhook 子系统配置，支持优先级：环境变量 > 配置文件 > 默认值。
func LoadWebhook() (*WebhookConfig, *WebhookSecrets, error) {
	configFile := pflag.String("config", "config/webhook.yaml", "配置文件路径")
	pflag.Parse()

	return loadWebhookFromFile(*configFile)
}

// WebhookMySQLAddrEnv Webhook 进程的 MySQL DSN（含账号密码）环境变量，必需，无内置默认值
const WebhookMySQLAddrEnv = "WEBHOOK_MYSQL_ADDR"

// webhookMySQLAddrKey webhook.yaml 中仅保留为空占位的 DSN 键，其值从不被读取
const webhookMySQLAddrKey = "mysql.addr"

// webhookGlobalPrefix 发布到全局 viper 的键前缀：webhook 子系统其余代码仍通过全局 viper 读取 webhook.*
const webhookGlobalPrefix = "webhook."

// loadWebhookFromFile 加载指定的 Webhook 配置文件（不解析命令行参数，便于测试）。
// 使用独立的 viper 实例读取文件与环境变量：DSN 只经由返回的 WebhookSecrets 传给数据库初始化，
// 不写入全局 viper（mysql.addr 不会出现在全局配置中），也不受主程序 MYSQL_ADDR 影响
func loadWebhookFromFile(configFile string) (*WebhookConfig, *WebhookSecrets, error) {
	v := viper.New()
	v.SetConfigFile(configFile)
	setWebhookDefaults(v)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	readErr := v.ReadInConfig()
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return nil, nil, fmt.Errorf("读取Webhook配置文件失败: %w", readErr)
	}

	// 不再支持没有 webhook 根节点的旧平铺格式：这些键只会被忽略、静默回落到默认值（端口可能变化），
	// 所以直接失败并提示移到 webhook: 下
	if misplaced := misplacedWebhookRootKeys(v); len(misplaced) > 0 {
		// 只列出键名，不输出值
		return nil, nil, fmt.Errorf("Webhook 配置文件根节点下出现了 webhook 字段 %s：不再支持没有 webhook 根节点的旧格式，请把这些键移到 webhook: 下",
			strings.Join(misplaced, ", "))
	}

	// 配置文件存在却没有 webhook 根节点（含写成 `webhook:` 而值为 null）：webhook.* 全部回落到环境变量/默认值，容易是写错了文件，记告警继续启动。
	// 只列出文件根节点下的键名，不输出值
	if readErr == nil && !v.InConfig("webhook") {
		roots := configFileRootKeys(v)
		listed := "无"
		if len(roots) > 0 {
			listed = strings.Join(roots, ", ")
		}
		configWarnf("[配置告警] Webhook 配置文件 %s 中没有 webhook 根节点（或其值为空），webhook.* 将全部取环境变量/默认值；文件根节点下的键：%s", configFile, listed)
	}

	bindStructEnvVars(v, reflect.TypeOf(WebhookConfig{}), "webhook")

	// SEC-02A/SEC-22：DSN 整体由 WEBHOOK_MYSQL_ADDR 注入，配置文件不写、无默认值，缺失即失败
	addr := strings.TrimSpace(os.Getenv(WebhookMySQLAddrEnv))
	if addr == "" {
		// 只列出变量名，不输出任何值
		return nil, nil, fmt.Errorf("缺少必需的密钥环境变量: %s（不提供内置默认值，请通过环境变量注入）", WebhookMySQLAddrEnv)
	}

	// 配置文件结构与原始文本校验（只看文件本身）：叶子写成 map/列表、父键写成标量/列表、整数字段不是十进制整数都在这里拒绝，
	// 同时收集 webhook 下的未知键。文件里写错就失败，与该键是否被环境变量覆盖无关
	var scan webhookFileScan
	if readErr == nil {
		var err error
		if scan, err = scanWebhookConfigFile(v.ConfigFileUsed()); err != nil {
			return nil, nil, err
		}
	}

	// 逐键取合并后的值（环境变量 > 配置文件 > 默认值）。v.UnmarshalKey("webhook") 只拿到配置文件里的
	// webhook 子树，环境变量和默认值进不了 cfg（WEBHOOK_PORT 因此从未生效），所以不能直接用。
	// 只构造这一次：解码 cfg 与发布到全局 viper 共用同一份、同一顺序
	merged, err := collectWebhookLeafValues(v, scan)
	if err != nil {
		return nil, nil, err
	}
	rawPort, hasPort := merged.vals[webhookPortKey]
	port := strings.TrimSpace(fmt.Sprint(rawPort))
	if !hasPort || rawPort == nil || !isValidPort(port) {
		// 只列出键名和变量名，不输出值
		return nil, nil, fmt.Errorf("Webhook 端口 %s（环境变量 WEBHOOK_PORT）必须是 1-65535 的纯数字", webhookPortKey)
	}
	// 去掉前导零后的规范值（如 000080 → 80）同时用于监听和全局 webhook.port
	merged.vals[webhookPortKey] = normalizePort(port)

	cfg, err := decodeWebhookConfig(merged)
	if err != nil {
		return nil, nil, err
	}
	// 整数字段范围校验：失败时与其他校验一样直接返回，不发布到全局
	if err := validateWebhookIntFields(cfg); err != nil {
		return nil, nil, err
	}

	// 未知键不发布、不拒绝启动，每次加载只告警一次，只列键名
	if len(scan.unknown) > 0 {
		configWarnf("[配置告警] Webhook 配置文件 %s 中 webhook 下有未识别的键（不会生效，也不发布到全局）：%s", configFile, strings.Join(scan.unknown, ", "))
	}

	// 只把 webhook.* 的已知叶子键发布到全局 viper，供 webhook 子系统现有读取方使用；mysql.* 与未知键不发布。
	// 与 cfg 使用同一份合并后的值
	merged.apply(viper.Set)

	secrets := &WebhookSecrets{MySQLAddr: addr}
	if readErr != nil {
		return cfg, secrets, NonFatalError{err: fmt.Errorf("读取Webhook配置文件失败: %w", readErr)}
	}
	return cfg, secrets, nil
}

// webhookPortKey Webhook 监听端口的配置键
const webhookPortKey = "webhook.port"

// misplacedWebhookRootKeys 返回配置文件根节点下出现的 WebhookConfig 字段键名（如 port、fixed_workers），
// 按结构体字段顺序。只看配置文件本身（InConfig），不受环境变量（如 PORT）和默认值影响
func misplacedWebhookRootKeys(v *viper.Viper) []string {
	var keys []string
	t := reflect.TypeOf(WebhookConfig{})
	for i := 0; i < t.NumField(); i++ {
		key := t.Field(i).Tag.Get("mapstructure")
		if key != "" && v.InConfig(key) {
			keys = append(keys, key)
		}
	}
	return keys
}

// webhookLeaf WebhookConfig 的一个叶子字段
type webhookLeaf struct {
	key   string       // 完整键名，如 webhook.im_feishu.request_timeout_seconds
	env   string       // 环境变量名（env 标签）
	kind  reflect.Kind // reflect.String 或 reflect.Int
	index []int        // 在 WebhookConfig 中的字段路径（FieldByIndex）
}

// webhookSchema 由反射 WebhookConfig 生成的两张表：
//   - parents：父键白名单（结构体字段，目前只有 webhook.im_feishu），只有它们可以是 map/null 并被跳过；
//   - leaves：叶子类型表（string/int 字段），合并值、解码和发布只认这张表里的键。
//
// 根节点 webhook 单独处理，不在 parents 中
type webhookSchema struct {
	parents  map[string]bool
	leaves   map[string]webhookLeaf
	leafKeys []string // 已排序
}

var webhookKeySchema = buildWebhookSchema(reflect.TypeOf(WebhookConfig{}), "webhook")

func buildWebhookSchema(t reflect.Type, root string) webhookSchema {
	s := webhookSchema{parents: map[string]bool{}, leaves: map[string]webhookLeaf{}}
	var walk func(t reflect.Type, prefix string, index []int)
	walk = func(t reflect.Type, prefix string, index []int) {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := f.Tag.Get("mapstructure")
			if tag == "" {
				continue
			}
			key := prefix + "." + tag
			idx := append(append([]int(nil), index...), i)
			switch f.Type.Kind() {
			case reflect.Struct:
				s.parents[key] = true
				walk(f.Type, key, idx)
			case reflect.String, reflect.Int:
				env := f.Tag.Get("env")
				if env == "" {
					env = strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
				}
				s.leaves[key] = webhookLeaf{key: key, env: env, kind: f.Type.Kind(), index: idx}
				s.leafKeys = append(s.leafKeys, key)
			default:
				// 新增其他类型的字段时必须同步扩展合并/解码逻辑
				panic(fmt.Sprintf("webhook 配置字段 %s 的类型 %s 未支持", key, f.Type.Kind()))
			}
		}
	}
	walk(t, root, nil)
	sort.Strings(s.leafKeys)
	return s
}

// webhookDecimalInt 整数字段唯一接受的写法：十进制整数，可带负号，不允许前导零、+ 号、空白、小数点、下划线和进制前缀
var webhookDecimalInt = regexp.MustCompile(`^(0|-?[1-9][0-9]*)$`)

// webhookTypeErrorText 某个叶子/父键类型不对时的说明（只含键名和变量名，不含取值）
func webhookTypeErrorText(key string) string {
	if leaf, ok := webhookKeySchema.leaves[key]; ok {
		if leaf.kind == reflect.Int {
			return fmt.Sprintf("%s（环境变量 %s）应为十进制整数", key, leaf.env)
		}
		return fmt.Sprintf("%s 应为字符串", key)
	}
	return fmt.Sprintf("%s 应为 map 或留空", key)
}

// webhookTypeError 汇总类型错误：按键名排序，只列键名
func webhookTypeError(bad map[string]bool) error {
	if len(bad) == 0 {
		return nil
	}
	keys := make([]string, 0, len(bad))
	for k := range bad {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	items := make([]string, len(keys))
	for i, k := range keys {
		items[i] = webhookTypeErrorText(k)
	}
	return fmt.Errorf("Webhook 配置类型错误: %s", strings.Join(items, "；"))
}

// webhookFileScan 配置文件本身的扫描结果
type webhookFileScan struct {
	bad     map[string]bool   // 类型错误的键
	unknown []string          // webhook 下不认识的键（已排序去重，不展开子键）
	raw     map[string]string // 叶子在文件里的原始标量文本（非 null）；大小写不同的重复键不记录
}

// scanWebhookConfigFile 用 yaml.v3 读取配置文件节点树，按两张表检查 webhook 子树。
// 只处理 YAML/JSON（yaml.v3 可解析 JSON）；其他格式返回空结果，仍由 collectWebhookLeafValues 按取值类型兜底
func scanWebhookConfigFile(path string) (webhookFileScan, error) {
	scan := webhookFileScan{bad: map[string]bool{}, raw: map[string]string{}}
	switch strings.ToLower(strings.TrimPrefix(filepath.Ext(path), ".")) {
	case "yaml", "yml", "json":
	default:
		return scan, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return scan, fmt.Errorf("读取Webhook配置文件失败: %w", err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		// viper 已成功解析过同一文件，这里只在文件被并发改动时出现；不回显内容
		return scan, errors.New("读取Webhook配置文件失败: 无法解析 YAML 节点")
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return scan, nil
	}
	root := resolveYAMLAlias(doc.Content[0])
	if root.Kind != yaml.MappingNode {
		return scan, nil
	}
	seen := map[string]int{}
	unknown := map[string]bool{}
	forEachYAMLPair(root, func(k, val *yaml.Node) {
		if strings.ToLower(k.Value) != "webhook" {
			return
		}
		switch {
		case isYAMLNull(val):
			// 根节点为空：走“没有 webhook 根节点”的告警分支
		case val.Kind == yaml.MappingNode:
			scanWebhookMapping(val, "webhook", &scan, seen, unknown)
		default:
			scan.bad["webhook"] = true
		}
	})
	for k, n := range seen {
		if n > 1 {
			delete(scan.raw, k)
		}
	}
	for k := range unknown {
		scan.unknown = append(scan.unknown, k)
	}
	sort.Strings(scan.unknown)
	return scan, nil
}

// scanWebhookMapping 检查 prefix 对应 mapping 下的每个键：
//   - 白名单父键：只能是 map 或 null，map 时递归；
//   - 叶子：只能是标量（null 等同未写）；整数叶子的原始文本必须是十进制整数；
//   - 其他：记为未知键，不展开子键
func scanWebhookMapping(m *yaml.Node, prefix string, scan *webhookFileScan, seen map[string]int, unknown map[string]bool) {
	forEachYAMLPair(m, func(k, val *yaml.Node) {
		key := prefix + "." + strings.ToLower(k.Value)
		if webhookKeySchema.parents[key] {
			switch {
			case isYAMLNull(val):
			case val.Kind == yaml.MappingNode:
				scanWebhookMapping(val, key, scan, seen, unknown)
			default:
				scan.bad[key] = true
			}
			return
		}
		leaf, ok := webhookKeySchema.leaves[key]
		if !ok {
			unknown[key] = true
			return
		}
		if val.Kind != yaml.ScalarNode {
			scan.bad[key] = true
			return
		}
		if isYAMLNull(val) {
			return
		}
		if leaf.kind == reflect.Int && !webhookDecimalInt.MatchString(val.Value) {
			scan.bad[key] = true
			return
		}
		seen[key]++
		scan.raw[key] = val.Value
	})
}

// forEachYAMLPair 遍历 mapping 的键值对（解析别名；<< 合并键按 YAML 语义展开到同一层）
func forEachYAMLPair(m *yaml.Node, fn func(k, val *yaml.Node)) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		k, val := m.Content[i], resolveYAMLAlias(m.Content[i+1])
		if k.Tag == "!!merge" || k.Value == "<<" {
			switch val.Kind {
			case yaml.MappingNode:
				forEachYAMLPair(val, fn)
			case yaml.SequenceNode:
				for _, item := range val.Content {
					if item = resolveYAMLAlias(item); item.Kind == yaml.MappingNode {
						forEachYAMLPair(item, fn)
					}
				}
			}
			continue
		}
		fn(k, val)
	}
}

func resolveYAMLAlias(n *yaml.Node) *yaml.Node {
	for n != nil && n.Kind == yaml.AliasNode && n.Alias != nil {
		n = n.Alias
	}
	return n
}

func isYAMLNull(n *yaml.Node) bool {
	return n == nil || (n.Kind == yaml.ScalarNode && n.ShortTag() == "!!null")
}

// webhookValues 逐键合并后的 webhook.* 叶子键取值；keys 已排序，写入顺序确定
type webhookValues struct {
	keys []string
	vals map[string]any
}

// apply 按 keys 的固定顺序逐个写入（发布到全局 viper 时使用）
func (m webhookValues) apply(set func(key string, val any)) {
	for _, key := range m.keys {
		set(key, m.vals[key])
	}
}

// collectWebhookLeafValues 按叶子类型表取每个叶子键合并后的值（环境变量 > 配置文件 > 默认值），并规范成字段的 Go 类型：
//   - 只遍历叶子表，父键（白名单）与未知键天然不进入合并值；
//   - 整数叶子：字符串（环境变量或加引号的 YAML）必须是十进制整数；取自配置文件的数值按文件原始文本解析
//     （010 不会变成 8，10.5 不会截断成 10）；超出 int 范围的按边界值保留，交给范围校验报错；
//   - 字符串叶子：取自配置文件的非字符串标量（如未加引号的 12345、true）用文件原始文本；
//   - map、列表或其他无法表示的值报类型错误（与 scan 的结果一起汇总）。
//
// 报错只列键名
func collectWebhookLeafValues(v *viper.Viper, scan webhookFileScan) (webhookValues, error) {
	bad := map[string]bool{}
	for k := range scan.bad {
		bad[k] = true
	}
	m := webhookValues{vals: make(map[string]any, len(webhookKeySchema.leafKeys))}
	for _, key := range webhookKeySchema.leafKeys {
		if !v.IsSet(key) {
			continue
		}
		if bad[key] {
			continue
		}
		leaf := webhookKeySchema.leaves[key]
		val := v.Get(key)
		raw, fromFile := scan.raw[key]
		var (
			out any
			ok  bool
		)
		if leaf.kind == reflect.Int {
			out, ok = normalizeWebhookInt(val, raw, fromFile)
		} else {
			out, ok = normalizeWebhookString(val, raw, fromFile)
		}
		if !ok {
			bad[key] = true
			continue
		}
		m.keys = append(m.keys, key)
		m.vals[key] = out
	}
	if err := webhookTypeError(bad); err != nil {
		return webhookValues{}, err
	}
	return m, nil
}

// parseWebhookDecimalInt 严格解析十进制整数；超出 int 范围时返回边界值（ok 仍为 true），由范围校验报错
func parseWebhookDecimalInt(s string) (int, bool) {
	if !webhookDecimalInt.MatchString(s) {
		return 0, false
	}
	n, err := strconv.ParseInt(s, 10, strconv.IntSize)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0, false
	}
	return int(n), true
}

// clampWebhookInt 把 int64 夹到 int 范围（32 位平台），越界值交给范围校验报错
func clampWebhookInt(n int64) int {
	if n > math.MaxInt {
		return math.MaxInt
	}
	if n < math.MinInt {
		return math.MinInt
	}
	return int(n)
}

func normalizeWebhookInt(val any, raw string, fromFile bool) (any, bool) {
	switch x := val.(type) {
	case string:
		return parseWebhookDecimalInt(x)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		if fromFile {
			// 数值来自配置文件：按原始文本解析，避免 YAML 的进制/浮点转换
			return parseWebhookDecimalInt(raw)
		}
		// 没有原始文本（内置默认值，或 TOML 等非 YAML 格式）：只接受整数类型，浮点不截断
		rv := reflect.ValueOf(x)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return clampWebhookInt(rv.Int()), true
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if u := rv.Uint(); u > math.MaxInt64 {
				return math.MaxInt, true
			} else {
				return clampWebhookInt(int64(u)), true
			}
		}
		return 0, false
	default:
		return 0, false
	}
}

func normalizeWebhookString(val any, raw string, fromFile bool) (any, bool) {
	switch x := val.(type) {
	case string:
		return x, true
	case nil:
		return "", true
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		if fromFile {
			return raw, true
		}
		return fmt.Sprint(x), true
	default:
		// map、列表、时间戳等
		if fromFile {
			if _, isMapOrSlice := webhookCompositeKind(val); !isMapOrSlice {
				return raw, true
			}
		}
		return nil, false
	}
}

// webhookCompositeKind 判断取值是否为 map 或列表
func webhookCompositeKind(val any) (reflect.Kind, bool) {
	if val == nil {
		return reflect.Invalid, false
	}
	k := reflect.TypeOf(val).Kind()
	return k, k == reflect.Map || k == reflect.Slice || k == reflect.Array
}

// configFileRootKeys 返回配置文件本身根节点下的键名（排序去重），不含默认值和环境变量带来的键
func configFileRootKeys(v *viper.Viper) []string {
	seen := map[string]bool{}
	var roots []string
	for _, key := range v.AllKeys() {
		root, _, _ := strings.Cut(key, ".")
		if !seen[root] && v.InConfig(root) {
			seen[root] = true
			roots = append(roots, root)
		}
	}
	sort.Strings(roots)
	return roots
}

// decodeWebhookConfig 把逐键合并后的 webhook.* 值按叶子类型表解码为 WebhookConfig。
// 不走 mapstructure 的 WeaklyTypedInput 宽松转换：值必须已经是字段的 Go 类型（collectWebhookLeafValues 已规范），
// 否则报错。报错只含键名和期望类型，不回显原值
func decodeWebhookConfig(merged webhookValues) (*WebhookConfig, error) {
	cfg := &WebhookConfig{}
	rv := reflect.ValueOf(cfg).Elem()
	var bad []string
	for _, key := range merged.keys {
		leaf, ok := webhookKeySchema.leaves[key]
		if !ok {
			continue
		}
		fv := rv.FieldByIndex(leaf.index)
		switch leaf.kind {
		case reflect.String:
			s, ok := merged.vals[key].(string)
			if !ok {
				bad = append(bad, key+" 期望 string")
				continue
			}
			fv.SetString(s)
		case reflect.Int:
			n, ok := merged.vals[key].(int)
			if !ok {
				bad = append(bad, key+" 期望 int")
				continue
			}
			fv.SetInt(int64(n))
		}
	}
	if len(bad) > 0 {
		return nil, fmt.Errorf("解析Webhook配置失败: %s", strings.Join(bad, "；"))
	}
	return cfg, nil
}

// webhookIntRule webhook 整数字段的取值范围（闭区间）
type webhookIntRule struct {
	key string
	env string
	min int
	max int
	get func(*WebhookConfig) int
}

// webhookIntRules WebhookConfig 全部整数字段的规则。下限依据各字段在 webhook 子系统中的实际用法；
// 上限按语义取宽松但合理的值，同时保证乘以 time.Second / time.Minute 不会溢出：
//   - fixed_workers：工作者数量，0 个工作者没有意义，最小 1；上限 1024（远超单进程合理并发，防止误写成超大值）；
//   - default_upgrade_minutes：发送组 upgrade_minutes 为 0 时（含义为“用默认值”）的升级等待分钟数，
//     content.go 用 time.Since(StartsAt) > N 分钟判断，为 0 时所有 firing 告警立即升级，等待失去意义，最小 1；
//     上限 10080（7 天），告警升级等待超过一周已无实际意义；
//   - common_map_renew_interval_seconds：缓存刷新周期，加载阶段拒绝，缓存层保留防御性回退；
//     最小 1，上限 86400（1 天），再长缓存基本不刷新；
//   - im_feishu.request_timeout_seconds：用作 http.Client.Timeout（秒），Go 中 0 表示不设超时（不限制），
//     不会立即超时也不会除零，允许 0；负数在 http.Client 中同样被当作不设超时，但不是有意义的写法，拒绝；
//     上限 600（10 分钟），单次 HTTP 请求等更久与不设超时无异
var webhookIntRules = []webhookIntRule{
	{"webhook.fixed_workers", "WEBHOOK_FIXED_WORKERS", 1, 1024, func(c *WebhookConfig) int { return c.FixedWorkers }},
	{"webhook.default_upgrade_minutes", "WEBHOOK_DEFAULT_UPGRADE_MINUTES", 1, 10080, func(c *WebhookConfig) int { return c.DefaultUpgradeMinutes }},
	{"webhook.common_map_renew_interval_seconds", "WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS", 1, 86400, func(c *WebhookConfig) int { return c.CommonMapRenewIntervalSeconds }},
	{"webhook.im_feishu.request_timeout_seconds", "WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS", 0, 600, func(c *WebhookConfig) int { return c.ImFeishu.RequestTimeoutSeconds }},
}

// validateWebhookIntFields 按 webhookIntRules 校验整数字段，列出全部不合法的键；报错只含键名、变量名和上下限，不含取值
func validateWebhookIntFields(cfg *WebhookConfig) error {
	var bad []string
	for _, r := range webhookIntRules {
		if n := r.get(cfg); n < r.min || n > r.max {
			bad = append(bad, fmt.Sprintf("%s（环境变量 %s）必须在 %d 到 %d 之间", r.key, r.env, r.min, r.max))
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("Webhook 整数配置超出允许范围: %s", strings.Join(bad, "；"))
	}
	return nil
}

// isValidPort 端口必须是纯 ASCII 数字且在 1-65535 之间（调用方先去首尾空白）。
// 先去掉前导零再查长度和范围：000080 视为 80；全是零（0、0000）仍拒绝
func isValidPort(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	t := strings.TrimLeft(s, "0")
	if t == "" || len(t) > 5 {
		return false
	}
	n := 0
	for _, r := range t {
		n = n*10 + int(r-'0')
	}
	return n >= 1 && n <= 65535
}

// normalizePort 去掉合法端口的前导零（000080 → 80），只在 isValidPort 通过后调用
func normalizePort(s string) string {
	return strings.TrimLeft(s, "0")
}

func setDefaults() {
	viper.SetDefault("server.port", "8889")

	viper.SetDefault("log.dir", "./logs")
	viper.SetDefault("log.level", "debug")

	// jwt.key1/key2、mysql.addr、tree.password_encryption_key 无内置默认值，
	// 必须通过环境变量注入，缺失时 Config.Validate 直接失败（见 requiredSecretEnvVars）。
	// jwt.issuer 不是密钥，取自 config/*.yaml，可用环境变量 JWT_ISSUER 覆盖
	viper.SetDefault("jwt.expiration", 30) // 单位：分钟

	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.password", "")

	viper.SetDefault("mysql.max_open_conns", 20)
	viper.SetDefault("mysql.max_idle_conns", 5)
	viper.SetDefault("mysql.conn_max_lifetime", "30m")
	viper.SetDefault("mysql.conn_max_idle_time", "5m")
	viper.SetDefault("mysql.connect_timeout", "5s")
	viper.SetDefault("mysql.startup_timeout", "10s")

	viper.SetDefault("tree.check_status_cron", "@every 300s")

	viper.SetDefault("k8s.refresh_cron", "@every 300s")

	viper.SetDefault("prometheus.refresh_cron", "@every 15s")
	viper.SetDefault("prometheus.enable_alert", 0)
	viper.SetDefault("prometheus.enable_record", 0)
	viper.SetDefault("prometheus.alert_webhook_addr", "http://localhost:8889/api/v1/alerts/receive")
	viper.SetDefault("prometheus.alert_webhook_file_dir", "/tmp/webhook_files")
	viper.SetDefault("prometheus.httpSdAPI", "http://localhost:8889/api/not_auth/getTreeNodeBindIps")

	viper.SetDefault("mock.enabled", true)

	viper.SetDefault(CronAllowLocalExecKey, false)

	viper.SetDefault("notification.email.enabled", false)
	viper.SetDefault("notification.email.smtp_host", "smtp.gmail.com")
	viper.SetDefault("notification.email.smtp_port", 587)
	viper.SetDefault("notification.email.username", "")
	viper.SetDefault("notification.email.password", "")
	viper.SetDefault("notification.email.from_name", "AI-CloudOps")
	viper.SetDefault("notification.email.max_retries", 3)
	viper.SetDefault("notification.email.retry_interval", "5m")
	viper.SetDefault("notification.email.timeout", "30s")
	viper.SetDefault("notification.email.use_tls", true)

	viper.SetDefault("notification.feishu.enabled", false)
	viper.SetDefault("notification.feishu.app_id", "")
	viper.SetDefault("notification.feishu.app_secret", "")
	viper.SetDefault("notification.feishu.webhook_url", "https://open.feishu.cn/open-apis/bot/v2/hook/")
	viper.SetDefault("notification.feishu.private_message_api", "https://open.feishu.cn/open-apis/im/v1/messages")
	viper.SetDefault("notification.feishu.tenant_access_token_api", "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal")
	viper.SetDefault("notification.feishu.max_retries", 3)
	viper.SetDefault("notification.feishu.retry_interval", "5m")
	viper.SetDefault("notification.feishu.timeout", "10s")
}

func setWebhookDefaults(v *viper.Viper) {
	v.SetDefault("webhook.port", "8889")
	v.SetDefault("webhook.fixed_workers", 10)
	v.SetDefault("webhook.front_domain", "http://localhost:3000")
	v.SetDefault("webhook.backend_domain", "http://localhost:8889")
	v.SetDefault("webhook.default_upgrade_minutes", 60)
	v.SetDefault("webhook.alert_manager_api", "http://localhost:9093")
	v.SetDefault("webhook.common_map_renew_interval_seconds", 300)
	v.SetDefault("webhook.im_feishu.group_message_api", "https://open.feishu.cn/open-apis/im/v1/messages")
	v.SetDefault("webhook.im_feishu.request_timeout_seconds", 10)
	v.SetDefault("webhook.im_feishu.private_robot_app_id", "")
	v.SetDefault("webhook.im_feishu.private_robot_app_secret", "")
	v.SetDefault("webhook.im_feishu.tenant_access_token_api", "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal")
}

func bindEnvVars() {
	bindStructEnvVars(viper.GetViper(), reflect.TypeOf(Config{}), "")
}

func bindStructEnvVars(v *viper.Viper, t reflect.Type, prefix string) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		mapstructureTag := field.Tag.Get("mapstructure")
		if mapstructureTag == "" {
			continue
		}

		var configKey string
		if prefix == "" {
			configKey = mapstructureTag
		} else {
			configKey = prefix + "." + mapstructureTag
		}

		actualType := field.Type
		if actualType.Kind() == reflect.Ptr {
			actualType = actualType.Elem()
		}

		if actualType.Kind() == reflect.Struct {
			bindStructEnvVars(v, actualType, configKey)
			continue
		}

		envTag := field.Tag.Get("env")
		if envTag != "" {
			_ = v.BindEnv(configKey, envTag)
			continue
		}

		envName := strings.ToUpper(strings.ReplaceAll(configKey, ".", "_"))
		_ = v.BindEnv(configKey, envName)
	}
}

func loadExternalConfig(ext *ExternalConfig) {
	if ext == nil {
		return
	}
	ext.LLM.APIKey = os.Getenv("LLM_API_KEY")
	ext.LLM.BaseURL = os.Getenv("LLM_BASE_URL")
	ext.Aliyun.AccessKeyID = os.Getenv("ALIYUN_ACCESS_KEY_ID")
	ext.Aliyun.AccessKeySecret = os.Getenv("ALIYUN_ACCESS_KEY_SECRET")
	ext.Tavily.APIKey = os.Getenv("TAVILY_API_KEY")
}

// CronAllowLocalExecKey 是否允许本地命令/脚本任务的配置键（环境变量 CRON_ALLOW_LOCAL_EXEC）
const CronAllowLocalExecKey = "cron.allow_local_exec"

// CronLocalExecAllowed 返回当前是否允许执行命令行(2)/脚本(4)任务；未配置时为 false
func CronLocalExecAllowed() bool {
	return viper.GetBool(CronAllowLocalExecKey)
}
