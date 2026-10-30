#!/usr/bin/env python3
"""Prepare exact committed sibling sources offline; never build or deploy them."""
from __future__ import annotations

import argparse
from dataclasses import dataclass
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import signal
import stat
import subprocess
import sys
import tarfile
import tempfile
import unicodedata
from urllib.parse import unquote, urlsplit

ROOT = Path(__file__).resolve().parents[1]
REPOSITORIES = {
    "backend": ("AI-CloudOps", "GoSimplicity/AI-CloudOps"),
    "web": ("AI-CloudOps-web", "GoSimplicity/AI-CloudOps-web"),
    "aiops": ("AI-CloudOps-aiops", "GoSimplicity/AI-CloudOps-aiops"),
}
PUBLIC_ENV_PATHS = {
    "web": {"apps/web-antd/.env", "apps/web-antd/.env.analyze", "apps/web-antd/.env.development", "apps/web-antd/.env.production"},
}
MAX_FILES = 100_000
MAX_FILE_BYTES = 64 * 1024**2
MAX_SOURCE_BYTES = 512 * 1024**2
MAX_ARCHIVE_BYTES = 1024**3
MAX_GIT_OUTPUT = 16 * 1024**2


class CandidateError(RuntimeError):
    """Fixed diagnostics only; never embed Git output, paths or remote URLs."""


class SafeParser(argparse.ArgumentParser):
    def error(self, message: str) -> None:
        raise CandidateError("invalid-options")


@dataclass(frozen=True)
class Blob:
    oid: str
    size: int
    executable: bool


@dataclass(frozen=True)
class Source:
    role: str
    folder: str
    repository: str
    path: Path
    revision: str
    objects: Path
    entries: dict[str, Blob]


def git_environment(extra: dict[str, str] | None = None) -> dict[str, str]:
    allowed = {"PATH", "HOME", "USER", "LOGNAME", "TMPDIR", "TMP", "TEMP"}
    result = {key: value for key, value in os.environ.items() if key in allowed}
    result.update({"LC_ALL": "C", "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": os.devnull,
                   "GIT_ATTR_NOSYSTEM": "1", "GIT_NO_REPLACE_OBJECTS": "1", "GIT_NO_LAZY_FETCH": "1",
                   "GIT_TERMINAL_PROMPT": "0", "GIT_PROTOCOL_FROM_USER": "0"})
    result.update(extra or {})
    return result


def git(repo: Path, arguments: list[str], *, environment: dict[str, str] | None = None,
        stdout=None, allowed_codes: tuple[int, ...] = (0,)) -> bytes:
    command = ["git", "--no-pager", "--no-optional-locks", "--no-replace-objects", "-C", str(repo),
               "-c", "protocol.allow=never", "-c", "core.fsmonitor=false", "-c", "core.hooksPath=" + os.devnull,
               "-c", "core.attributesFile=" + os.devnull, "-c", "gc.auto=0", "-c", "maintenance.auto=false", *arguments]
    process = None
    try:
        process = subprocess.Popen(command, env=git_environment(environment), stdin=subprocess.DEVNULL,
                                   stdout=subprocess.PIPE if stdout is None else stdout, stderr=subprocess.PIPE,
                                   start_new_session=True)
        output, _ = process.communicate(timeout=120)
    except (OSError, subprocess.TimeoutExpired):
        if process is not None:
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            process.communicate()
        raise CandidateError("git-unavailable-or-timed-out") from None
    if process.returncode not in allowed_codes or len(output or b"") > MAX_GIT_OUTPUT:
        raise CandidateError("git-operation-failed")
    return output or b""


def expected_remote(value: str, repository: str) -> bool:
    # Authentication material may be present in local origin config. Parse only
    # the known host/path identity; do not persist or print the original string.
    scp = re.fullmatch(r"(?:[^@/]+@)?github\.com:([^?#]+)(?:[?#].*)?", value, flags=re.IGNORECASE)
    if scp:
        path = scp.group(1)
    else:
        parsed = urlsplit(value)
        if parsed.scheme not in {"https", "ssh"} or parsed.hostname != "github.com":
            return False
        path = parsed.path.lstrip("/")
    path = unquote(path).rstrip("/")
    if path.lower().endswith(".git"):
        path = path[:-4]
    return path.casefold() == repository.casefold()


