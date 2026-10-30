#!/usr/bin/env python3
"""Explicit local, network-isolated physical backup/restore acceptance."""
import argparse
import json
import os
from pathlib import Path
import secrets
import subprocess
import sys
import tempfile
import time

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'scripts'))
import recovery_backup as tool


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--execute', action='store_true')
    parser.add_argument('--docker-socket', type=Path)
    args = parser.parse_args()
    if not args.execute:
        print('DRY RUN: no containers or volumes created')
        return
    tool.configure_docker(args.docker_socket)
    os.umask(0o077)
    project = 'cloudops-fixture-' + secrets.token_hex(16)
    evidence = Path(tempfile.mkdtemp(prefix='cloudops-recovery-evidence-'))
    print('Private evidence:', evidence, flush=True)
    containers = []
    volumes = []
    restored = None

    def command(*parts, data=None):
        result = subprocess.run(tool.docker_command(*parts), input=data, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=120)
        if result.returncode:
            raise RuntimeError('isolated Docker operation failed; private evidence retained')
        return result.stdout.decode().strip()

    def create(role, image, volume=None, restored_volume=None):
        mount = restored_volume or volume
        labels = ['--label', 'cloudops.recovery.fixture=' + project, '--label', 'com.docker.compose.project=' + project, '--label', 'com.docker.compose.service=' + role]
        options = ['run', '-d', '--pull=never', '--network', 'none', *labels]
        if mount:
            target = '/var/lib/mysql' if role == 'mysql' else '/data' if role == 'redis' else '/app/data'
            options += ['--mount', f'type=volume,src={mount},dst={target}']
        if role == 'mysql': options += ['-e', 'MYSQL_ALLOW_EMPTY_PASSWORD=yes', image]
        elif role == 'redis': options += [image, 'redis-server', '--appendonly', 'yes', '--appendfsync', 'always']
        else: options += ['--entrypoint', 'sh', image, '-c', 'true']
        cid = command(*options)
        containers.append(cid)
        return cid

    def ready(cid, role):
        for _ in range(120):
            if role == 'mysql' and command('exec', cid, 'cat', '/proc/1/comm') != 'mysqld':
                time.sleep(0.5)
                continue
            cmd = ['exec', cid, 'mysqladmin', '-uroot', 'ping', '--silent'] if role == 'mysql' else ['exec', cid, 'redis-cli', 'PING']
            result = subprocess.run(tool.docker_command(*cmd), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=10)
            if result.returncode == 0: return
            time.sleep(0.5)
        raise RuntimeError('isolated data service not ready')

    try:
        for role in tool.ROLES:
            name = project + '-' + role
            command('volume', 'create', '--label', 'cloudops.recovery.fixture=' + project, name)
            volumes.append(name)
        by_role = dict(zip(tool.ROLES, volumes))
        source = {role: create(role, 'mysql:8.4.11' if role == 'mysql' else tool.IMAGE, by_role[role]) for role in tool.ROLES}
        create('backend', tool.IMAGE)
        create('web', tool.IMAGE)
        ready(source['mysql'], 'mysql'); ready(source['redis'], 'redis')
        command('exec', '-i', source['mysql'], 'mysql', '-uroot', data=b'CREATE DATABASE fixture; CREATE TABLE fixture.evidence(id INT PRIMARY KEY, value VARCHAR(40)); INSERT INTO fixture.evidence VALUES(1,"preserved");')
        for db, key in [('0', 'queue-state'), ('1', 'approval-tombstone')]:
            command('exec', source['redis'], 'redis-cli', '-n', db, 'SET', key, 'preserved')
        command('run', '--rm', '--network', 'none', '--mount', f'type=volume,src={by_role["aiops"]},dst=/data', '--entrypoint', 'sh', tool.IMAGE, '-c', 'mkdir -p /data/knowledge; printf preserved > /data/knowledge/document.txt')
        command('stop', '-t', '60', source['mysql'], source['redis'])
        config = evidence / 'deployment.env'; config.write_text('fixture-only=true\n'); config.chmod(0o600)
        key = evidence / 'backup.key'; key.write_bytes(secrets.token_bytes(32)); key.chmod(0o600)
        descriptor = {'version': 1, 'project': project, 'containers': source, 'volumes': by_role, 'private_config': [{'name': '.env', 'path': str(config)}], 'revisions': {role: 'a' * 40 for role in ('backend', 'web', 'aiops')}}
        desc = evidence / 'source.json'; desc.write_text(json.dumps(descriptor)); desc.chmod(0o600)
        tool.backup(desc, key, evidence / 'bundle')
        restored = tool.restore(evidence / 'bundle', key, evidence / 'restored')
        restored_mysql = create('mysql', 'mysql:8.4.11', restored_volume=restored['volumes']['mysql'])
        restored_redis = create('redis', tool.IMAGE, restored_volume=restored['volumes']['redis'])
        ready(restored_mysql, 'mysql'); ready(restored_redis, 'redis')
        assert command('exec', restored_mysql, 'mysql', '-uroot', '-Nse', 'SELECT value FROM fixture.evidence WHERE id=1') == 'preserved'
        for db, name in [('0', 'queue-state'), ('1', 'approval-tombstone')]:
            assert command('exec', restored_redis, 'redis-cli', '-n', db, 'GET', name) == 'preserved'
        assert command('run', '--rm', '--network', 'none', '--mount', f'type=volume,src={restored["volumes"]["aiops"]},dst=/data,readonly', '--entrypoint', 'cat', tool.IMAGE, '/data/knowledge/document.txt') == 'preserved'
        assert (evidence / 'restored/private/.env').read_bytes() == config.read_bytes()
        print('PASS isolated MySQL rows, Redis databases/queue/tombstone, knowledge and private config restored')
        print('Private evidence:', evidence)
    finally:
        for cid in reversed(containers):
            if command('inspect', '--format', '{{index .Config.Labels "cloudops.recovery.fixture"}}', cid) != project:
                raise RuntimeError('refusing foreign container cleanup')
            command('rm', '-f', cid)
        for volume in volumes:
            if command('volume', 'inspect', '--format', '{{index .Labels "cloudops.recovery.fixture"}}', volume) != project:
                raise RuntimeError('refusing foreign volume cleanup')
            command('volume', 'rm', volume)
        if restored:
            for volume in restored['volumes'].values():
                if command('volume', 'inspect', '--format', '{{index .Labels "cloudops.restore.id"}}', volume) != restored['restore_id']:
                    raise RuntimeError('refusing foreign restore cleanup')
                command('volume', 'rm', volume)


if __name__ == '__main__':
    main()
