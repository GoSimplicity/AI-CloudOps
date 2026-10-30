#!/usr/bin/env python3
"""Owned, local-only E04 physical backup -> upgrade -> old-model restore proof."""
from __future__ import annotations

import argparse
import hashlib
import io
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import stat
import subprocess
import sys
import tarfile
import tempfile
import time

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts"))
import recovery_backup as recovery

SOURCE = "a8f1700675bef0fe950bce5018893b9a2366f3b4"
SOCKET = Path("/var/run/docker.sock")
OWNER_LABEL = "cloudops.known.rollback.owner"
MYSQL = "mysql:8.4.11"
REDIS = "redis:7.4.11-alpine"
PHASES = ("prepare-offline-tools", "create-owned-fixture", "seed-source-49", "check-source-old-model", "quiesce-backup", "upgrade-through-cli", "restore-quarantine", "check-restored-old-model", "cleanup-owned")


class RollbackError(RuntimeError):
    pass


class SafeParser(argparse.ArgumentParser):
    def error(self, message):
        raise RollbackError("invalid-options")


def local_environment() -> dict[str, str]:
    if any(key.startswith("DOCKER_") and value for key, value in os.environ.items()):
        raise RollbackError("docker-environment-override-forbidden")
    return {key: value for key, value in os.environ.items() if key in {"PATH", "HOME", "TMPDIR", "TMP", "TEMP", "LANG", "LC_ALL", "LC_CTYPE", "TZ"}}


def source_statements(snapshot: dict) -> list[str]:
    if snapshot.get("source_revision") != SOURCE or len(snapshot.get("tables", {})) != 49:
        raise RollbackError("source-snapshot-mismatch")
    creates, constraints = [], []
    for name, ddl in sorted(snapshot["tables"].items()):
        if not re.fullmatch(r"[a-zA-Z0-9_]+", name) or not ddl.startswith(f"CREATE TABLE `{name}` (\n"):
            raise RollbackError("source-ddl-format")
        lines = ddl.splitlines()
        kept = [lines[0]]
        for line in lines[1:-1]:
            if line.lstrip().startswith("CONSTRAINT "):
                constraints.append(f"ALTER TABLE `{name}` ADD " + line.strip().removesuffix(","))
            else:
                kept.append(line)
        kept[-1] = kept[-1].removesuffix(",")
        creates.append("\n".join([*kept, lines[-1]]))
    return creates + constraints


def unpack_old_archive(archive: bytes, target: Path, provenance: dict) -> None:
    with tarfile.open(fileobj=io.BytesIO(archive), mode="r:") as source:
        members = source.getmembers()
        for member in members:
            parts = Path(member.name).parts
            if member.name.startswith("/") or ".." in parts or not (member.isfile() or member.isdir()):
                raise RollbackError("unsafe-old-archive")
        source.extractall(target, members=members, filter="data")
    for name, digest in provenance.items():
        if name not in {"go.mod", "go.sum", "pkg/di/init.go"} and not re.fullmatch(r"internal/model/[A-Za-z0-9_]+\.go", name):
            raise RollbackError("invalid-source-provenance-path")
        path = target / name
        if path.is_symlink() or not path.is_file() or hashlib.sha256(path.read_bytes()).hexdigest() != digest:
            raise RollbackError("old-source-provenance-mismatch")


CIPHER_SOURCE = r'''package main
import("crypto/aes";"crypto/cipher";"crypto/rand";"encoding/base64";"encoding/json";"os")
func main(){key,e:=os.ReadFile(os.Args[1]);if e!=nil||len(key)!=32{os.Exit(2)};block,e:=aes.NewCipher(key);if e!=nil{os.Exit(2)};g,e:=cipher.NewGCM(block);if e!=nil{os.Exit(2)};out:=map[string]string{};for _,v:=range []string{"access","secret","password"}{n:=make([]byte,g.NonceSize());if _,e=rand.Read(n);e!=nil{os.Exit(2)};out[v]=base64.StdEncoding.EncodeToString(g.Seal(n,n,[]byte("fixture-"+v),nil))};if json.NewEncoder(os.Stdout).Encode(out)!=nil{os.Exit(2)}}
'''