def relative_source_path(value: str, role: str) -> str:
    if (not value or value.startswith("/") or "\\" in value or len(value.encode("utf-8")) > 4096
            or any(ord(character) < 32 or ord(character) == 127 for character in value)
            or re.match(r"^[A-Za-z]:", value)):
        raise CandidateError("unsafe-source-path")
    parts = value.split("/")
    if any(part in {"", ".", ".."} for part in parts):
        raise CandidateError("unsafe-source-path")
    if any(part.casefold() in {".git", ".cloudops-fresh", "backups.local", "manifest.local"} for part in parts):
        raise CandidateError("private-or-control-path")
    basename = parts[-1].casefold()
    if (basename == ".env" or basename.startswith(".env.")) and value not in PUBLIC_ENV_PATHS.get(role, set()):
        raise CandidateError("personal-environment-must-not-be-tracked")
    return value


def canonical_path(value: str) -> str:
    return unicodedata.normalize("NFD", value).casefold()


def tracked_entries(repo: Path, revision: str, role: str) -> dict[str, Blob]:
    raw = git(repo, ["ls-tree", "-r", "-l", "-z", revision])
    entries: dict[str, Blob] = {}
    canonical: set[str] = set()
    total = 0
    for row in raw.split(b"\0"):
        if not row:
            continue
        header, path_bytes = row.split(b"\t", 1)
        mode, kind, oid, size = header.decode("ascii").split()
        path = relative_source_path(path_bytes.decode("utf-8"), role)
        if mode not in {"100644", "100755"} or kind != "blob" or not re.fullmatch(r"[0-9a-f]{40}", oid):
            raise CandidateError("links-or-submodules-not-supported")
        length = int(size)
        total += length
        key = canonical_path(path)
        if length < 0 or length > MAX_FILE_BYTES or total > MAX_SOURCE_BYTES or key in canonical:
            raise CandidateError("source-tree-limit-or-collision")
        canonical.add(key)
        entries[path] = Blob(oid, length, mode == "100755")
        if len(entries) > MAX_FILES:
            raise CandidateError("source-tree-limit")
    if not entries:
        raise CandidateError("empty-source-tree")
    for path in entries:
        for parent in PurePosixPath(path).parents:
            if str(parent) != "." and canonical_path(str(parent)) in canonical:
                raise CandidateError("source-file-directory-collision")
    return entries


def collect_sources(root: Path) -> list[Source]:
    sources: list[Source] = []
    for role, (folder, repository) in REPOSITORIES.items():
        path = root.parent / folder
        if path.is_symlink() or not path.is_dir():
            raise CandidateError("fixed-sibling-repository-required")
        path = path.resolve(strict=True)
        if any(os.path.lexists(parent / ".git") for parent in path.parents):
            raise CandidateError("nested-source-repository-refused")
        top = Path(git(path, ["rev-parse", "--show-toplevel"]).decode().strip()).resolve(strict=True)
        if top != path:
            raise CandidateError("nested-checkout-not-supported")
        # Refuse locally configured execution helpers before invoking status.
        helpers = git(path, ["config", "--local", "--includes", "--name-only", "--get-regexp",
                             r"^(filter\..*\.(clean|process)|tar\..*\.command)$"], allowed_codes=(0, 1))
        if helpers.strip():
            raise CandidateError("local-execution-helpers-not-supported")
        remote = git(path, ["remote", "get-url", "origin"]).decode().strip()
        if not expected_remote(remote, repository):
            raise CandidateError("unexpected-origin-repository")
        index_flags = git(path, ["ls-files", "-v", "-z"])
        if any(row[:1] != b"H" for row in index_flags.split(b"\0") if row):
            raise CandidateError("hidden-or-sparse-index-state-refused")
        if git(path, ["status", "--porcelain=v1", "-z", "--untracked-files=all", "--ignore-submodules=all"]).strip():
            raise CandidateError("uncommitted-work-refused")
        revision = git(path, ["rev-parse", "--verify", "HEAD^{commit}"]).decode().strip()
        if not re.fullmatch(r"[0-9a-f]{40}", revision):
            raise CandidateError("full-sha1-commit-required")
        object_path = Path(git(path, ["rev-parse", "--git-path", "objects"]).decode().strip())
        objects = (path / object_path).resolve(strict=True) if not object_path.is_absolute() else object_path.resolve(strict=True)
        if not objects.is_dir():
            raise CandidateError("local-object-store-required")
        entries = tracked_entries(path, revision, role)
        sources.append(Source(role, folder, repository, path, revision, objects, entries))
    return sources


