#!/usr/bin/env python3
"""Verify persisted cluster quota settings against the dedicated local kind cluster."""
import argparse
import json
import os
from pathlib import Path
import stat
import subprocess
import time
from uuid import uuid4

from kubernetes_smoke import API, CONTEXT, SmokeFailure, command, require


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--url', default='http://localhost:18080')
    parser.add_argument('--registration-kubeconfig', type=Path, required=True)
    args = parser.parse_args()
    path = Path(os.environ['KUBECONFIG'])
    for file in (path, args.registration_kubeconfig):
        require(file.is_absolute() and file.is_file() and not file.is_symlink(), 'Private absolute kubeconfig required')
        require(not stat.S_IMODE(file.stat().st_mode) & 0o077, 'Kubeconfig must have private permissions')
    kubectl = ['kubectl', '--kubeconfig', str(path), '--context', CONTEXT]
    raw = json.loads(command(kubectl + ['config', 'view', '--raw', '--minify', '-o', 'json']))
    require(raw.get('current-context') == CONTEXT, 'Dedicated context required')
    nodes = json.loads(command(kubectl + ['get', 'nodes', '-o', 'json']))['items']
    require(nodes and all(x['metadata']['name'].startswith('cloudops-release-') for x in nodes), 'Foreign cluster forbidden')
    registration = json.loads(command(['kubectl', '--kubeconfig', str(args.registration_kubeconfig), '--context', CONTEXT, 'config', 'view', '--raw', '--minify', '-o', 'json']))
    # The alternate address may differ for container routing, but never identity or CA.
    require(raw['users'] == registration['users'], 'Registration credentials must match dedicated cluster')
    require(raw['clusters'][0]['cluster']['certificate-authority-data'] == registration['clusters'][0]['cluster']['certificate-authority-data'], 'Registration CA mismatch')
    name = 'cloudops-quota-api-' + uuid4().hex[:10]
    api = API(args.url)
    api.login()
    ident = None
    created = False
    try:
        subprocess.run(kubectl + ['create', 'namespace', name], check=True, capture_output=True)
        created = True
        api.request('POST', '/api/k8s/cluster/create', {
            'name': name, 'env': 2,
            'api_server_addr': registration['clusters'][0]['cluster']['server'],
            'kube_config_content': json.dumps(registration), 'action_timeout_seconds': 30,
            'cpu_request': '100', 'cpu_limit': '200', 'restrict_namespace': [name],
        })
        matches = api.cluster_rows(name)
        require(len(matches) == 1, 'Owned cluster record not found')
        ident = matches[0]['id']
        quota_name = 'cloudops-quota-' + str(ident)
        quota = json.loads(command(kubectl + ['get', 'resourcequota', quota_name, '-n', name, '-o', 'json']))
        require(quota['spec']['hard']['requests.cpu'] == '100m', 'API quota unit mapping incorrect')
        pod = {'apiVersion': 'v1', 'kind': 'Pod', 'metadata': {'name': 'bounded', 'namespace': name}, 'spec': {'containers': [{'name': 'web', 'image': 'busybox:1.37.0', 'command': ['sleep', '120'], 'resources': {'requests': {'cpu': '200m'}, 'limits': {'cpu': '200m'}}}]}}

        def create_pod():
            return subprocess.run(kubectl + ['create', '-f', '-'], input=json.dumps(pod), text=True, capture_output=True, timeout=20)

        response = create_pod()
        require(response.returncode != 0 and 'exceeded quota' in response.stderr, 'Over-budget Pod was not rejected by quota')
        print('PASS quota persisted through API and rejects over-budget Pod')
        api.request('PUT', f'/api/k8s/cluster/{ident}/update', {'name': name, 'cpu_request': '300', 'cpu_limit': '400', 'restrict_namespace': [name]})
        deadline = time.monotonic() + 15
        while True:
            response = create_pod()
            if response.returncode == 0:
                break
            require('exceeded quota' in response.stderr and time.monotonic() < deadline, 'Updated quota did not admit Pod')
            time.sleep(0.25)
        print('PASS quota edit is enforced after controller convergence')
        api.request('PUT', f'/api/k8s/cluster/{ident}/update', {'name': name, 'restrict_namespace': []})
        require(not command(kubectl + ['get', 'resourcequota', quota_name, '-n', name, '--ignore-not-found', '-o', 'json']).strip(), 'Quota not removed after clearing settings')
        row = api.cluster_rows(name)[0]
        require(not row.get('cpu_request') and not row.get('restrict_namespace'), 'Cleared settings not persisted')
        print('PASS quota removal and persisted empty configuration')
    finally:
        if ident is not None:
            api.request('DELETE', f'/api/k8s/cluster/{ident}/delete')
        if created:
            command(kubectl + ['delete', 'namespace', name, '--wait=false'])
        api.request('POST', '/api/user/logout', {})


if __name__ == '__main__':
    try:
        main()
    except SmokeFailure as error:
        raise SystemExit(str(error)) from None