# Deliberately only old models + GORM/SQL reads. No old DI, table initializer,
# application entry point, scheduler, provider, notification or worker is used.
OLD_CHECKER = r'''package main
import("context";"crypto/aes";"crypto/cipher";"crypto/sha256";"database/sql";"encoding/base64";"encoding/hex";"encoding/json";"fmt";"os";"regexp";"sort";"strings";"time";"github.com/GoSimplicity/AI-CloudOps/internal/model";"gorm.io/driver/mysql";"gorm.io/gorm";"gorm.io/gorm/logger")
var phase="arguments"
func stop(){fmt.Fprintln(os.Stderr,"old-model-read-check-failed:"+phase);os.Exit(2)}
func norm(s string)string{lines:=strings.Split(s,"\n");if len(lines)<3{stop()};var cols,keys []string;for _,line:=range lines[1:len(lines)-1]{line=strings.TrimSuffix(line,",");if strings.HasPrefix(strings.TrimSpace(line),"`"){cols=append(cols,line)}else{keys=append(keys,line)}};sort.Strings(keys);last:=regexp.MustCompile(` AUTO_INCREMENT=[0-9]+`).ReplaceAllString(lines[len(lines)-1],"");return lines[0]+"\n"+strings.Join(append(cols,keys...),",\n")+"\n"+last}
func plain(value string,key []byte)string{data,e:=base64.StdEncoding.DecodeString(value);if e!=nil{stop()};block,e:=aes.NewCipher(key);if e!=nil{stop()};g,e:=cipher.NewGCM(block);if e!=nil||len(data)<g.NonceSize(){stop()};p,e:=g.Open(nil,data[:g.NonceSize()],data[g.NonceSize():],nil);if e!=nil{stop()};return string(p)}
func main(){if len(os.Args)!=4{stop()};dsn,e:=os.ReadFile(os.Args[1]);if e!=nil{stop()};raw,e:=os.ReadFile(os.Args[2]);if e!=nil{stop()};key,e:=os.ReadFile(os.Args[3]);if e!=nil||len(key)!=32{stop()};var snapshot struct{Tables map[string]string `json:"tables"`};if json.Unmarshal(raw,&snapshot)!=nil||len(snapshot.Tables)!=49{stop()};phase="database-open";pool,e:=sql.Open("mysql",strings.TrimSpace(string(dsn)));if e!=nil{stop()};defer pool.Close();ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second);defer cancel();tx,e:=pool.BeginTx(ctx,&sql.TxOptions{ReadOnly:true});if e!=nil{stop()};defer tx.Rollback();db,e:=gorm.Open(mysql.New(mysql.Config{Conn:tx,SkipInitializeWithVersion:true}),&gorm.Config{DisableAutomaticPing:true,Logger:logger.Discard});if e!=nil{stop()};phase="inventory";var names []string;rows,e:=tx.QueryContext(ctx,"SELECT table_name FROM information_schema.tables WHERE table_schema=DATABASE() ORDER BY table_name");if e!=nil{stop()};for rows.Next(){var name string;if rows.Scan(&name)!=nil{stop()};names=append(names,name)};if rows.Err()!=nil{stop()};rows.Close();if len(names)!=49{stop()};digest:=sha256.New();total:=0;for _,name:=range names{phase="table-ddl";expected,ok:=snapshot.Tables[name];if !ok||!regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(name){stop()};var returned,ddl string;if tx.QueryRowContext(ctx,"SHOW CREATE TABLE `"+name+"`").Scan(&returned,&ddl)!=nil||returned!=name||norm(ddl)!=norm(expected){stop()};phase="row-digest";data,e:=tx.QueryContext(ctx,"SELECT * FROM `"+name+"`");if e!=nil{stop()};columns,e:=data.Columns();if e!=nil{stop()};var values []string;for data.Next(){line:=make([]any,len(columns));ptr:=make([]any,len(columns));for i:=range line{ptr[i]=&line[i]};if data.Scan(ptr...)!=nil{stop()};b,e:=json.Marshal(line);if e!=nil{stop()};values=append(values,string(b));total++};if data.Err()!=nil{stop()};data.Close();sort.Strings(values);b,e:=json.Marshal([]any{name,columns,values});if e!=nil{stop()};digest.Write(b)};phase="account-model-cipher";var account model.CloudAccount;if db.Unscoped().First(&account,21).Error!=nil||plain(account.AccessKey,key)!="fixture-access"||plain(account.SecretKey,key)!="fixture-secret"{stop()};phase="resource-model-cipher";var resource model.TreeCloudResource;if db.Unscoped().First(&resource,41).Error!=nil||plain(resource.Password,key)!="fixture-password"{stop()};phase="soft-deleted-notification-model";var notification model.WorkorderNotification;if db.Unscoped().First(&notification,62).Error!=nil||int(notification.IsDefault)!=1{stop()};phase="legacy-queue-model";var queue model.WorkorderNotificationQueue;if db.Unscoped().First(&queue,71).Error!=nil{stop()};phase="null-audit";var nulls bool;if tx.QueryRowContext(ctx,"SELECT old_value IS NULL AND new_value IS NULL FROM cl_tree_cloud_resource_change_log WHERE id=52").Scan(&nulls)!=nil||!nulls{stop()};out:=map[string]any{"tables":49,"rows":total,"data_sha256":hex.EncodeToString(digest.Sum(nil)),"old_models_read":true,"synthetic_cipher_valid":true,"null_and_soft_deleted_rows":true};if json.NewEncoder(os.Stdout).Encode(out)!=nil{stop()}}
'''


