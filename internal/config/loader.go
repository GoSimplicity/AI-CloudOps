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
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/pkg/envmode"
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

	file, err := resolveConfigFile(*configFile)
	if err != nil {
		return nil, nil, err
	}

	return loadFromFile(file)
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

// resolveConfigFile 校验 ENV 并给出要读的配置文件：不论是否传了 --config，
// ENV 未设置、为空（含只有空白）或无法识别都拒绝启动；未传 --config 时按规范化后的 ENV 选默认文件。
func resolveConfigFile(flagValue string) (string, error) {
	mode, err := envmode.Current()
	if err != nil {
		return "", fmt.Errorf("启动环境校验失败: %w", err)
	}
	if flagValue != "" {
		return flagValue, nil
	}
	if mode == envmode.Production {
		return "config/config.production.yaml", nil
	}
	return "config/config.development.yaml", nil
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
	// 只接受 YAML（.json 按 YAML 子集走同一套结构扫描）；其他格式没有原文级校验，文件不存在时也直接拒绝
	if err := checkWebhookConfigExt(configFile); err != nil {
		return nil, nil, err
	}
	// 非空的父键环境变量（WEBHOOK、WEBHOOK_IM_FEISHU）会遮蔽整棵子树，叶子静默变成零值，直接失败
	if err := checkWebhookParentEnvVars(); err != nil {
		return nil, nil, err
	}

	v := viper.New()
	setWebhookDefaults(v)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 配置文件只读一次：viper 与结构扫描用同一份字节，避免两次读取之间文件被替换导致两边看到的内容不一致
	data, readErr := os.ReadFile(configFile)
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return nil, nil, errors.New("读取Webhook配置文件失败: 无法读取文件")
	}
	if readErr == nil {
		v.SetConfigType(strings.TrimPrefix(filepath.Ext(configFile), "."))
		if err := v.ReadConfig(bytes.NewReader(data)); err != nil {
			// 解析错误会回显文件内容（可能含密钥），只保留行号和白名单原因
			return nil, nil, fmt.Errorf("读取Webhook配置文件失败: %s", sanitizeWebhookParseError(err))
		}
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

	// 配置文件结构与原始文本校验（只看文件本身，按 YAML 合并规则算出的生效条目）：多文档、别名键、带引号或带 tag 的 <<、
	// 键上 !!str 以外的显式 tag、NUL 字符、合并链/嵌套超过 64 层、带点号的键、大小写重复键、叶子不允许的显式 tag、
	// 叶子写成 map/列表、父键写成标量/列表、整数字段不是十进制整数，同时收集 webhook 下的未知键。
	// 文件里写错就失败，与该键是否被环境变量覆盖无关
	var scan webhookFileScan
	if readErr == nil {
		var err error
		if scan, err = scanWebhookConfigData(data, filepath.Ext(configFile) == ".json"); err != nil {
			return nil, nil, err
		}
	}

	// 逐键取合并后的值（环境变量 > 配置文件 > 默认值）。v.UnmarshalKey("webhook") 只拿到配置文件里的
	// webhook 子树，环境变量和默认值进不了 cfg（WEBHOOK_PORT 因此从未生效），所以不能直接用。
	// 只构造这一次：解码 cfg 与发布到全局 viper 共用同一份、同一顺序。文件问题与取值问题一起汇总报错
	merged, err := collectWebhookLeafValues(v, scan)
	if err != nil {
		return nil, nil, err
	}
	rawPort, hasPort := merged.vals[webhookPortKey]
	// 与整数字段一致，只去首尾 ASCII 空白（空格、\t、\r、\n）；NBSP、\v、全角空格等不去除，按非法端口报错
	port := strings.Trim(fmt.Sprint(rawPort), webhookIntEnvSpace)
	if !hasPort || rawPort == nil || !isValidPort(port) {
		// 只列出键名和来源，不输出值
		return nil, nil, fmt.Errorf("Webhook 端口 %s必须是 1-65535 的纯数字", webhookRangeLabel(webhookPortKey, merged.src[webhookPortKey]))
	}
	// 去掉前导零后的规范值（如 000080 → 80）同时用于监听和全局 webhook.port
	merged.vals[webhookPortKey] = normalizePort(port)

	cfg, err := decodeWebhookConfig(merged)
	if err != nil {
		return nil, nil, err
	}
	// 整数字段范围校验：失败时与其他校验一样直接返回，不发布到全局
	if err := validateWebhookIntFields(cfg, merged.src); err != nil {
		return nil, nil, err
	}

	// 未知键不发布、不拒绝启动，每次加载只告警一次，只列键名
	if len(scan.unknown) > 0 {
		configWarnf("[配置告警] Webhook 配置文件 %s 中 webhook 下有未识别的键（不会生效，也不发布到全局）：%s", configFile, strings.Join(scan.unknown, ", "))
	}

	// 只把 webhook.* 的已知叶子键发布到全局 viper，供 webhook 子系统现有读取方使用；mysql.* 与未知键不发布。
	// 与 cfg 使用同一份合并后的值。全局 viper 的 Set 不是并发安全的，并发加载时串行写入
	webhookGlobalMu.Lock()
	merged.apply(viper.Set)
	webhookGlobalMu.Unlock()

	secrets := &WebhookSecrets{MySQLAddr: addr}
	if readErr != nil {
		return cfg, secrets, NonFatalError{err: fmt.Errorf("读取Webhook配置文件失败: %w", readErr)}
	}
	return cfg, secrets, nil
}

