#!/usr/bin/env python3
"""Quiesced, authenticated backup and new-volume restore for CloudOps.

Default is dry-run. The operator must stop every writer before an actual backup
and must keep restored services quarantined until external effects are audited.
"""
from __future__ import annotations

import argparse
from datetime import datetime, timezone
import hashlib
import hmac
import json
import os
from pathlib import Path, PurePosixPath
import re
import secrets
import selectors
import shutil
import stat
import subprocess
import tarfile
import tempfile
import time
from typing import BinaryIO

MAX_ARCHIVE = 4 * 1024**3
MAX_ENTRIES = 500_000
ROLES = ("mysql", "redis", "aiops")
IMAGE = "redis:7.4.11-alpine"
DOCKER_SOCKET = ""


def docker_command(*args: str) -> list[str]:
    if not DOCKER_SOCKET:
        raise RecoveryError("必须显式指定本机 Docker socket")
    return ["docker", "--host", f"unix://{DOCKER_SOCKET}", *args]


def configure_docker(path: Path | None) -> None:
    global DOCKER_SOCKET
    if path is None:
        raise RecoveryError("必须显式指定本机 Docker socket")
    try:
        resolved = path.resolve(strict=True)
        metadata = resolved.stat()
        if not stat.S_ISSOCK(metadata.st_mode):
            raise RecoveryError("Docker socket 无效")
    except OSError:
        raise RecoveryError("无法访问本机 Docker socket") from None
    DOCKER_SOCKET = str(resolved)


class RecoveryError(Exception):
    pass


def docker(*args: str, timeout: int = 30) -> str:
    try:
        result = subprocess.run(docker_command(*args), capture_output=True, timeout=timeout, check=False)
    except (OSError, subprocess.TimeoutExpired):
        raise RecoveryError("容器工具不可用或超时") from None
    if result.returncode:
        raise RecoveryError("容器操作失败；原始输出已隐藏")
    return result.stdout.decode().strip()


def private_file(path: Path, limit: int) -> bytes:
    try:
        fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
        with os.fdopen(fd, "rb") as source:
            metadata = os.fstat(source.fileno())
            if not stat.S_ISREG(metadata.st_mode) or metadata.st_uid != os.getuid() or metadata.st_mode & 0o077:
                raise RecoveryError("输入文件须为当前用户私有的普通文件")
            data = source.read(limit + 1)
    except OSError:
        raise RecoveryError("无法安全读取私有文件") from None
    if not data or len(data) > limit:
        raise RecoveryError("输入文件为空或超过上限")
    return data


def load_key(path: Path) -> bytes:
    key = private_file(path, 4096)
    if len(key) < 32:
        raise RecoveryError("完整性密钥至少需要32字节")
    return key


def safe_directory(path: Path) -> Path:
    # Resolve the parent once and create by directory descriptor, never following
    # a destination symlink or reopening it under a different parent.
    if path.name in {"", ".", ".."}:
        raise RecoveryError("目标目录无效")
    try:
        parent = path.parent.resolve(strict=True)
        fd = os.open(parent, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
        try:
            os.mkdir(path.name, 0o700, dir_fd=fd)
        finally:
            os.close(fd)
    except OSError:
        raise RecoveryError("目标目录必须是新建目录") from None
    return parent / path.name


def write_private(path: Path, content: bytes) -> None:
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "wb") as output:
        output.write(content)
        output.flush()
        os.fsync(output.fileno())


def canonical(data: object) -> bytes:
    return json.dumps(data, sort_keys=True, separators=(",", ":"), ensure_ascii=True).encode()


def identifier(value: object, pattern: str) -> str:
    if not isinstance(value, str) or not re.fullmatch(pattern, value):
        raise RecoveryError("目标标识无效")
    return value


def config_name(value: object) -> str:
    name = identifier(value, r"[a-zA-Z0-9._-]{1,128}")
    if name in {".", ".."}:
        raise RecoveryError("私有配置名称无效")
    return name