def validate_output(destination: Path, root: Path) -> Path:
    expanded = destination.expanduser()
    if os.path.lexists(expanded):
        raise CandidateError("output-already-exists")
    path = expanded.resolve(strict=False)
    repositories = [(root.parent / folder).resolve(strict=False) for folder, _ in REPOSITORIES.values()]
    if any(path == repo or path.is_relative_to(repo) or repo.is_relative_to(path) for repo in repositories):
        raise CandidateError("output-must-be-outside-source-repositories")
    for parent in [path.parent, *path.parent.parents]:
        if os.path.lexists(parent / ".git"):
            raise CandidateError("output-must-not-be-nested-in-any-repository")
        if parent.exists() and not parent.is_dir():
            raise CandidateError("output-parent-not-directory")
    return path


def private_file(path: Path, content: bytes) -> None:
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_CLOEXEC | os.O_NOFOLLOW, 0o600)
    with os.fdopen(descriptor, "wb") as output:
        os.fchmod(output.fileno(), 0o600)
        output.write(content)
        output.flush()
        os.fsync(output.fileno())


def source_directories(root: Path, relative: PurePosixPath) -> Path:
    current = root
    for component in relative.parts:
        current = current / component
        current.mkdir(mode=0o755, exist_ok=True)
        info = current.lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid():
            raise CandidateError("unsafe-source-directory")
        os.chmod(current, 0o755)
    return current


def archive_source(source: Source, path: Path, scratch: Path) -> str:
    # A clean bare reader prevents .git/info/attributes, replacement refs and
    # local tar format helpers from reinterpreting the selected commit.
    with tempfile.TemporaryDirectory(prefix="git-reader-", dir=scratch) as directory:
        reader = Path(directory) / "reader.git"
        template = Path(directory) / "empty-template"
        template.mkdir(mode=0o700)
        git(Path(directory), ["init", "--bare", "--quiet", "--template=" + str(template), str(reader)])
        descriptor = os.open(path, os.O_RDWR | os.O_CREAT | os.O_EXCL | os.O_CLOEXEC | os.O_NOFOLLOW, 0o600)
        with os.fdopen(descriptor, "w+b", buffering=0) as output:
            os.fchmod(output.fileno(), 0o600)
            git(reader, ["archive", "--format=tar", "--prefix=" + source.folder + "/", source.revision],
                environment={"GIT_OBJECT_DIRECTORY": str(source.objects)}, stdout=output)
            if output.tell() > MAX_ARCHIVE_BYTES:
                raise CandidateError("archive-exceeds-limit")
            output.flush()
            os.fsync(output.fileno())
            output.seek(0)
            digest = hashlib.file_digest(output, "sha256").hexdigest()
    return digest


def extract_source(source: Source, archive_path: Path, destination: Path) -> int:
    if archive_path.stat().st_size > MAX_ARCHIVE_BYTES:
        raise CandidateError("archive-exceeds-limit")
    descriptor = os.open(archive_path, os.O_RDONLY | os.O_CLOEXEC | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(descriptor, "rb") as raw:
        if not stat.S_ISREG(os.fstat(raw.fileno()).st_mode):
            raise CandidateError("archive-not-regular-file")
        with tarfile.open(fileobj=raw, mode="r:") as archive:
            expected_dirs = {source.folder}
            for path in source.entries:
                for parent in PurePosixPath(path).parents:
                    if str(parent) != ".":
                        expected_dirs.add(source.folder + "/" + str(parent))
            members: list[tuple[tarfile.TarInfo, str]] = []
            seen: set[str] = set()
            files: set[str] = set()
            total_size = 0
            for member in archive:
                name = member.name.rstrip("/") if member.isdir() else member.name
                if len(members) >= MAX_FILES * 2 or not name or not name.startswith(source.folder + "/") and name != source.folder:
                    raise CandidateError("unexpected-archive-root-or-size")
                if name == source.folder:
                    if not member.isdir():
                        raise CandidateError("invalid-archive-root")
                    relative = ""
                else:
                    relative = relative_source_path(name[len(source.folder) + 1:], source.role)
                key = canonical_path(name)
                if key in seen:
                    raise CandidateError("duplicate-archive-path")
                seen.add(key)
                if member.isdir():
                    if name not in expected_dirs:
                        raise CandidateError("unexpected-archive-directory")
                elif member.isreg() and member.sparse is None:
                    expected = source.entries.get(relative)
                    total_size += member.size
                    if (expected is None or member.size < 0 or member.size > MAX_FILE_BYTES
                            or total_size > MAX_SOURCE_BYTES or member.size != expected.size
                            or bool(member.mode & 0o111) != expected.executable):
                        raise CandidateError("archive-differs-from-selected-tree")
                    files.add(relative)
                else:
                    raise CandidateError("archive-links-devices-and-special-files-refused")
                members.append((member, relative))
            if archive.pax_headers.get("comment") != source.revision or files != set(source.entries):
                raise CandidateError("archive-origin-or-tracked-file-set-mismatch")
            # Validate the complete namespace before extracting anything. A new
            # private destination, exclusive files and no link support prevent
            # archive entries from redirecting writes outside this source root.
            target = destination / source.folder
            target.mkdir(mode=0o755)
            os.chmod(target, 0o755)
            for member, relative in members:
                if not relative:
                    continue
                path = target.joinpath(*PurePosixPath(relative).parts)
                if member.isdir():
                    source_directories(target, PurePosixPath(relative))
                    continue
                source_directories(target, PurePosixPath(relative).parent)
                output_fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_CLOEXEC | os.O_NOFOLLOW, 0o600)
                expected = source.entries[relative]
                blob_hash = hashlib.sha1(("blob " + str(expected.size) + "\0").encode())
                stream = archive.extractfile(member)
                if stream is None:
                    os.close(output_fd)
                    raise CandidateError("archive-member-unreadable")
                with os.fdopen(output_fd, "wb") as output, stream:
                    remaining = expected.size
                    while remaining:
                        chunk = stream.read(min(1024 * 1024, remaining))
                        if not chunk:
                            raise CandidateError("archive-member-truncated")
                        output.write(chunk)
                        blob_hash.update(chunk)
                        remaining -= len(chunk)
                    if blob_hash.hexdigest() != expected.oid:
                        raise CandidateError("archive-blob-does-not-match-commit")
                    os.fchmod(output.fileno(), 0o755 if expected.executable else 0o644)
            return len(files)