class Runner:
    def __init__(self, go: str | None = None):
        self.environment = local_environment()
        self.go = go or shutil.which("go")
        self.project = "cloudops-fixture-" + secrets.token_hex(16)
        self.owner = secrets.token_hex(16)
        self.directory: Path | None = None
        self.phase = "prepare-offline-tools"
        self.sequence = 0
        self.containers: set[str] = set()
        self.volumes: set[str] = set()
        self.original_command = recovery.docker_command
        self.original_image = recovery.IMAGE
        self.images: dict[str, str] = {}

    def labels(self, role: str) -> list[str]:
        return ["--label", f"{OWNER_LABEL}={self.owner}", "--label", f"cloudops.recovery.fixture={self.project}", "--label", f"com.docker.compose.project={self.project}", "--label", f"com.docker.compose.service={role}"]

    def docker_command(self, *arguments: str) -> list[str]:
        parts = list(arguments)
        expected_labels = {OWNER_LABEL: self.owner, "cloudops.recovery.fixture": self.project, "com.docker.compose.project": self.project}
        for index, part in enumerate(parts[:-1]):
            if part == "--label":
                key, separator, value = parts[index + 1].partition("=")
                if key in expected_labels and (not separator or value != expected_labels[key]):
                    raise RollbackError("foreign-resource-label")
        if parts and parts[0] == "run":
            if "--network" not in parts or parts[parts.index("--network") + 1] != "none":
                raise RollbackError("fixture-network-isolation-required")
            self.sequence += 1
            if "--name" in parts:
                name = parts[parts.index("--name") + 1]
                if not re.fullmatch(re.escape(self.project) + r"-[a-z0-9-]+", name):
                    raise RollbackError("foreign-container-name")
            else:
                name = f"{self.project}-helper-{self.sequence}"
                parts[1:1] = ["--name", name]
            self.containers.add(name)
            if "--pull=always" in parts or "--pull=missing" in parts:
                raise RollbackError("fixture-image-pull-forbidden")
            if "--pull=never" not in parts:
                parts[1:1] = ["--pull=never"]
            parts[1:1] = self.labels("helper")
            # The published helper image declares VOLUME /data. Cover it so
            # archive helpers and stopped fake apps cannot leak anonymous volumes.
            if not any(re.search(r"(?:^|,)(?:dst|target|destination)=/data(?:,|$)", part) for part in parts):
                parts[1:1] = ["--tmpfs", "/data:rw,noexec,nosuid"]
        if parts[:2] == ["volume", "create"]:
            name = parts[-1]
            if not (name.startswith(self.project + "-") or re.fullmatch(r"cloudops-restore-[a-f0-9]{32}-(mysql|redis|aiops)", name)):
                raise RollbackError("foreign-volume-name")
            self.volumes.add(name)
            parts[2:2] = ["--label", f"{OWNER_LABEL}={self.owner}", "--label", f"cloudops.recovery.fixture={self.project}", "--label", f"com.docker.compose.project={self.project}"]
        return ["docker", "--host", "unix:///var/run/docker.sock", *parts]

    def command(self, *parts: str, data: bytes | None = None, timeout: int = 120) -> bytes:
        if data is not None and parts and parts[0] in {"run", "exec"} and "-i" not in parts:
            raise RollbackError("fixture-stdin-forwarding-required")
        try:
            result = subprocess.run(self.docker_command(*parts), input=data, env=self.environment, capture_output=True, timeout=timeout)
        except (OSError, subprocess.TimeoutExpired):
            raise RollbackError("owned-docker-operation-failed") from None
        if result.returncode:
            raise RollbackError("owned-docker-operation-failed")
        return result.stdout

    def owned(self, kind: str, name: str) -> None:
        if name not in (self.containers if kind == "container" else self.volumes):
            raise RollbackError("untracked-resource")
        args = ("inspect",) if kind == "container" else ("volume", "inspect")
        prefix = ".Config.Labels" if kind == "container" else ".Labels"
        template = "|".join('{{index ' + prefix + ' "' + key + '"}}' for key in (OWNER_LABEL, "com.docker.compose.project", "cloudops.recovery.fixture"))
        if self.command(*args, "--format", template, name).decode().strip() != "|".join((self.owner, self.project, self.project)):
            raise RollbackError("foreign-resource-owner")

    def private(self, name: str, data: bytes) -> Path:
        path = self.directory / name
        recovery.write_private(path, data)
        return path

    def local(self, args: list[str], *, cwd: Path = ROOT, env: dict | None = None, timeout: int = 600) -> bytes:
        try:
            result = subprocess.run(args, cwd=cwd, env=env or self.environment, capture_output=True, timeout=timeout)
        except (OSError, subprocess.TimeoutExpired):
            raise RollbackError("offline-build-or-source-failed") from None
        if result.returncode:
            raise RollbackError("offline-build-or-source-failed")
        return result.stdout

    def compile(self, source: Path, target: Path, architecture: str, package: str) -> None:
        if not self.go:
            raise RollbackError("go-toolchain-required")
        environment = {**self.environment, "GOTOOLCHAIN": "local", "GOPROXY": "off", "GOSUMDB": "off", "GOOS": "linux", "GOARCH": architecture, "CGO_ENABLED": "0"}
        # Derive GOROOT from this explicitly selected tool, not another ambient
        # Go version. Module downloads and toolchain auto-switching stay disabled.
        environment["GOROOT"] = self.local([self.go, "env", "GOROOT"], env=environment).decode().strip()
        self.local([self.go, "build", "-mod=readonly", "-o", str(target), package], cwd=source, env=environment)
        target.chmod(0o700)

    def create(self, role: str, image: str, volume: str | None = None) -> str:
        name = self.project + "-" + role
        options = ["run", "--name", name, "-d", "--pull=never", "--network", "none", *self.labels(role)]
        if volume:
            self.owned("volume", volume)
            target = "/var/lib/mysql" if role in {"mysql", "restored-mysql"} else "/data" if role in {"redis", "restored-redis"} else "/app/data"
            options += ["--mount", f"type=volume,src={volume},dst={target}"]
        if role in {"mysql", "restored-mysql"}:
            options += ["-e", "MYSQL_ALLOW_EMPTY_PASSWORD=yes", image]
        elif role in {"redis", "restored-redis"}:
            options += [image, "redis-server", "--appendonly", "yes", "--appendfsync", "always"]
        else:
            options += ["--entrypoint", "sh", image, "-c", "true"]
        self.command(*options)
        self.owned("container", name)
        return name

    def ready(self, name: str, role: str) -> None:
        self.owned("container", name)
        deadline = time.monotonic() + 90
        while time.monotonic() < deadline:
            args = ("mysqladmin", "--protocol=socket", "-uroot", "ping", "--silent") if role == "mysql" else ("redis-cli", "PING")
            if role != "mysql" or self.command("exec", name, "cat", "/proc/1/comm").strip() == b"mysqld":
                try:
                    self.command("exec", name, *args, timeout=10)
                    return
                except RollbackError:
                    pass
            time.sleep(0.25)
        raise RollbackError("owned-data-readiness-failed")

    def install(self, name: str, path: Path, destination: str) -> None:
        self.owned("container", name)
        self.command("cp", str(path), f"{name}:{destination}")

    def private_inside(self, name: str, destination: str, data: bytes) -> None:
        self.owned("container", name)
        if not re.fullmatch(r"/tmp/[a-zA-Z0-9_.-]+", destination):
            raise RollbackError("invalid-private-fixture-path")
        self.command("exec", "-i", name, "sh", "-c", f"umask 077; cat > {destination}; chmod 600 {destination}", data=data)

    def sql(self, name: str, statements: str) -> bytes:
        self.owned("container", name)
        return self.command("exec", "-i", name, "mysql", "--protocol=socket", "--default-character-set=utf8mb4", "-uroot", "--batch", "--raw", "--skip-column-names", data=statements.encode())

    def checker(self, name: str) -> dict:
        self.install(name, self.directory / "old-checker", "/tmp/old-checker")
        self.install(name, self.directory / "source.json", "/tmp/source.json")
        self.private_inside(name, "/tmp/source.key", (self.directory / "resource.key").read_bytes())
        self.private_inside(name, "/tmp/old.dsn", b"root@unix(/var/run/mysqld/mysqld.sock)/cloudops?charset=utf8mb4&parseTime=true&loc=UTC")
        result = json.loads(self.command("exec", name, "/tmp/old-checker", "/tmp/old.dsn", "/tmp/source.json", "/tmp/source.key"))
        if result.get("tables") != 49 or not result.get("old_models_read") or not result.get("synthetic_cipher_valid") or not result.get("null_and_soft_deleted_rows"):
            raise RollbackError("old-model-check-incomplete")
        return result

    def cleanup(self) -> None:
        # All labels are checked before the first destructive operation.
        existing_containers = self.command("ps", "-aq", "--filter", f"label={OWNER_LABEL}={self.owner}").decode().split()
        names = []
        for identifier in existing_containers:
            name = self.command("inspect", "--format", "{{.Name}}", identifier).decode().strip().removeprefix("/")
            self.owned("container", name)
            names.append(name)
        for volume in self.volumes:
            self.owned("volume", volume)
        for name in names:
            self.command("rm", "--force", "--volumes", name)
        for volume in sorted(self.volumes):
            self.command("volume", "rm", volume)

    def execute(self) -> dict:
        recovery.configure_docker(SOCKET)
        self.directory = Path(tempfile.mkdtemp(prefix="cloudops-known-rollback-"))
        self.directory.chmod(0o700)
        print("Private evidence:", self.directory, flush=True)
        self.private("fixture.json", recovery.canonical({"project": self.project, "owner": self.owner, "state": "creating-owned-fixture"}))
        recovery.docker_command = self.docker_command
        try:
            snapshot_bytes = (ROOT / "internal/dbschema/known_a8f1700.json").read_bytes()
            snapshot = json.loads(snapshot_bytes)
            statements = source_statements(snapshot)
            self.private("source.json", snapshot_bytes)
            old = self.directory / "old"; old.mkdir(mode=0o700)
            archive = self.local(["git", "archive", "--format=tar", SOURCE, "go.mod", "go.sum", "internal/model", "pkg/di/init.go"])
            unpack_old_archive(archive, old, snapshot["source_files_sha256"])
            for role, image, permitted in (("mysql", MYSQL, {"/var/lib/mysql"}), ("redis", REDIS, {"/data"})):
                image_id = self.command("image", "inspect", "--format", "{{.Id}}", image).decode().strip()
                if not re.fullmatch(r"sha256:[a-f0-9]{64}", image_id): raise RollbackError("invalid-local-image")
                declared = json.loads(self.command("image", "inspect", "--format", "{{json .Config.Volumes}}", image_id)) or {}
                if set(declared) != permitted: raise RollbackError("unexpected-image-anonymous-volume")
                self.images[role] = image_id
            recovery.IMAGE = self.images["redis"]
            architecture = self.command("image", "inspect", "--format", "{{.Architecture}}", self.images["mysql"]).decode().strip()
            if architecture not in {"arm64", "amd64"}: raise RollbackError("unsupported-local-image-architecture")
            helper = old / "cmd/rollbackcheck"; helper.mkdir(parents=True); (helper / "main.go").write_text(OLD_CHECKER)
            self.compile(old, self.directory / "old-checker", architecture, "./cmd/rollbackcheck")
            self.compile(ROOT, self.directory / "schema", architecture, "./cmd/schema")
            cipher_source = self.private("cipher.go", CIPHER_SOURCE.encode())
            self.compile(ROOT, self.directory / "cipher", architecture, str(cipher_source))
            planned = self.command("run", "--rm", "--pull=never", "--network", "none", "--mount", f'type=bind,src={self.directory / "schema"},dst=/fixture/schema,readonly', "--entrypoint", "/fixture/schema", self.images["redis"], "upgrade-a8f1700")
            fields = dict(re.findall(r"(source|source_checksum|implementation_checksum|target_checksum)=([a-f0-9]+)", planned.decode()))
            if fields.get("source") != SOURCE or fields.get("source_checksum") != hashlib.sha256(snapshot_bytes).hexdigest() or any(not re.fullmatch(r"[a-f0-9]{64}", fields.get(key, "")) for key in ("source_checksum", "implementation_checksum", "target_checksum")):
                raise RollbackError("known-upgrade-cli-not-ready")
            self.phase = "create-owned-fixture"
            by_role = {}
            for role in recovery.ROLES:
                volume = self.project + "-" + role
                self.command("volume", "create", volume); self.owned("volume", volume); by_role[role] = volume
            containers = {role: self.create(role, self.images["mysql"] if role == "mysql" else self.images["redis"], by_role[role]) for role in recovery.ROLES}
            for role in ("backend", "web"): self.create(role, self.images["redis"])
            mysql, redis = containers["mysql"], containers["redis"]
            self.ready(mysql, "mysql"); self.ready(redis, "redis")
            self.phase = "seed-source-49"
            key = secrets.token_hex(16).encode(); self.private("resource.key", key)
            self.install(mysql, self.directory / "cipher", "/tmp/cipher")
            self.private_inside(mysql, "/tmp/source.key", key)
            encrypted = json.loads(self.command("exec", mysql, "/tmp/cipher", "/tmp/source.key"))
            self.sql(mysql, "CREATE DATABASE cloudops CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci; USE cloudops;\n" + ";\n".join(statements) + ";\n" + seed_sql(encrypted))
            self.command("exec", redis, "redis-cli", "RPUSH", "fixture:queue", "pending-one", "pending-two")
            self.command("exec", redis, "redis-cli", "-n", "1", "SET", "fixture:approval", "approved-not-replayed")
            self.command("exec", redis, "redis-cli", "-n", "2", "SET", "fixture:tombstone", "do-not-replay")
            # Seed only filesystem data, never start an AIOps application.
            knowledge = b"fixture knowledge document\n"
            self.command("run", "--rm", "-i", "--pull=never", "--network", "none", "--mount", f'type=volume,src={by_role["aiops"]},dst=/app/data', "--entrypoint", "sh", self.images["redis"], "-c", "mkdir -p /app/data/knowledge; cat > /app/data/knowledge/document.txt", data=knowledge)
            seeded_knowledge = self.command("run", "--rm", "--pull=never", "--network", "none", "--mount", f'type=volume,src={by_role["aiops"]},dst=/app/data,readonly', "--entrypoint", "cat", self.images["redis"], "/app/data/knowledge/document.txt")
            if seeded_knowledge != knowledge: raise RollbackError("knowledge-seed-not-confirmed")
            self.phase = "check-source-old-model"
            before = self.checker(mysql)
            self.private("old-checker-before.json", recovery.canonical(before))
            private_config = self.private("settings.env", b"TREE_PASSWORD_ENCRYPTION_KEY=" + key + b"\nNOTIFICATION_ENABLED=false\n")
            signing = self.private("backup.key", secrets.token_bytes(32))
            revisions = {"backend": SOURCE}
            for role, folder in (("web", "AI-CloudOps-web"), ("aiops", "AI-CloudOps-aiops")):
                revision = self.local(["git", "-C", str(ROOT.parent / folder), "rev-parse", "HEAD"]).decode().strip()
                if not re.fullmatch(r"[a-f0-9]{40}", revision): raise RollbackError("source-revision-unavailable")
                revisions[role] = revision
            ids = {role: self.command("inspect", "--format", "{{.Id}}", name).decode().strip() for role, name in containers.items()}
            descriptor = {"version": 1, "project": self.project, "containers": ids, "volumes": by_role, "private_config": [{"name": ".env", "path": str(private_config)}], "revisions": revisions}
            desc = self.private("descriptor.json", recovery.canonical(descriptor))
            self.phase = "quiesce-backup"
            for name in (mysql, redis): self.owned("container", name); self.command("stop", "--time", "60", name)
            recovery.backup(desc, signing, self.directory / "bundle")
            manifest_hash = hashlib.sha256((self.directory / "bundle/manifest.json").read_bytes()).hexdigest()
            self.phase = "upgrade-through-cli"
            self.command("start", mysql); self.ready(mysql, "mysql")
            self.install(mysql, self.directory / "schema", "/tmp/schema")
            self.private_inside(mysql, "/tmp/upgrade.dsn", b"root@unix(/var/run/mysqld/mysqld.sock)/cloudops?charset=utf8mb4&parseTime=true&loc=UTC")
            self.private_inside(mysql, "/tmp/upgrade.note", b"owned physical backup/restore acceptance fixture")
            self.private_inside(mysql, "/tmp/upgrade.proof", recovery.canonical({"version": 1, "source_revision": SOURCE, "backup_sha256": manifest_hash, "backup_ready": True}))
            self.command("exec", mysql, "/tmp/schema", "upgrade-a8f1700", "--execute", "--confirm-no-writers", "--dsn-file", "/tmp/upgrade.dsn", "--upgrade-note-file", "/tmp/upgrade.note", "--backup-proof-file", "/tmp/upgrade.proof", "--notification-timezone", "UTC", "--attest-historical-time", timeout=180)
            self.command("exec", mysql, "/tmp/schema", "verify", "--dsn-file", "/tmp/upgrade.dsn")
            if self.sql(mysql, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='cloudops' AND table_name<>'cloudops_schema_history'").strip() != b"58": raise RollbackError("upgrade-target-inventory")
            self.command("stop", "--time", "60", mysql)
            self.private("upgrade-confirmed.json", recovery.canonical({"business_tables":58,"plan":fields}))
            self.phase = "restore-quarantine"
            restored = recovery.restore(self.directory / "bundle", signing, self.directory / "restored")
            if restored.get("state") != "restored_quarantined": raise RollbackError("restore-not-quarantined")
            restored_mysql = self.create("restored-mysql", self.images["mysql"], restored["volumes"]["mysql"])
            restored_redis = self.create("restored-redis", self.images["redis"], restored["volumes"]["redis"])
            self.ready(restored_mysql, "mysql"); self.ready(restored_redis, "redis")
            self.phase = "check-restored-old-model"
            after = self.checker(restored_mysql)
            self.private("old-checker-after.json", recovery.canonical(after))
            if before != after: raise RollbackError("old-data-or-schema-not-restored")
            for database, command in (("0", ("LRANGE", "fixture:queue", "0", "-1")), ("1", ("GET", "fixture:approval")), ("2", ("GET", "fixture:tombstone"))):
                actual = self.command("exec", restored_redis, "redis-cli", "-n", database, *command).strip()
                expected = {"0": b"pending-one\npending-two", "1": b"approved-not-replayed", "2": b"do-not-replay"}[database]
                if actual != expected: raise RollbackError("redis-state-not-restored")
            recovered_knowledge = self.command("run", "--rm", "--pull=never", "--network", "none", "--mount", f'type=volume,src={restored["volumes"]["aiops"]},dst=/app/data,readonly', "--entrypoint", "cat", self.images["redis"], "/app/data/knowledge/document.txt")
            if recovered_knowledge != knowledge or (self.directory / "restored/private/.env").read_bytes() != private_config.read_bytes(): raise RollbackError("knowledge-or-private-config-not-restored")
            result = {"source_revision": SOURCE, "project": self.project, "owner": self.owner, "old_checker": after, "upgraded_business_tables": 58, "backup_manifest_sha256": manifest_hash, "upgrade_plan": fields, "images": self.images, "old_checker_sha256": hashlib.sha256((self.directory / "old-checker").read_bytes()).hexdigest(), "upgrade_binary_sha256": hashlib.sha256((self.directory / "schema").read_bytes()).hexdigest(), "restore_state": restored["state"], "data_copy_seconds": restored["data_copy_seconds"], "redis_queue_and_markers": True, "knowledge_and_private_config": True, "no_application_or_worker_started": True, "whole_stack_rto_verified": False, "external_effects_restored": False}
            self.private("result.json", recovery.canonical(result))
            self.phase = "cleanup-owned"; self.cleanup()
            self.private("cleanup.complete.json", recovery.canonical({"state":"owned-resources-removed","project":self.project,"owner":self.owner}))
            print("PASS owned E04 physical backup/upgrade/old-model restore; no whole-stack RTO or external-effect claim")
            return result
        finally:
            recovery.docker_command = self.original_command
            recovery.IMAGE = self.original_image


def seed_sql(encrypted: dict[str, str]) -> str:
    literal = lambda value: "X'" + value.encode().hex() + "'"
    queries = [
        "INSERT INTO cl_system_departments(id,name,parent_id,created_at,deleted_at) VALUES(11,'fixture',0,'2020-01-02 03:04:05',0)",
        "INSERT INTO cl_system_users(id,department_id,username,password,created_at,deleted_at) VALUES(12,11,'fixture','not-auth','2020-01-02 03:04:05',0)",
        "INSERT INTO cl_tree_cloud_account(id,name,provider,access_key,secret_key,status,create_user_id,create_user_name,created_at,deleted_at) VALUES(21,'fixture',1," + literal(encrypted["access"]) + "," + literal(encrypted["secret"]) + ",2,12,'fixture','2020-01-02 03:04:05',0)",
        "INSERT INTO cl_tree_cloud_account_region(id,cloud_account_id,region,region_name,status,created_at,deleted_at) VALUES(31,21,'fixture','fixture',2,'2020-01-02 03:04:05',0)",
        "INSERT INTO cl_tree_cloud_resource(id,name,cloud_account_id,cloud_account_region_id,password,created_at,deleted_at) VALUES(41,'fixture',21,31," + literal(encrypted["password"]) + ",'2020-01-02 03:04:05',0)",
        "INSERT INTO cl_tree_cloud_resource_sync_history(id,cloud_account_id,sync_mode,sync_status,created_at,deleted_at) VALUES(42,21,'incremental','success','2020-01-02 03:04:05',0)",
        "INSERT INTO cl_tree_cloud_resource_change_log(id,resource_id,field_name,old_value,new_value,created_at,deleted_at) VALUES(51,41,' password ','fixture-old','fixture-new','2020-01-02 03:04:05',123),(52,41,'key',NULL,NULL,'2020-01-02 03:04:05',0)",
        "INSERT INTO cl_workorder_notification(id,name,event_types,channels,recipient_types,message_template,operator_id,is_default,status,created_at,deleted_at) VALUES(61,'fixture','[]','[]','[]','fixture',12,0,2,'2020-01-02 03:04:05',0)",
        "INSERT INTO cl_workorder_notification(id,name,event_types,channels,recipient_types,message_template,operator_id,is_default,status,trigger_type,scheduled_time,created_at,deleted_at) VALUES(62,'fixture deleted','[]','[]','[]','fixture',12,1,2,'scheduled','2030-01-02 03:04:05','2020-01-02 03:04:05',123)",
        "INSERT INTO cl_workorder_notification_queue(id,notification_id,event_type,channel,recipient_type,recipient_id,recipient_addr,content,status,scheduled_at,created_at,deleted_at) VALUES(71,61,'instance_created','email','custom','fixture','fixture@example.test','must not replay',2,'2030-01-02 03:04:05','2020-01-02 03:04:05',0)",
        "INSERT INTO cl_cron_jobs(id,name,schedule,status,created_at,deleted_at) VALUES(81,'disabled fixture','0 * * * *',2,'2020-01-02 03:04:05',0)",
    ]
    return ";\n".join(queries) + ";\n"


def main(argv: list[str] | None = None) -> int:
    parser = SafeParser(description=__doc__)
    parser.add_argument("--execute", action="store_true")
    parser.add_argument("--go", help="explicit local Go toolchain; downloads remain disabled")
    runner = None
    try:
        args = parser.parse_args(argv)
        if not args.execute:
            print("DRY RUN: owned fixture only; no Docker, credentials, backup, upgrade or restore executed")
            return 0
        os.umask(0o077)
        runner = Runner(args.go)
        runner.execute()
        return 0
    except (RollbackError, recovery.RecoveryError, OSError, ValueError, KeyError, TypeError, KeyboardInterrupt, tarfile.TarError):
        if runner and runner.directory:
            recovery.write_private(runner.directory / "failure.json", recovery.canonical({"phase": runner.phase, "state": "failed-evidence-retained"}))
            print("FAIL owned rollback fixture; phase=" + runner.phase + "; private evidence=" + str(runner.directory), file=sys.stderr)
        else:
            print("FAIL fixture preflight; no supplied values or private output shown", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