def load_descriptor(path: Path) -> dict:
    try:
        data = json.loads(private_file(path, 65536))
        if set(data) != {"version", "project", "containers", "volumes", "private_config", "revisions"} or data["version"] != 1:
            raise ValueError
        identifier(data["project"], r"[a-z0-9][a-z0-9_-]{0,62}")
        if set(data["containers"]) != set(ROLES) or set(data["volumes"]) != set(ROLES):
            raise ValueError
        for value in data["containers"].values():
            identifier(value, r"[a-f0-9]{64}")
        for value in data["volumes"].values():
            identifier(value, r"[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}")
        if len(set(data["containers"].values())) != 3 or len(set(data["volumes"].values())) != 3:
            raise ValueError
        if set(data["revisions"]) != {"backend", "web", "aiops"}:
            raise ValueError
        for revision in data["revisions"].values():
            identifier(revision, r"[a-f0-9]{40,64}")
        files = data["private_config"]
        if not isinstance(files, list) or not 1 <= len(files) <= 16:
            raise ValueError
        config_names = set()
        for item in files:
            if not isinstance(item, dict) or set(item) != {"name", "path"}:
                raise ValueError
            name = config_name(item["name"])
            if name.casefold() in config_names or not isinstance(item["path"], str) or not item["path"].startswith("/"):
                raise ValueError
            config_names.add(name.casefold())
        return data
    except (ValueError, TypeError, KeyError, json.JSONDecodeError):
        raise RecoveryError("备份描述文件无效") from None


def verify_fixture(descriptor: dict) -> None:
    project = descriptor["project"]
    if not re.fullmatch(r"cloudops-fixture-[a-f0-9]{32}", project):
        raise RecoveryError("默认只允许工具创建的隔离fixture")
    for container in descriptor["containers"].values():
        label = docker("inspect", "--format", '{{index .Config.Labels "cloudops.recovery.fixture"}}', container)
        if label != project:
            raise RecoveryError("fixture容器身份不匹配")
    for volume in descriptor["volumes"].values():
        label = docker("volume", "inspect", "--format", '{{index .Labels "cloudops.recovery.fixture"}}', volume)
        if label != project:
            raise RecoveryError("fixture卷身份不匹配")


def verify_quiescent(descriptor: dict) -> tuple[dict[str, str], dict[str, str]]:
    project = descriptor["project"]
    running = docker("ps", "-q", "--filter", f"label=com.docker.compose.project={project}")
    if running:
        raise RecoveryError("项目仍有运行中的容器，拒绝快照")
    all_ids = docker("ps", "-aq", "--no-trunc", "--filter", f"label=com.docker.compose.project={project}").splitlines()
    if not set(descriptor["containers"].values()).issubset(all_ids):
        raise RecoveryError("源项目容器集合不完整")
    service_images: dict[str, str] = {}
    for container in all_ids:
        info = docker("inspect", "--format", '{{.Id}}|{{index .Config.Labels "com.docker.compose.service"}}|{{.State.Status}}|{{.Image}}', container).split("|")
        if len(info) != 4 or info[0] != container or info[2] != "exited":
            raise RecoveryError("项目容器未全部停止")
        service = identifier(info[1], r"[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}")
        image_id = identifier(info[3], r"sha256:[a-f0-9]{64}")
        if service in service_images and service_images[service] != image_id:
            raise RecoveryError("同一服务的副本镜像不一致")
        service_images[service] = image_id
    if not {"backend", "web", "aiops", "mysql", "redis"}.issubset(service_images):
        raise RecoveryError("三仓服务或数据服务容器不完整")
    images: dict[str, str] = {}
    for role in ROLES:
        cid = descriptor["containers"][role]
        template = '{{.Id}}|{{index .Config.Labels "com.docker.compose.project"}}|{{index .Config.Labels "com.docker.compose.service"}}|{{.State.Status}}|{{.State.ExitCode}}|{{.State.OOMKilled}}|{{.Image}}'
        parts = docker("inspect", "--format", template, cid).split("|")
        if len(parts) != 7 or parts[0] != cid or parts[1] != project or parts[2] != role or parts[3] != "exited":
            raise RecoveryError("源容器身份或停写状态不匹配")
        if role in {"mysql", "redis"} and (parts[4] != "0" or parts[5] != "false"):
            raise RecoveryError("数据库或Redis未干净关闭")
        images[role] = identifier(parts[6], r"sha256:[a-f0-9]{64}")
        mounts = docker("inspect", "--format", '{{range .Mounts}}{{.Type}}|{{.Name}}|{{.Destination}};{{end}}', cid)
        expected = {"mysql": "/var/lib/mysql", "redis": "/data", "aiops": "/app/data"}[role]
        entries = [item.split("|") for item in mounts.split(";") if item]
        matches = [item for item in entries if len(item) == 3 and item[2] == expected]
        if len(matches) != 1 or matches[0][0] != "volume" or matches[0][1] != descriptor["volumes"][role]:
            raise RecoveryError("源卷与容器挂载不匹配")
        volume = descriptor["volumes"][role]
        if docker("volume", "inspect", "--format", "{{.Name}}", volume) != volume:
            raise RecoveryError("源卷身份不匹配")
        if docker("ps", "-q", "--filter", f"volume={volume}"):
            raise RecoveryError("源卷仍被运行中的容器使用")
    return images, service_images