def prepare(root: Path, destination: Path) -> dict:
    print("Candidate phase: preflight-output", flush=True)
    output = validate_output(destination, root)
    print("Candidate phase: preflight-repositories", flush=True)
    sources = collect_sources(root)
    # No output mutation until every repository has passed the same preflight.
    output.parent.mkdir(parents=True, mode=0o700, exist_ok=True)
    validate_output(output, root)
    output.mkdir(mode=0o700)
    os.chmod(output, 0o700)
    private_file(output / "INCOMPLETE", b"No valid candidate until preparation completes.\n")
    archives = output / "archives"; archives.mkdir(mode=0o700)
    extracted = output / "sources"; extracted.mkdir(mode=0o700)
    records = {}
    for source in sources:
        print("Candidate phase: archive-" + source.role, flush=True)
        archive = archives / (source.folder + ".tar")
        digest = archive_source(source, archive, output)
        print("Candidate phase: extract-" + source.role, flush=True)
        count = extract_source(source, archive, extracted)
        records[source.role] = {"repository": source.repository, "revision": source.revision,
                                "archive": "archives/" + archive.name, "archive_sha256": digest,
                                "source_directory": "sources/" + source.folder, "file_count": count}
        print("source=" + source.role + " revision=" + source.revision, flush=True)
    manifest = {"schema_version": 1, "status": "prepared-not-built",
                "prepared_at": datetime.now(timezone.utc).isoformat(), "sources": records,
                "verification": {"working_trees": "clean-at-preflight", "archive_blob_ids": "matched-selected-commits",
                                 "build": "not-run", "tests": "not-run", "deployment": "not-run"}}
    print("Candidate phase: write-manifest", flush=True)
    private_file(output / "candidate.json", (json.dumps(manifest, indent=2) + "\n").encode())
    (output / "INCOMPLETE").unlink()
    print("PASS source candidate prepared-not-built; no build, test or deployment performed", flush=True)
    return manifest


def main(arguments: list[str] | None = None) -> int:
    parser = SafeParser(prog="release-candidate.py", description=__doc__)
    parser.add_argument("--prepare", action="store_true")
    parser.add_argument("--output", type=Path)
    try:
        options = parser.parse_args(arguments)
        if not options.prepare:
            print("Offline plan only: check three clean sibling HEADs, freeze full revisions, archive, safely extract and record hashes.")
            print("No credentials, Git repositories, Docker, services or historical release candidate were read or changed.")
            print("Use --prepare --output NEW_DIRECTORY to prepare sources; this never builds, tests or deploys.")
            return 0
        if options.output is None:
            raise CandidateError("new-output-required")
        prepare(ROOT, options.output)
        return 0
    except (CandidateError, OSError, ValueError, TypeError, KeyError, tarfile.TarError):
        print("FAIL source candidate preparation; raw Git/configuration values omitted; partial output preserved", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