// webhookGlobalMu 串行化 loadWebhookFromFile 对全局 viper 的写入
var webhookGlobalMu sync.Mutex

// webhookConfigExts 允许的配置文件扩展名（区分大小写）。.json 是 YAML 的子集，与 YAML 走同一套结构扫描
var webhookConfigExts = map[string]bool{".yaml": true, ".yml": true, ".json": true}

// checkWebhookConfigExt 拒绝 YAML/JSON 以外的配置文件格式（toml、ini、properties、hcl、env、无扩展名、大写扩展名等）
func checkWebhookConfigExt(configFile string) error {
	if webhookConfigExts[filepath.Ext(configFile)] {
		return nil
	}
	return fmt.Errorf("Webhook 配置文件只支持 .yaml / .yml（.json 按 YAML 子集解析）格式，不支持 %s", configFile)
}

// webhookParentEnvVars 父键对应的环境变量名：WEBHOOK 与各父键（WEBHOOK_IM_FEISHU），按名字排序
func webhookParentEnvVars() []string {
	names := []string{"WEBHOOK"}
	for p := range webhookKeySchema.parents {
		names = append(names, strings.ToUpper(strings.ReplaceAll(p, ".", "_")))
	}
	sort.Strings(names)
	return names
}

// checkWebhookParentEnvVars 父键环境变量非空即失败（空串与未设置等同，与 viper 一致），只列变量名
func checkWebhookParentEnvVars() error {
	var set []string
	for _, name := range webhookParentEnvVars() {
		if os.Getenv(name) != "" {
			set = append(set, name)
		}
	}
	if len(set) == 0 {
		return nil
	}
	return fmt.Errorf("环境变量 %s 对应的是父键（整棵子树），非空时会遮蔽其下全部叶子键、使其静默变成零值，不支持；请删除或置空，改用叶子变量（如 WEBHOOK_PORT、WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS）",
		strings.Join(set, ", "))
}

// webhookParseLineRe 只从锚定位置取行号：开头、“yaml: ” 之后或换行缩进之后紧跟的 “line N:”，
// 不会误取键名、取值或 Go 字段路径里出现的 “line 7”
var webhookParseLineRe = regexp.MustCompile(`(?:^|yaml: |\n[ \t]*)line ([0-9]+):`)

// webhookParseReasons 解析错误原因白名单：按顺序匹配错误串（小写）中的片段，输出固定的中文说明，不回显任何原文
var webhookParseReasons = []struct{ match, text string }{
	{"already defined", "同一层有重复的键"},
	{"invalid map key", "键不是合法的 map 键"},
	{"tab character", "缩进里有 Tab 字符"},
	{"cannot start any token", "包含不能出现在此处的字符（如 Tab 缩进）"},
	{"mapping values are not allowed", "冒号或缩进位置错误"},
	{"did not find expected", "结构不完整（缺少预期的键、值、结束引号或括号）"},
	{"unexpected end", "内容意外结束（缺少结束引号或括号）"},
	{"unknown anchor", "引用了未定义的锚点"},
	{"anchor", "锚点用法错误"},
	{"unicode", "字符串转义或字符编码无效"},
	{"escape", "字符串转义或字符编码无效"},
	{"utf-8", "不是合法的 UTF-8 编码"},
	{"control characters", "包含不允许的控制字符"},
	{"cannot unmarshal", "取值类型不符或数值超出范围"},
	{"invalid character", "JSON 语法错误"},
}