def stream_archive(volume: str, target: Path, helper_image: str, *, mysql: bool = False) -> None:
    cmd = docker_command("run", "--rm", "--network", "none", "--user", "0", "--mount", f"type=volume,src={volume},dst=/source,readonly", "--entrypoint", "tar", helper_image, "-C", "/source", "-cf", "-", ".")
    # MySQL leaves a runtime socket symlink after clean shutdown. It is not data
    # and must never be restored; all other links remain rejected by validation.
    if mysql:
        cmd.insert(-1, "--exclude=./mysql.sock")
    try:
        process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
    except OSError:
        raise RecoveryError("无法启动备份容器") from None
    deadline = time.monotonic() + 600
    size = 0
    try:
        fd = os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
        with os.fdopen(fd, "wb") as output, selectors.DefaultSelector() as selector:
            selector.register(process.stdout, selectors.EVENT_READ)
            while True:
                remaining = deadline - time.monotonic()
                if remaining <= 0 or not selector.select(remaining):
                    raise RecoveryError("备份流超时")
                block = os.read(process.stdout.fileno(), 1024 * 1024)
                if not block:
                    break
                size += len(block)
                if size > MAX_ARCHIVE:
                    raise RecoveryError("备份超过安全上限")
                output.write(block)
            output.flush()
            os.fsync(output.fileno())
        if process.wait(timeout=5) != 0 or size == 0:
            raise RecoveryError("备份流未完成")
    finally:
        if process.poll() is None:
            process.kill()
        process.wait()
        if process.stdout is not None:
            process.stdout.close()


def digest_file(path: Path) -> dict:
    size = path.stat().st_size
    if not 0 < size <= MAX_ARCHIVE:
        raise RecoveryError("备份文件大小无效")
    hasher = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            hasher.update(chunk)
    return {"size": size, "sha256": hasher.hexdigest()}


def validate_archive(source: BinaryIO) -> None:
    seen: set[str] = set()
    total = 0
    try:
        source.seek(0)
        with tarfile.open(fileobj=source, mode="r:") as archive:
            for item in archive:
                name = item.name
                path = PurePosixPath(name)
                if path.is_absolute() or ".." in path.parts or not (item.isdir() or item.isfile()):
                    raise RecoveryError("备份包含不安全的路径或文件类型")
                normalized = str(path)
                if normalized in seen or len(seen) >= MAX_ENTRIES:
                    raise RecoveryError("备份包含重复或过多的文件")
                seen.add(normalized)
                total += item.size
                if total > MAX_ARCHIVE:
                    raise RecoveryError("备份解包大小超过安全上限")
        if not seen:
            raise RecoveryError("备份归档为空")
        source.seek(0)
    except (tarfile.TarError, OSError, ValueError):
        raise RecoveryError("备份归档无效或不完整") from None


def backup(descriptor_path: Path, key_path: Path, output: Path, allow_deployment: bool = False) -> None:
    key = load_key(key_path)
    descriptor = load_descriptor(descriptor_path)
    if not allow_deployment:
        verify_fixture(descriptor)
    images, service_images = verify_quiescent(descriptor)
    helper_image = identifier(docker("image", "inspect", "--format", "{{.Id}}", IMAGE), r"sha256:[a-f0-9]{64}")
    private = [private_file(Path(item["path"]), 1024 * 1024) for item in descriptor["private_config"]]
    output = safe_directory(output)
    write_private(output / "INCOMPLETE", b"backup incomplete\n")
    captured = datetime.now(timezone.utc).isoformat()
    entries: dict[str, dict] = {}
    for role in ROLES:
        name = role + ".tar"
        stream_archive(descriptor["volumes"][role], output / name, helper_image, mysql=role == "mysql")
        with (output / name).open("rb") as archive:
            validate_archive(archive)
        entries[name] = digest_file(output / name)
    for index, content in enumerate(private):
        name = f"private-{index}.bin"
        write_private(output / name, content)
        entries[name] = digest_file(output / name)
    # Recheck all quiescence invariants after collection. Operators must also
    # prevent external writers from restarting throughout this interval.
    if verify_quiescent(descriptor) != (images, service_images):
        raise RecoveryError("源容器在备份期间发生变化")
    manifest = {"version": 1, "source_kind": "deployment" if allow_deployment else "fixture", "captured_at": captured, "images": images, "service_images": service_images, "helper_image": helper_image, "revisions": descriptor["revisions"], "files": entries,
                "private_names": [item["name"] for item in descriptor["private_config"]], "private_count": len(private), "scope": ["mysql-physical", "redis-all-databases-aof", "aiops-data", "private-config"],
                "external_effects_reconciled": False}
    payload = canonical(manifest)
    write_private(output / "manifest.json", payload)
    write_private(output / "manifest.hmac", hmac.new(key, payload, hashlib.sha256).hexdigest().encode())
    (output / "INCOMPLETE").unlink()


