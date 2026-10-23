#!/usr/bin/env python3
"""在 AIOps 容器内核对实际 HTTP/WebSocket 路由与 Go 权限目录，不发送请求"""

import contextlib
import io
import json
import re
import sys


def main():
    # FastAPI 0.159.1 将 include_router 延迟展开，此处使用已锁定版本的有效路由上下文
    with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
        from app.main import app
        from fastapi.routing import APIRoute, APIWebSocketRoute, _IncludedRouter

    declared = json.loads(sys.argv[1])
    expected = {(row["method"], row["path"]) for row in declared}
    if len(expected) != len(declared):
        raise RuntimeError("AI permission catalogue contains duplicate entries")
    actual = set()
    supported = {"GET", "POST", "PUT", "DELETE"}
    for entry in app.routes:
        contexts = entry.effective_route_contexts() if isinstance(entry, _IncludedRouter) else [entry]
        for route in contexts:
            original = getattr(route, "original_route", route)
            path = (
                getattr(route, "path_format", None)
                or getattr(getattr(route, "starlette_route", None), "path", None)
                or getattr(route, "path", "")
            )
            if not path.startswith("/api/v1/"):
                continue
            if isinstance(original, APIRoute):
                methods = original.methods
            elif isinstance(original, APIWebSocketRoute):
                methods = {"GET"}
            else:
                raise RuntimeError("Uncatalogued AI route type: " + type(original).__name__)
            if not methods <= supported:
                raise RuntimeError("AI route uses unsupported permission method: " + path)
            path = re.sub(r"\{([^}:]+)(?::[^}]+)?\}", r":\1", path)
            actual.update((method, path) for method in methods)
    if not actual:
        raise RuntimeError("AI route inventory is empty")
    if actual != expected:
        raise RuntimeError(
            f"AI route catalogue drift: missing={sorted(actual - expected)}, stale={sorted(expected - actual)}"
        )
    print(f"PASS {len(actual)} actual AI HTTP/WebSocket route permissions, including hidden aliases")


if __name__ == "__main__":
    main()
