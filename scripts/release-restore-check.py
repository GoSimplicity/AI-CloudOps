#!/usr/bin/env python3
"""在本轮 cloudops MySQL 容器中用限权临时用户验收独立数据库恢复"""

import argparse
import hashlib
from pathlib import Path
import re
import secrets
import shutil
import subprocess
import threading


class RestoreError(RuntimeError):
    pass


def identifier(value):
    if not re.fullmatch(r"restore_[a-f0-9]{24}", value):
        raise RestoreError("拒绝非工具生成的恢复资源名称")
    return value


def digest(data):
    return hashlib.sha256(data).hexdigest()


class DockerMySQL:
    def __init__(self, container, timeout):
        self.container = container
        self.timeout = timeout

    def run(self, command, payload=b""):
        try:
            result = subprocess.run(command, input=payload, capture_output=True, timeout=self.timeout)
        except (OSError, subprocess.TimeoutExpired):
            raise RestoreError("MySQL 容器操作失败或超时") from None
        if result.returncode:
            raise RestoreError("MySQL 操作失败；连接详情、SQL 与驱动输出已隐藏")
        return result.stdout

    def verify_container(self):
        labels = '{{index .Config.Labels "com.docker.compose.project"}} {{index .Config.Labels "com.docker.compose.service"}} {{.State.Health.Status}}'
        if self.run(["docker", "inspect", "--format", labels, self.container]).strip() != b"cloudops mysql healthy":
            raise RestoreError("仅允许本轮 healthy 的 cloudops/mysql Compose 容器")

    def admin(self, sql):
        # root 只执行工具内构造的可信管理或源库只读 SQL
        shell = 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql --protocol=socket --user=root --batch --raw --skip-column-names'
        return self.run(["docker", "exec", "-i", self.container, "sh", "-c", shell], sql.encode())

    def restricted_command(self, user, database):
        identifier(user)
        identifier(database)
        shell = ('IFS= read -r restore_password; MYSQL_PWD="$restore_password" exec mysql '
                 '--protocol=socket --batch --raw --skip-column-names --binary-mode '
                 '--skip-commands --skip-named-commands --local-infile=0 '
                 '--user="$1" --database="$2"')
        return ["docker", "exec", "--user", "mysql", "-i", self.container, "sh", "-c", shell, "restore-check", user, database]

    def query(self, user, database, password, sql):
        return self.run(self.restricted_command(user, database), password.encode() + b"\n" + sql.encode())

    def restore(self, user, database, password, dump):
        # 流式导入且丢弃全部客户端输出，不将不可信 SELECT 结果或密码输出到日志
        try:
            process = subprocess.Popen(self.restricted_command(user, database), stdin=subprocess.PIPE, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        except OSError:
            raise RestoreError("无法启动限权恢复客户端") from None
        failures = []

        def write_dump():
            try:
                with dump.open("rb") as source:
                    process.stdin.write(password.encode() + b"\n")
                    shutil.copyfileobj(source, process.stdin, length=1024 * 1024)
            except (OSError, ValueError):
                failures.append(True)
            finally:
                try:
                    process.stdin.close()
                except (OSError, ValueError):
                    pass

        writer = threading.Thread(target=write_dump, daemon=True)
        writer.start()
        try:
            if process.wait(timeout=self.timeout):
                raise RestoreError("限权恢复失败，未放宽数据库或客户端安全权限")
        except subprocess.TimeoutExpired:
            raise RestoreError("恢复超时，临时会话与资源将被清理") from None
        finally:
            if process.poll() is None:
                process.kill()
            process.wait()
            writer.join(timeout=5)
        if writer.is_alive() or failures:
            raise RestoreError("备份流未完整导入")

    def cleanup(self, database, user):
        identifier(database)
        identifier(user)
        # docker exec 客户端超时可能留下 SQL 会话，先终止该临时用户的会话
        try:
            sessions = self.admin(f"SELECT ID FROM information_schema.PROCESSLIST WHERE USER='{user}';")
            for session in sessions.splitlines():
                if not session.isdigit():
                    raise RestoreError("无法安全识别临时恢复会话")
                try:
                    self.admin(f"KILL CONNECTION {int(session)};")
                except RestoreError:
                    # 会话可能在查询与 KILL 之间自行结束，最终以资源检查为准
                    pass
        finally:
            try:
                self.admin(f"DROP DATABASE IF EXISTS `{database}`;")
            finally:
                self.admin(f"DROP USER IF EXISTS '{user}'@'localhost';")
        left = self.admin(f"SELECT (SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME='{database}') + (SELECT COUNT(*) FROM mysql.user WHERE User='{user}');")
        if left.strip() != b"0":
            raise RestoreError("临时恢复资源未清理完成")



def schema_fingerprint(query, database):
    statements = [
        f"SELECT JSON_ARRAY(TABLE_NAME,TABLE_TYPE,IFNULL(ENGINE,''),IFNULL(TABLE_COLLATION,'')) FROM information_schema.TABLES WHERE TABLE_SCHEMA='{database}' ORDER BY TABLE_NAME;",
        f"SELECT JSON_ARRAY(TABLE_NAME,COLUMN_NAME,ORDINAL_POSITION,COLUMN_TYPE,IS_NULLABLE,IFNULL(COLUMN_DEFAULT,'<NULL>'),EXTRA,IFNULL(COLLATION_NAME,'')) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA='{database}' ORDER BY TABLE_NAME,ORDINAL_POSITION;",
        f"SELECT JSON_ARRAY(TABLE_NAME,INDEX_NAME,NON_UNIQUE,SEQ_IN_INDEX,IFNULL(COLUMN_NAME,''),IFNULL(SUB_PART,0),INDEX_TYPE) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA='{database}' ORDER BY TABLE_NAME,INDEX_NAME,SEQ_IN_INDEX;",
        f"SELECT JSON_ARRAY(TABLE_NAME,CONSTRAINT_NAME,COLUMN_NAME,REFERENCED_TABLE_NAME,REFERENCED_COLUMN_NAME) FROM information_schema.KEY_COLUMN_USAGE WHERE TABLE_SCHEMA='{database}' AND REFERENCED_TABLE_NAME IS NOT NULL ORDER BY TABLE_NAME,CONSTRAINT_NAME,ORDINAL_POSITION;",
    ]
    results = [query(sql) for sql in statements]
    return len(results[0].splitlines()), digest(b"\x00".join(results))


def critical_data(query, database):
    admin = query(f"SELECT JSON_ARRAY(id,username,password,enable,account_type,deleted_at) FROM `{database}`.cl_system_users WHERE username='admin' AND deleted_at=0 ORDER BY id;")
    permissions = query(f"SELECT JSON_ARRAY(id,path,method,deleted_at) FROM `{database}`.cl_system_apis WHERE deleted_at=0 ORDER BY id;")
    if len(admin.splitlines()) != 1 or not permissions.strip():
        raise RestoreError("可信关键数据基线缺少唯一管理员或有效权限目录")
    return digest(admin), digest(permissions)


def restore_check(client, dump):
    if not dump.is_file() or dump.stat().st_size == 0:
        raise RestoreError("需要存在且非空的独立 mysql.sql 备份文件")
    client.verify_container()
    expected_schema = schema_fingerprint(client.admin, "cloudops")
    expected_data = critical_data(client.admin, "cloudops")
    if expected_schema[0] == 0:
        raise RestoreError("可信源库没有可核对的表结构")
    database = "restore_" + secrets.token_hex(12)
    user = "restore_" + secrets.token_hex(12)
    password = secrets.token_hex(32)
    # 预检名称未被其他资源占用，再将本次生成的名称纳入 finally 清理范围
    occupied = client.admin(f"SELECT (SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME='{database}') + (SELECT COUNT(*) FROM mysql.user WHERE User='{user}');")
    if occupied.strip() != b"0":
        raise RestoreError("随机恢复名称发生冲突，未修改已有资源")
    try:
        client.admin(f"CREATE DATABASE `{database}` CHARACTER SET utf8mb4; CREATE USER '{user}'@'localhost' IDENTIFIED BY '{password}' WITH MAX_USER_CONNECTIONS 2; GRANT SELECT,INSERT,UPDATE,DELETE,CREATE,DROP,ALTER,INDEX,REFERENCES,LOCK TABLES,TRIGGER,CREATE VIEW,SHOW VIEW ON `{database}`.* TO '{user}'@'localhost';")
        client.restore(user, database, password, dump)
        query = lambda sql: client.query(user, database, password, sql)
        actual_schema = schema_fingerprint(query, database)
        if actual_schema != expected_schema:
            raise RestoreError("恢复结构与可信源库不一致，快照可能存在版本漂移或损坏")
        if critical_data(query, database) != expected_data:
            raise RestoreError("恢复关键数据与可信基线不一致，不能确认恢复成功")
        return expected_schema[0]
    finally:
        try:
            client.cleanup(database, user)
        except RestoreError:
            raise RestoreError(f"临时资源清理失败，请检查本次数据库 {database} 与用户 {user}") from None


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("dump", type=Path)
    parser.add_argument("--container", default="cloudops-mysql-1")
    parser.add_argument("--timeout", type=int, default=120, help="单次容器操作与导入期限，单位秒")
    args = parser.parse_args()
    if args.timeout < 1 or args.timeout > 3600:
        parser.error("timeout 必须在1至3600秒之间")
    try:
        count = restore_check(DockerMySQL(args.container, args.timeout), args.dump)
    except RestoreError as error:
        parser.exit(1, f"恢复验收失败：{error}\n")
    print(f"恢复验收通过：{count} 张表结构、管理员和权限目录匹配；临时数据库与用户已清理")


if __name__ == "__main__":
    main()