def verified_copy(bundle: Path, name: str, spec: dict) -> BinaryIO:
    identifier(name, r"(?:mysql|redis|aiops)\.tar|private-[0-9]+\.bin")
    copied = None
    try:
        fd = os.open(bundle / name, os.O_RDONLY | os.O_NOFOLLOW)
        with os.fdopen(fd, "rb") as source:
            meta = os.fstat(source.fileno())
            if not stat.S_ISREG(meta.st_mode) or meta.st_uid != os.getuid() or meta.st_mode & 0o077 or meta.st_size != spec["size"] or not 0 < meta.st_size <= (1024 * 1024 if name.startswith("private-") else MAX_ARCHIVE):
                raise RecoveryError("备份文件大小或类型不匹配")
            copied = tempfile.TemporaryFile(buffering=0)
            hasher = hashlib.sha256()
            while chunk := source.read(1024 * 1024):
                copied.write(chunk)
                hasher.update(chunk)
        if not hmac.compare_digest(hasher.hexdigest(), spec["sha256"]):
            raise RecoveryError("备份文件校验失败")
        copied.seek(0)
        result, copied = copied, None
        return result
    except (OSError, KeyError, TypeError):
        raise RecoveryError("无法安全读取备份") from None
    finally:
        if copied is not None:
            copied.close()


def load_manifest(bundle: Path, key: bytes) -> dict:
    if not bundle.is_dir() or bundle.is_symlink() or (bundle / "INCOMPLETE").exists():
        raise RecoveryError("备份目录无效或未完成")
    payload = private_file(bundle / "manifest.json", 65536)
    signature = private_file(bundle / "manifest.hmac", 128).decode("ascii", errors="strict")
    if not hmac.compare_digest(signature, hmac.new(key, payload, hashlib.sha256).hexdigest()):
        raise RecoveryError("备份完整性认证失败")
    try:
        manifest = json.loads(payload)
        count = manifest["private_count"]
        if manifest["version"] != 1 or not isinstance(count, int) or not 1 <= count <= 16:
            raise ValueError
        names = manifest["private_names"]
        if not isinstance(names, list) or len(names) != count or any(not isinstance(name, str) for name in names) or len({name.casefold() for name in names}) != count:
            raise ValueError
        for name in names:
            config_name(name)
        expected = {*(role + ".tar" for role in ROLES), *(f"private-{i}.bin" for i in range(count))}
        if set(manifest["files"]) != expected or set(manifest["images"]) != set(ROLES) or set(manifest["revisions"]) != {"backend", "web", "aiops"}:
            raise ValueError
        if not {"backend", "web", "aiops", "mysql", "redis"}.issubset(manifest["service_images"]):
            raise ValueError
        for service, image in manifest["service_images"].items():
            identifier(service, r"[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}")
            identifier(image, r"sha256:[a-f0-9]{64}")
        identifier(manifest["helper_image"], r"sha256:[a-f0-9]{64}")
        for image in manifest["images"].values():
            identifier(image, r"sha256:[a-f0-9]{64}")
        for revision in manifest["revisions"].values():
            identifier(revision, r"[a-f0-9]{40,64}")
        return manifest
    except (KeyError, TypeError, ValueError, json.JSONDecodeError):
        raise RecoveryError("备份清单无效") from None