// sanitizeWebhookParseError 把 viper / yaml.v3 / encoding/json 的解析错误收敛成“配置文件第 N 行: 原因”（无行号时为
// “配置文件解析失败: 原因”）。原因只取白名单里的固定说明，匹配不上一律写“无法解析（YAML/JSON 语法错误）”，
// 不回显错误串里的任何原文（键名、取值片段、数字、Go 字段路径）
func sanitizeWebhookParseError(err error) string {
	s := strings.ReplaceAll(err.Error(), "While parsing config: ", "")
	prefix := "配置文件解析失败: "
	if m := webhookParseLineRe.FindStringSubmatch(s); m != nil {
		prefix = "配置文件第 " + m[1] + " 行: "
	}
	lower := strings.ToLower(s)
	for _, r := range webhookParseReasons {
		if strings.Contains(lower, r.match) {
			return prefix + r.text
		}
	}
	return prefix + "无法解析（YAML/JSON 语法错误）"
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

// webhookIssuePrefix 文件写法问题与取值类型问题汇总报错的前缀
const webhookIssuePrefix = "Webhook 配置写法或类型错误: "

// webhookIssues 汇总的问题：排序键 → 说明。按排序键输出，结果稳定；说明只含键名、变量名或行号，不含取值
type webhookIssues map[string]string

// addKey 记一条与键路径相关的问题（排序键以键路径开头，同一键的不同问题用 kind 区分）
func (is webhookIssues) addKey(key, kind, text string) {
	is[key+"\x00"+kind] = text
}

// addLine 记一条只能用行号定位的问题，排在键路径类问题之后
func (is webhookIssues) addLine(line int, kind, text string) {
	is[fmt.Sprintf("~%09d\x00%s", line, kind)] = text
}

func (is webhookIssues) err() error {
	if len(is) == 0 {
		return nil
	}
	keys := make([]string, 0, len(is))
	for k := range is {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	items := make([]string, len(keys))
	for i, k := range keys {
		items[i] = is[k]
	}
	return errors.New(webhookIssuePrefix + strings.Join(items, "；"))
}

// webhookSourceLabel 报错时按值的实际来源描述键：配置文件 / 环境变量 / 内置默认值
func webhookSourceLabel(key, src string) string {
	switch src {
	case webhookSrcFile:
		return "配置文件中的 " + key
	case webhookSrcEnv:
		env := ""
		if leaf, ok := webhookKeySchema.leaves[key]; ok {
			env = leaf.env
		}
		return fmt.Sprintf("%s（环境变量 %s）", key, env)
	case webhookSrcDefault:
		return key + "（内置默认值）"
	}
	return key
}

// 取值来源
const (
	webhookSrcFile    = "file"
	webhookSrcEnv     = "env"
	webhookSrcDefault = "default"
)

// webhookTypeErrorText 某个叶子/父键类型不对时的说明（只含键名、来源，不含取值）
func webhookTypeErrorText(key, src string) string {
	label := webhookRangeLabel(key, src)
	if leaf, ok := webhookKeySchema.leaves[key]; ok {
		if leaf.kind == reflect.Int {
			return label + "应为十进制整数"
		}
		return label + "应为字符串"
	}
	return label + "应为 map 或留空"
}

// webhookFileScan 配置文件本身的扫描结果
type webhookFileScan struct {
	bad     map[string]bool   // 已记问题的键（叶子或父键），取值阶段跳过这些键及其子键
	issues  webhookIssues     // 文件写法问题
	unknown []string          // webhook 下不认识的键（已排序去重，不展开子键）
	raw     map[string]string // 叶子在文件里生效条目的原始标量文本（非 null）
}

func (s *webhookFileScan) addBad(key, kind, text string) {
	if s.bad == nil {
		s.bad = map[string]bool{}
	}
	if s.issues == nil {
		s.issues = webhookIssues{}
	}
	s.bad[key] = true
	s.issues.addKey(key, kind, text)
}

func (s *webhookFileScan) addLine(line int, kind, text string) {
	if s.issues == nil {
		s.issues = webhookIssues{}
	}
	s.issues.addLine(line, kind, text)
}

// webhookYAMLMaxDepth 遍历合并结果时的最大深度（防御异常输入，正常配置远小于此）
const webhookYAMLMaxDepth = 64

// scanWebhookConfigData 用 yaml.v3 解析配置文件内容（与 viper 读取的是同一份字节；YAML/JSON，扩展名已由
// checkWebhookConfigExt 限定，isJSON 时先把 JSON 的代理对转义和 \/ 换成 YAML 能识别的写法），检查：
//   - 只允许一个 YAML 文档（viper 只读第一个，其余会被静默忽略）；
//   - 全文件结构：键写成别名（*x）、带引号或带 tag 的 <<、同一 mapping 多个 <<、键上 !!str 以外的显式 tag、键或值含 NUL；
//   - 全文件按 YAML 合并结果：同一父节点下小写后相同的键（大小写变体或完全重复，口径同 check-webhook-env.sh），根节点带点号的键，
//     合并链或嵌套超过 webhookYAMLMaxDepth 层；
//   - webhook 子树（按合并结果，只看生效条目）：带点号的键、父键/叶子类型、叶子的显式 tag、整数叶子的十进制原文
func scanWebhookConfigData(data []byte, isJSON bool) (webhookFileScan, error) {
	scan := webhookFileScan{bad: map[string]bool{}, issues: webhookIssues{}, raw: map[string]string{}}
	if isJSON {
		data = jsonEscapesToYAML(data)
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		if errors.Is(err, io.EOF) {
			return scan, nil
		}
		return scan, fmt.Errorf("读取Webhook配置文件失败: %s", sanitizeWebhookParseError(err))
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return scan, fmt.Errorf("读取Webhook配置文件失败: %s", sanitizeWebhookParseError(err))
		}
		return scan, errors.New("读取Webhook配置文件失败: 配置文件包含多个 YAML 文档（---），只支持一个")
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return scan, nil
	}
	checkYAMLKeyStructure(&doc, &scan)
	root := resolveYAMLAlias(doc.Content[0])
	if root.Kind != yaml.MappingNode {
		return scan, nil
	}
	checkYAMLCaseDuplicates(root, "", &scan, 0)

	unknown := map[string]bool{}
	for _, p := range effectiveYAMLPairs(root, 0, &scan) {
		lower := strings.ToLower(p.k.Value)
		if strings.Contains(lower, ".") {
			addDottedKeyIssue(&scan, lower, p.k.Line)
			continue
		}
		if lower != "webhook" {
			continue
		}
		switch {
		case isYAMLNull(p.v):
			// 根节点为空：走“没有 webhook 根节点”的告警分支
		case p.v.Kind == yaml.MappingNode:
			scanWebhookMapping(p.v, "webhook", &scan, unknown, 0)
		default:
			scan.addBad("webhook", "type", webhookTypeErrorText("webhook", webhookSrcFile))
		}
	}
	for k := range unknown {
		scan.unknown = append(scan.unknown, k)
	}
	sort.Strings(scan.unknown)
	return scan, nil
}

// addDottedKeyIssue 带点号的键：拼出的路径是 webhook 已知键时写键名，否则只写行号（不回显任意键名）
func addDottedKeyIssue(scan *webhookFileScan, path string, line int) {
	if _, ok := webhookKeySchema.leaves[path]; ok || webhookKeySchema.parents[path] || path == "webhook" {
		scan.addBad(path, "dotted", fmt.Sprintf("配置文件中的 %s 写成了带点号的键名（第 %d 行），请改成嵌套写法", path, line))
		return
	}
	scan.addLine(line, "dotted", fmt.Sprintf("配置文件第 %d 行的键名带点号，请改成嵌套写法", line))
}

// scanWebhookMapping 按合并结果检查 prefix 对应 mapping 下的每个生效键：
//   - 键名带点号：拒绝；
//   - 白名单父键：只能是 map 或 null，map 时递归；
//   - 叶子：只能是标量（null 等同未写）；显式 tag 只允许 !!str、!!int、!!null；整数叶子的原始文本必须是十进制整数；
//   - 其他：记为未知键，不展开子键
func scanWebhookMapping(m *yaml.Node, prefix string, scan *webhookFileScan, unknown map[string]bool, depth int) {
	if depth > webhookYAMLMaxDepth {
		return // schema 只有两层，不可达；超深嵌套已由 checkYAMLCaseDuplicates 报错
	}
	for _, p := range effectiveYAMLPairs(m, 0, scan) {
		k, val := p.k, p.v
		lower := strings.ToLower(k.Value)
		key := prefix + "." + lower
		if strings.Contains(lower, ".") {
			addDottedKeyIssue(scan, key, k.Line)
			continue
		}
		if webhookKeySchema.parents[key] {
			switch {
			case isYAMLNull(val):
			case val.Kind == yaml.MappingNode:
				scanWebhookMapping(val, key, scan, unknown, depth+1)
			default:
				scan.addBad(key, "type", webhookTypeErrorText(key, webhookSrcFile))
			}
			continue
		}
		leaf, ok := webhookKeySchema.leaves[key]
		if !ok {
			unknown[key] = true
			continue
		}
		if val.Kind != yaml.ScalarNode {
			scan.addBad(key, "type", webhookTypeErrorText(key, webhookSrcFile))
			continue
		}
		if val.Style&yaml.TaggedStyle != 0 && !webhookAllowedLeafTags[val.ShortTag()] {
			scan.addBad(key, "tag", fmt.Sprintf("配置文件中的 %s 使用了不允许的 YAML tag（只允许 !!str、!!int、!!null 或不写）", key))
			continue
		}
		if isYAMLNull(val) {
			continue
		}
		if leaf.kind == reflect.Int && !webhookDecimalInt.MatchString(val.Value) {
			scan.addBad(key, "type", webhookTypeErrorText(key, webhookSrcFile))
			continue
		}
		// 大小写变体/重复键已由 checkYAMLCaseDuplicates 记到 scan.bad，取值阶段会跳过，这里不必再处理 raw 冲突
		scan.raw[key] = val.Value
	}
}

// webhookAllowedLeafTags 叶子允许的显式 tag
var webhookAllowedLeafTags = map[string]bool{"!!str": true, "!!int": true, "!!null": true}

// yamlPair 合并结果中的一个生效条目；v 已解析别名
type yamlPair struct {
	k, v *yaml.Node
}

// isYAMLMergeKey 只认不带引号、不带 tag、解析为 !!merge 的 << 为合并键。
// 带引号或带 tag 的 << 由结构检查一律拒绝：yaml.v3 中带引号的 "<<"（无 tag）是普通键、不合并，
// 而 !!merge "<<"、!!merge << 会真的合并，两种都不支持
func isYAMLMergeKey(k *yaml.Node) bool {
	return k.Kind == yaml.ScalarNode && k.Value == "<<" && k.ShortTag() == "!!merge" &&
		k.Style&(yaml.DoubleQuotedStyle|yaml.SingleQuotedStyle|yaml.TaggedStyle) == 0
}

// effectiveYAMLPairs 按 yaml.v3 的合并规则算出 mapping 的生效条目：
//   - 本层显式写的键全部保留（含重复，交给重复键检查），优先于合并来的键；
//   - << 的值是 mapping 或 mapping 列表（可为别名），列表里靠前的优先；来源自身的显式键优先于来源内部的 <<（递归）；
//   - 合并来的键只有在同拼写的键尚未出现时才加入（与 yaml.v3 一致：大小写不同的变体都会进入结果）。
//
// 写成别名的键、带引号或带 tag 的 << 不进入结果（由 checkYAMLKeyStructure 报错）。
// 合并链超过 webhookYAMLMaxDepth 层时记校验错误（按行号），不再静默截断
func effectiveYAMLPairs(m *yaml.Node, depth int, scan *webhookFileScan) []yamlPair {
	var out []yamlPair
	var merge *yaml.Node
	taken := map[string]bool{}
	for i := 0; i+1 < len(m.Content); i += 2 {
		k := m.Content[i]
		if isYAMLMergeKey(k) {
			merge = resolveYAMLAlias(m.Content[i+1]) // 与 yaml.v3 一致，多个 << 只有最后一个生效（结构检查已拒绝）
			continue
		}
		if k.Kind != yaml.ScalarNode || k.Value == "<<" {
			continue
		}
		out = append(out, yamlPair{k: k, v: resolveYAMLAlias(m.Content[i+1])})
		taken[k.Value] = true
	}
	if merge == nil {
		return out
	}
	if depth >= webhookYAMLMaxDepth {
		scan.addLine(m.Line, "mergedepth", fmt.Sprintf("配置文件第 %d 行：<< 合并链超过 %d 层，不支持", m.Line, webhookYAMLMaxDepth))
		return out
	}
	var sources []*yaml.Node
	switch merge.Kind {
	case yaml.MappingNode:
		sources = []*yaml.Node{merge}
	case yaml.SequenceNode:
		for _, item := range merge.Content {
			if item = resolveYAMLAlias(item); item.Kind == yaml.MappingNode {
				sources = append(sources, item)
			}
		}
	}
	for _, src := range sources {
		for _, p := range effectiveYAMLPairs(src, depth+1, scan) {
			if !taken[p.k.Value] {
				taken[p.k.Value] = true
				out = append(out, p)
			}
		}
	}
	return out
}

// checkYAMLKeyStructure 遍历整份文件的原始节点（不跟随别名，锚点处已遍历过），按行号报（不回显键名和取值）：
// 键写成别名、带引号或带 tag 的 <<、同一 mapping 下多个 <<、键上 !!str 以外的显式 tag（如 !!binary 键会被解码成
// 另一个键名，绕过原文校验）、键或值含 NUL 字符
func checkYAMLKeyStructure(n *yaml.Node, scan *webhookFileScan) {
	switch n.Kind {
	case yaml.ScalarNode:
		if strings.ContainsRune(n.Value, 0) {
			scan.addLine(n.Line, "nul", fmt.Sprintf("配置文件第 %d 行：键或值包含 NUL 字符（\\u0000），不支持", n.Line))
		}
	case yaml.MappingNode:
		merges := 0
		for i := 0; i+1 < len(n.Content); i += 2 {
			k := n.Content[i]
			switch {
			case k.Kind == yaml.AliasNode:
				scan.addLine(k.Line, "alias", fmt.Sprintf("配置文件第 %d 行：键不能写成 YAML 别名（*名称），请直接写键名", k.Line))
			case isYAMLMergeKey(k):
				if merges++; merges == 2 {
					scan.addLine(k.Line, "merge2", fmt.Sprintf("配置文件第 %d 行：同一层只能写一个 <<（多个时只有最后一个生效），请合并成 <<: [*a, *b]", k.Line))
				}
			case k.Kind == yaml.ScalarNode && (k.Value == "<<" || k.ShortTag() == "!!merge"):
				scan.addLine(k.Line, "qmerge", fmt.Sprintf("配置文件第 %d 行：带引号或带 tag 的 << 不被支持（合并请写不带引号、不带 tag 的 <<）", k.Line))
			case k.Kind == yaml.ScalarNode && k.Style&yaml.TaggedStyle != 0 && k.ShortTag() != "!!str":
				scan.addLine(k.Line, "keytag", fmt.Sprintf("配置文件第 %d 行：键使用了不允许的 YAML tag（键只允许不写 tag 或 !!str）", k.Line))
			}
			checkYAMLKeyStructure(k, scan)
			checkYAMLKeyStructure(n.Content[i+1], scan)
		}
	case yaml.SequenceNode, yaml.DocumentNode:
		for _, c := range n.Content {
			checkYAMLKeyStructure(c, scan)
		}
	}
}

// checkYAMLCaseDuplicates 按合并结果检查整份文件：同一父节点下小写后相同的键（大小写变体或完全相同）不合规，
// 口径同 scripts/check-webhook-env.sh（ee753b4）：只报键路径（小写），列表项里的 mapping 不参与
func checkYAMLCaseDuplicates(m *yaml.Node, prefix string, scan *webhookFileScan, depth int) {
	if depth > webhookYAMLMaxDepth {
		scan.addLine(m.Line, "nestdepth", fmt.Sprintf("配置文件第 %d 行：嵌套超过 %d 层，不支持", m.Line, webhookYAMLMaxDepth))
		return
	}
	groups := map[string][]yamlPair{}
	var order []string
	for _, p := range effectiveYAMLPairs(m, 0, scan) {
		lower := strings.ToLower(p.k.Value)
		if _, ok := groups[lower]; !ok {
			order = append(order, lower)
		}
		groups[lower] = append(groups[lower], p)
	}
	for _, lower := range order {
		path := lower
		if prefix != "" {
			path = prefix + "." + lower
		}
		if len(groups[lower]) > 1 {
			scan.addBad(path, "dup", fmt.Sprintf("配置文件中的 %s 有仅大小写不同或完全相同的同层重复键（会互相覆盖，生效哪一个不确定）", path))
		}
		for _, p := range groups[lower] {
			if p.v.Kind == yaml.MappingNode {
				checkYAMLCaseDuplicates(p.v, path, scan, depth+1)
			}
		}
	}
}

// jsonEscapesToYAML 把 JSON 字符串里 YAML 不认的两种合法转义换成等价写法，只用于结构扫描（viper 仍按 encoding/json
// 解析原始字节）：UTF-16 代理对（\uD83D\uDE00）换成 \U0001F600，\/ 换成 /。
// 只处理双引号字符串内部、未被反斜杠转义的序列；落单的代理项原样保留，交给解析器报错
func jsonEscapesToYAML(data []byte) []byte {
	if !bytes.Contains(data, []byte(`\`)) {
		return data
	}
	out := make([]byte, 0, len(data))
	inStr := false
	for i := 0; i < len(data); i++ {
		c := data[i]
		switch {
		case !inStr:
			inStr = c == '"'
		case c == '"':
			inStr = false
		case c == '\\' && i+1 < len(data):
			if hi, ok := jsonHex4(data, i+1); ok && hi >= 0xD800 && hi <= 0xDBFF && i+11 < len(data) && data[i+6] == '\\' {
				if lo, ok := jsonHex4(data, i+7); ok && lo >= 0xDC00 && lo <= 0xDFFF {
					out = fmt.Appendf(out, "\\U%08X", 0x10000+(hi-0xD800)<<10+(lo-0xDC00))
					i += 11
					continue
				}
			}
			if data[i+1] == '/' {
				out = append(out, '/')
			} else {
				out = append(out, c, data[i+1])
			}
			i++
			continue
		}
		out = append(out, c)
	}
	return out
}

// jsonHex4 解析 data[i:] 开头的 “u” + 4 位十六进制
func jsonHex4(data []byte, i int) (rune, bool) {
	if i+5 > len(data) || data[i] != 'u' {
		return 0, false
	}
	n, err := strconv.ParseUint(string(data[i+1:i+5]), 16, 32)
	if err != nil {
		return 0, false
	}
	return rune(n), true
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

// webhookValues 逐键合并后的 webhook.* 叶子键取值；keys 已排序，写入顺序确定；src 记录每个键的取值来源
type webhookValues struct {
	keys []string
	vals map[string]any
	src  map[string]string
}

// apply 按 keys 的固定顺序逐个写入（发布到全局 viper 时使用）
func (m webhookValues) apply(set func(key string, val any)) {
	for _, key := range m.keys {
		set(key, m.vals[key])
	}
}

// webhookIntEnvSpace 整数类环境变量与端口允许去掉的首尾 ASCII 空白（NBSP、\v、\f、全角空格等不在内）
const webhookIntEnvSpace = " \t\r\n"

// webhookURLLeaves URL 类字符串叶子（环境变量、配置文件、默认值同样处理）：去首尾空白；值为 true 的是拼接前缀，
// 再去尾部斜杠（避免拼出 hook/ /<token>、//api/...）。tenant_access_token_api 是完整接口地址，只去空白
var webhookURLLeaves = map[string]bool{
	"webhook.front_domain":                      true,
	"webhook.backend_domain":                    true,
	"webhook.alert_manager_api":                 true,
	"webhook.im_feishu.group_message_api":       true,
	"webhook.im_feishu.tenant_access_token_api": false,
}

func trimWebhookURL(s string, trimSlash bool) string {
	s = strings.TrimSpace(s)
	if trimSlash {
		s = strings.TrimRight(s, "/")
	}
	return s
}

// webhookUnderBadKey key 或其某个上级键已记过文件问题
func webhookUnderBadKey(key string, bad map[string]bool) bool {
	for k := key; ; {
		if bad[k] {
			return true
		}
		i := strings.LastIndex(k, ".")
		if i < 0 {
			return false
		}
		k = k[:i]
	}
}

// collectWebhookLeafValues 按叶子类型表取每个叶子键合并后的值（环境变量 > 配置文件 > 默认值），并规范成字段的 Go 类型：
//   - 只遍历叶子表，父键（白名单）与未知键天然不进入合并值；
//   - 叶子取不到值（IsSet=false，父键被遮蔽）一律报错，不跳过；
//   - 整数叶子：环境变量去掉首尾 ASCII 空白（空格、\t、\r、\n）后必须是十进制整数；取自配置文件的按文件生效条目的原始文本解析
//     （010 不会变成 8，10.5 不会截断成 10）；超出 int 范围的按边界值保留，交给范围校验报错；
//   - 字符串叶子：环境变量原样保留；取自配置文件的非字符串标量（如未加引号的 12345、true）用文件原始文本；
//   - map、列表或其他无法表示的值报类型错误。
//
// 与 scan 的文件问题一起汇总报错，按来源写“配置文件中的 KEY”或“KEY（环境变量 ENV）”，只列键名
func collectWebhookLeafValues(v *viper.Viper, scan webhookFileScan) (webhookValues, error) {
	issues := webhookIssues{}
	for k, t := range scan.issues {
		issues[k] = t
	}
	m := webhookValues{vals: make(map[string]any, len(webhookKeySchema.leafKeys)), src: map[string]string{}}
	for _, key := range webhookKeySchema.leafKeys {
		if webhookUnderBadKey(key, scan.bad) {
			continue
		}
		if !v.IsSet(key) {
			issues.addKey(key, "unset", fmt.Sprintf("%s 取不到值：其父键被环境变量或配置文件遮蔽", key))
			continue
		}
		leaf := webhookKeySchema.leaves[key]
		val := v.Get(key)
		raw, fromFile := scan.raw[key]
		src := webhookSrcDefault
		var (
			out any
			ok  bool
		)
		switch {
		case os.Getenv(leaf.env) != "":
			src = webhookSrcEnv
			if s, isStr := val.(string); isStr && leaf.kind == reflect.Int {
				out, ok = parseWebhookDecimalInt(strings.Trim(s, webhookIntEnvSpace))
			} else if leaf.kind == reflect.Int {
				out, ok = normalizeWebhookInt(val, "", false)
			} else {
				out, ok = normalizeWebhookString(val, "", false)
			}
		case fromFile:
			src = webhookSrcFile
			if leaf.kind == reflect.Int {
				out, ok = parseWebhookDecimalInt(raw)
			} else {
				out, ok = normalizeWebhookString(val, raw, true)
			}
		case v.InConfig(key):
			// viper 认为值来自配置文件、扫描却没拿到可用原文（如经别名键、带 tag 的键等非常规写法写入，结构检查通常已报错）：
			// 无法按原文校验，不能按内置默认值的口径放行
			issues.addKey(key, "noraw", fmt.Sprintf("配置文件中的 %s 无法按原文校验（可能经别名键、带 tag 的键等非常规写法写入），请改成普通写法", key))
			continue
		default:
			if leaf.kind == reflect.Int {
				out, ok = normalizeWebhookInt(val, "", false)
			} else {
				out, ok = normalizeWebhookString(val, "", false)
			}
		}
		if !ok {
			issues.addKey(key, "type", webhookTypeErrorText(key, src))
			continue
		}
		if trimSlash, isURL := webhookURLLeaves[key]; isURL {
			out = trimWebhookURL(out.(string), trimSlash)
		}
		m.keys = append(m.keys, key)
		m.vals[key] = out
		m.src[key] = src
	}
	if err := issues.err(); err != nil {
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

// normalizeWebhookInt 规范整数叶子的取值。字符串按严格十进制解析；数值类型来自配置文件时按原始文本解析，
// 否则（内置默认值）只接受整数类型，浮点不截断
func normalizeWebhookInt(val any, raw string, fromFile bool) (any, bool) {
	switch x := val.(type) {
	case string:
		return parseWebhookDecimalInt(x)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		if fromFile {
			// 数值来自配置文件：按原始文本解析，避免 YAML 的进制/浮点转换
			return parseWebhookDecimalInt(raw)
		}
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
//   - im_feishu.request_timeout_seconds：用作 http.Client.Timeout（秒）。Go 中 0 和负数都表示不设超时，
//     飞书接口卡住时请求会永久挂起，因此最小 1；上限 600（10 分钟），单次 HTTP 请求等更久与不设超时无异
var webhookIntRules = []webhookIntRule{
	{"webhook.fixed_workers", "WEBHOOK_FIXED_WORKERS", 1, 1024, func(c *WebhookConfig) int { return c.FixedWorkers }},
	{"webhook.default_upgrade_minutes", "WEBHOOK_DEFAULT_UPGRADE_MINUTES", 1, 10080, func(c *WebhookConfig) int { return c.DefaultUpgradeMinutes }},
	{"webhook.common_map_renew_interval_seconds", "WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS", 1, 86400, func(c *WebhookConfig) int { return c.CommonMapRenewIntervalSeconds }},
	{"webhook.im_feishu.request_timeout_seconds", "WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS", 1, 600, func(c *WebhookConfig) int { return c.ImFeishu.RequestTimeoutSeconds }},
}

// validateWebhookIntFields 按 webhookIntRules 校验整数字段，列出全部不合法的键；报错按取值来源写
// “配置文件中的 KEY” / “KEY（环境变量 ENV）” / “KEY（内置默认值）”，只含键名、变量名和上下限，不含取值。
// src 为空（来源未知）时只写键名
func validateWebhookIntFields(cfg *WebhookConfig, src map[string]string) error {
	var bad []string
	for _, r := range webhookIntRules {
		if n := r.get(cfg); n < r.min || n > r.max {
			bad = append(bad, fmt.Sprintf("%s必须在 %d 到 %d 之间", webhookRangeLabel(r.key, src[r.key]), r.min, r.max))
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("Webhook 整数配置超出允许范围: %s", strings.Join(bad, "；"))
	}
	return nil
}

// webhookRangeLabel 类型/范围/端口报错里的键描述：来源未知时只写键名；以全角括号结尾时不再补空格
func webhookRangeLabel(key, src string) string {
	label := webhookSourceLabel(key, src)
	if strings.HasSuffix(label, "）") {
		return label
	}
	return label + " "
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

	// mock 默认值随环境变化：生产环境（ENV 为 production/prod/prd，或未设置/无法识别）默认关闭，开发/测试环境保持开启。
	// 显式配置（配置文件 mock.enabled 或环境变量 MOCK_ENABLED）仍然优先；生产环境开启会被 Validate 拒绝。
	viper.SetDefault("mock.enabled", defaultMockEnabled())

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
	// 群聊机器人 webhook 地址前缀，发送时拼成 <前缀>/<机器人 token>（见 content.SentFeishuGroup），不带尾斜杠
	v.SetDefault("webhook.im_feishu.group_message_api", "https://open.feishu.cn/open-apis/bot/v2/hook")
	v.SetDefault("webhook.im_feishu.request_timeout_seconds", 10)
	v.SetDefault("webhook.im_feishu.private_robot_app_id", "")
	v.SetDefault("webhook.im_feishu.private_robot_app_secret", "")
	v.SetDefault("webhook.im_feishu.tenant_access_token_api", "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal")
}

// IsProduction 判断当前进程是否应按生产环境处理，委托给 pkg/envmode，
// 与 Load 选择配置文件、Validate 的生产校验、pkg/ssh 的主机密钥策略使用同一判定。
// ENV 会先 TrimSpace、ToLower 再匹配别名（production/prod/prd）；未设置或未知取值按生产处理（fail closed），
// 而 Load 会对这两种情况直接拒绝启动。
func IsProduction() bool {
	return envmode.IsProduction()
}

// defaultMockEnabled 仅在明确开发环境时开启 mock，其他环境关闭。
func defaultMockEnabled() bool {
	return !IsProduction()
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
