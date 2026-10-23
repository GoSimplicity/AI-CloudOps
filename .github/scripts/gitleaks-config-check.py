#!/usr/bin/env python3
"""检查基准 .gitleaks.toml 是否使用了 [extend] path（review3g N1）。

gitleaks 用 viper 读配置：键名不区分大小写，带点的键（"extend.path"、["extend.path"]）按 "." 拆成层级，
\\u 转义在 TOML 层就被还原。因此用 TOML 解析器（tomllib，Python 3.11+）读出完整结构，把所有键按 viper
的方式（小写、按 "." 拆分）展开成路径，只要有路径以 extend.path 开头就拒绝；同时拒绝：
  - 多个大小写不同的 extend 写法（viper 合并行为未定义）；
  - extend 不是表（字符串/数组表等）；
  - TOML 解析失败；
  - 没有 tomllib（Python < 3.11）。
退出码：0 = 通过；1 = 拒绝/出错（原因写到 stderr）。
用法：python3 gitleaks-config-check.py <配置文件>
"""
import sys


def fail(msg):
    sys.stderr.write("[gitleaks-config-check] ERROR: %s\n" % msg)
    sys.exit(1)


def walk(node, prefix, out):
    if isinstance(node, dict):
        for k, v in node.items():
            parts = prefix + [p.lower() for p in str(k).split(".")]
            out.append(parts)
            walk(v, parts, out)
    elif isinstance(node, list):
        for v in node:
            walk(v, prefix, out)


def main(argv):
    if len(argv) != 2:
        fail("用法: gitleaks-config-check.py <配置文件>")
    try:
        import tomllib
    except ImportError:
        fail("缺少 tomllib（需要 Python 3.11+），无法可靠检查 [extend]，拒绝使用该配置")
    try:
        with open(argv[1], "rb") as fh:
            data = tomllib.load(fh)
    except Exception as exc:  # 解析失败一律拒绝
        fail("配置无法按 TOML 解析（%s），拒绝使用" % exc.__class__.__name__)

    ext_keys = [k for k in data if str(k).split(".")[0].lower() == "extend"]
    if len(ext_keys) > 1:
        fail("配置中有多个 extend 写法（%d 个，大小写/点号不同），拒绝使用" % len(ext_keys))
    for k in ext_keys:
        if "." not in str(k) and not isinstance(data[k], dict):
            fail("配置中的 extend 不是表（%s），拒绝使用" % type(data[k]).__name__)

    paths = []
    walk(data, [], paths)
    for p in paths:
        if len(p) >= 2 and p[0] == "extend" and p[1] == "path":
            fail("配置的 [extend] 使用了 path（会读取 HEAD 工作区中的文件），拒绝使用")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
