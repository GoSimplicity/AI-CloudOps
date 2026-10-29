#!/usr/bin/env python3
"""验收启动后的YAML数据库API，或在独立限权MySQL库中验证完整迁移"""
import argparse
import ipaddress
import json
import os
from pathlib import Path
import re
import secrets
import signal
import subprocess
from urllib.error import HTTPError
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parents[2]


class SchemaError(RuntimeError):
    pass


def command(args, payload=None, env=None, timeout=30):
    process = None
    try:
        process = subprocess.Popen(args, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                   stderr=subprocess.PIPE, env=env, cwd=ROOT,
                                   start_new_session=True)
        output, _ = process.communicate(payload, timeout=timeout)
    except (OSError, subprocess.TimeoutExpired):
        if process is not None:
            if os.name == "posix":
                try:
                    os.killpg(process.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
            else:
                process.kill()
            process.communicate()
        raise SchemaError("验收命令失败或超时，连接信息未输出") from None
    if process.returncode:
        raise SchemaError("验收命令失败，SQL、驱动输出与凭据已隐藏")
    return output


def isolated_mysql(container=None, package="./pkg/di", test="TestMySQLStartupMigrationAndYAMLCRUD"):
    if container is None:
        container = command(["docker", "compose", "-f", "deploy/release/compose.yaml", "ps", "-q", "mysql"]).decode().strip()
    expected = os.environ.get("COMPOSE_PROJECT_NAME", "cloudops")
    if not re.fullmatch(r"cloudops(?:-ci-[0-9]+)?", expected):
        raise SchemaError("仅允许本轮本地或CI的cloudops编排")
    labels = '{{index .Config.Labels "com.docker.compose.project"}} {{index .Config.Labels "com.docker.compose.service"}} {{.State.Health.Status}}'
    if command(["docker", "inspect", "--format", labels, container]).decode().strip() != expected + " mysql healthy":
        raise SchemaError("MySQL容器身份或健康状态不符")
    addresses = command(["docker", "inspect", "--format", "{{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}", container]).decode().split()
    if not addresses or not ipaddress.ip_address(addresses[0]).is_private:
        raise SchemaError("未找到本机私有MySQL容器地址")
    name = "schema_" + secrets.token_hex(12)
    password = secrets.token_hex(24)
    admin_shell = 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql --protocol=socket --user=root --batch --raw --skip-column-names'

    def admin(sql):
        return command(["docker", "exec", "-i", container, "sh", "-c", admin_shell], sql.encode())

    try:
        admin(f"CREATE DATABASE `{name}` CHARACTER SET utf8mb4; CREATE USER '{name}'@'%' IDENTIFIED BY '{password}'; GRANT ALL PRIVILEGES ON `{name}`.* TO '{name}'@'%';")
        env = {**os.environ, "GOTOOLCHAIN": "auto", "CLOUDOPS_MYSQL_TEST_DSN":
               f"{name}:{password}@tcp({addresses[0]}:3306)/{name}?charset=utf8mb4&parseTime=True&loc=UTC"}
        output = command(["go", "test", "-json", "-tags=integration", package, "-run", "^" + test + "$", "-count=1", "-timeout=120s"], env=env, timeout=600)
        events = [json.loads(line) for line in output.splitlines() if line.startswith(b"{")]
        if not any(event.get("Action") == "pass" and event.get("Test") == test for event in events):
            raise SchemaError("隔离MySQL测试没有实际执行成功，不能以跳过充当验收")
    finally:
        # 只回收自身临时用户的连接、数据库与用户，不读取业务配置
        try:
            for connection in admin(f"SELECT ID FROM information_schema.PROCESSLIST WHERE USER='{name}';").splitlines():
                if not connection.isdigit():
                    raise SchemaError("无法安全识别临时测试连接")
                try:
                    admin(f"KILL CONNECTION {int(connection)};")
                except SchemaError:
                    pass
        finally:
            try:
                admin(f"DROP DATABASE IF EXISTS `{name}`;")
            finally:
                admin(f"DROP USER IF EXISTS '{name}'@'%';")
        left = admin(f"SELECT (SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME='{name}')+(SELECT COUNT(*) FROM mysql.user WHERE User='{name}');")
        if left.strip() != b"0":
            raise SchemaError("隔离测试资源未清理完成")
    print("PASS isolated MySQL regression and owned resource cleanup")


def api_check(base):
    def ok(path, payload=None, token=None, method=None):
        headers = {"User-Agent": "cloudops-schema-smoke/1.0"}
        if token:
            headers["Authorization"] = "Bearer " + token
        if payload is not None:
            headers["Content-Type"] = "application/json"
        request = Request(base.rstrip("/") + path, data=json.dumps(payload).encode() if payload is not None else None, headers=headers, method=method)
        try:
            with urlopen(request, timeout=30) as response:
                result = json.load(response)
                status = response.status
        except HTTPError as error:
            raise SchemaError(f"schema API HTTP failure: {error.code}") from None
        if status != 200 or result.get("code") != 0:
            raise SchemaError("schema API business failure; response body hidden")
        return result.get("data")

    token = ok("/api/user/login", {"username": "admin", "password": os.environ["BOOTSTRAP_ADMIN_PASSWORD"]})["accessToken"]
    try:
        for resource in ("yaml_template", "yaml_task"):
            ok(f"/api/k8s/{resource}/1/list?page=1&size=10", token=token)
        print("PASS startup YAML template/task database list APIs")
    finally:
        ok("/api/user/logout", {}, token)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--url", default="http://localhost:18080")
    database = parser.add_mutually_exclusive_group()
    database.add_argument("--mysql-migration", action="store_true")
    database.add_argument("--mysql-notification", action="store_true")
    database.add_argument("--mysql-cloud-account", action="store_true")
    database.add_argument("--mysql-instance-source", action="store_true")
    database.add_argument("--mysql-notification-idempotent", action="store_true")
    database.add_argument("--mysql-notification-outbox", action="store_true")
    database.add_argument("--mysql-cloud-temporary", action="store_true")
    database.add_argument("--mysql-notification-schedule", action="store_true")
    database.add_argument("--mysql-notification-timezone", action="store_true")
    parser.add_argument("--container")
    args = parser.parse_args()
    try:
        if args.mysql_migration:
            isolated_mysql(args.container)
        elif args.mysql_notification:
            isolated_mysql(args.container, "./internal/workorder/dao", "TestMySQLNotificationChannelPagination")
        elif args.mysql_cloud_account:
            isolated_mysql(args.container, "./internal/tree/service", "TestMySQLCloudAccountRegionDelta")
        elif args.mysql_instance_source:
            isolated_mysql(args.container, "./internal/workorder/service", "TestMySQLInstanceNotificationSource")
        elif args.mysql_notification_idempotent:
            isolated_mysql(args.container, "./internal/workorder/dao", "TestMySQLNotificationIdempotentUpdate")
        elif args.mysql_notification_outbox:
            isolated_mysql(args.container, "./internal/workorder/service", "TestMySQLOutboxAtomicAndDelivery")
        elif args.mysql_cloud_temporary:
            isolated_mysql(args.container, "./internal/tree/service", "TestMySQLTemporaryCloudCredentials")
        elif args.mysql_notification_schedule:
            isolated_mysql(args.container, "./internal/workorder/service", "TestMySQLScheduleAtomicConcurrentRestart")
        elif args.mysql_notification_timezone:
            isolated_mysql(args.container, "./pkg/di", "TestMySQLNotificationTimeZoneGuard")
        else:
            api_check(args.url)
    except (SchemaError, KeyError, ValueError):
        raise SystemExit("FAIL schema acceptance; private response, SQL and credentials were not printed") from None


if __name__ == "__main__":
    main()