def restore(bundle: Path, key_path: Path, output: Path, allow_deployment: bool = False) -> dict:
    key = load_key(key_path)
    manifest = load_manifest(bundle, key)
    if manifest.get("source_kind") not in {"fixture", "deployment"} or (manifest["source_kind"] == "deployment" and not allow_deployment):
        raise RecoveryError("默认只允许隔离fixture备份")
    output = safe_directory(output)
    write_private(output / "INCOMPLETE", b"restore incomplete; inspect receipt\n")
    copies: dict[str, BinaryIO] = {}
    started = time.monotonic()
    restore_id = secrets.token_hex(16)
    volumes = {role: f"cloudops-restore-{restore_id}-{role}" for role in ROLES}
    receipt = {"restore_id": restore_id, "volumes": volumes, "source_manifest_sha256": hashlib.sha256(canonical(manifest)).hexdigest(), "state": "preparing"}
    write_private(output / "receipt.json", canonical(receipt))
    try:
        for name, spec in manifest["files"].items():
            copies[name] = verified_copy(bundle, name, spec)
            if name.endswith(".tar"):
                validate_archive(copies[name])
        for image in {manifest["helper_image"], *manifest["service_images"].values()}:
            # Require the exact saved image ID to exist locally before any restore.
            if docker("image", "inspect", "--format", "{{.Id}}", image) != image:
                raise RecoveryError("恢复镜像与备份不匹配")
        private_dir = output / "private"
        private_dir.mkdir(mode=0o700)
        for index, name in enumerate(manifest["private_names"]):
            write_private(private_dir / name, copies[f"private-{index}.bin"].read())
        for role in ROLES:
            name = volumes[role]
            result = subprocess.run(docker_command("volume", "inspect", name), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False)
            if result.returncode == 0:
                raise RecoveryError("新卷名称已存在，拒绝覆盖")
            if docker("volume", "create", "--label", f"cloudops.restore.id={restore_id}", "--label", f"cloudops.restore.role={role}", name) != name:
                raise RecoveryError("创建新卷失败")
            identity = docker("volume", "inspect", "--format", '{{.Name}}|{{index .Labels "cloudops.restore.id"}}|{{index .Labels "cloudops.restore.role"}}', name)
            if identity != f"{name}|{restore_id}|{role}" or docker("ps", "-q", "--filter", f"volume={name}"):
                raise RecoveryError("新卷身份或使用状态无效")
            docker("run", "--rm", "--network", "none", "--user", "0", "--mount", f"type=volume,src={name},dst=/target,readonly", "--entrypoint", "sh", manifest["helper_image"], "-c", 'test -z "$(ls -A /target)"')
            copies[role + ".tar"].seek(0)
            cmd = docker_command("run", "--rm", "-i", "--network", "none", "--user", "0", "--mount", f"type=volume,src={name},dst=/target", "--entrypoint", "tar", manifest["helper_image"], "--numeric-owner", "-C", "/target", "-xf", "-")
            try:
                result = subprocess.run(cmd, stdin=copies[role + ".tar"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=600, check=False)
            except (OSError, subprocess.TimeoutExpired):
                raise RecoveryError("恢复容器失败或超时") from None
            if result.returncode:
                raise RecoveryError("恢复归档失败，目标卷已隔离保留")
        receipt["state"] = "restored_quarantined"
        receipt["data_copy_seconds"] = round(time.monotonic() - started, 3)
        write_private(output / "receipt.complete.json", canonical(receipt))
        (output / "INCOMPLETE").unlink()
        return receipt
    finally:
        for file in copies.values():
            file.close()


def main() -> None:
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="action", required=True)
    save = sub.add_parser("backup")
    save.add_argument("--source", type=Path, required=True)
    save.add_argument("--key-file", type=Path, required=True)
    save.add_argument("--output", type=Path, required=True)
    save.add_argument("--confirm-stopped", action="store_true")
    recover = sub.add_parser("restore")
    recover.add_argument("--bundle", type=Path, required=True)
    recover.add_argument("--key-file", type=Path, required=True)
    recover.add_argument("--output", type=Path, required=True)
    recover.add_argument("--confirm-quarantine", action="store_true")
    for command_parser in (save, recover):
        command_parser.add_argument("--execute", action="store_true", help="实际创建备份或新恢复卷；默认只展示计划")
        command_parser.add_argument("--docker-socket", type=Path, help="本机 Docker Unix socket；执行时必填")
        command_parser.add_argument("--allow-deployment", action="store_true", help="受权操作者显式选择真实部署；默认只允许隔离fixture")
    args = parser.parse_args()
    if not args.execute:
        print(f"DRY RUN: {args.action}；没有读取源数据、连接服务或创建资源")
        return
    try:
        configure_docker(args.docker_socket)
        if args.action == "backup":
            if not args.confirm_stopped:
                raise RecoveryError("需要确认应用、任务和外部写入均已停止")
            backup(args.source, args.key_file, args.output, args.allow_deployment)
            print("PASS 备份已认证；外部副作用尚未核对")
        else:
            if not args.confirm_quarantine:
                raise RecoveryError("需要确认恢复后保持隔离，不启动工作进程")
            receipt = restore(args.bundle, args.key_file, args.output, args.allow_deployment)
            print(f"PASS 新卷已恢复并隔离；数据复制耗时 {receipt['data_copy_seconds']} 秒，未验收服务RTO")
    except (RecoveryError, OSError, UnicodeError, ValueError):
        parser.exit(1, "FAIL 操作未完成；私有证据和新卷已保留，不输出凭据或原始错误\n")


if __name__ == "__main__":
    main()
