#!/usr/bin/env python3
"""Exercise CloudOps against an explicitly created, disposable kind cluster.

Preparation (perform only after the operator authorizes starting kind):
  kind create cluster --name cloudops-release --image \
    kindest/node:v1.33.1@sha256:050072256b9a903bd914c0b2866828150cb229cea0efe5892e2b644d5dd3b34f \
    --kubeconfig "$KUBECONFIG" --wait 120s
  chmod 600 "$KUBECONFIG"
  # Export BOOTSTRAP_ADMIN_PASSWORD privately, then:
  python3 scripts/integration/kubernetes_smoke.py --execute

This script never creates/deletes kind itself or changes the global kubectl context.
It creates a uniquely named CloudOps cluster record and namespace, then removes
only that record and resources bearing its run marker. KUBECONFIG must reference
one private file, current context kind-cloudops-release, and a loopback API server.
For a container backend, --registration-kubeconfig may select a private copy
using host.docker.internal with tls-server-name=127.0.0.1, or the exact dedicated
kind node hostname. The original CA and client identity must match; local kubectl
verification still uses the original loopback KUBECONFIG. TLS cannot be disabled.
Credentials remain in memory; neither subprocess output nor API bodies are logged.

Official sources:
https://github.com/kubernetes-sigs/kind/releases/tag/v0.29.0
https://kind.sigs.k8s.io/docs/user/quick-start/
https://github.com/kubernetes/client-go/blob/v0.35.3/README.md

kind v0.29.0's published default is Kubernetes v1.33.1 (arm64 and amd64).
client-go v0.35.3 matches Kubernetes 1.35 APIs, not exactly 1.33. This smoke uses
only shared stable core/v1 and apps/v1 APIs; passing does not certify 1.35-only
features or every version combination. kubectl v1.33.1 matches this server.
"""

import argparse
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import time
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode, urlsplit
from urllib.request import ProxyHandler, Request, build_opener
from uuid import uuid4


CLUSTER = 'cloudops-release'
CONTEXT = 'kind-' + CLUSTER
OWNER_LABEL = 'cloudops.io/release-smoke'


class SmokeFailure(RuntimeError):
    """Only deliberately sanitized text may be printed from this exception."""


def require(condition, message):
    if not condition:
        raise SmokeFailure(message)


def command(args, timeout=40):
    try:
        result = subprocess.run(args, capture_output=True, text=True, timeout=timeout, check=False)
    except (OSError, subprocess.TimeoutExpired):
        raise SmokeFailure(f'{args[0]} failed or timed out') from None
    require(result.returncode == 0, f'{args[0]} exited {result.returncode}; output withheld')
    return result.stdout


class API:
    def __init__(self, url):
        parsed = urlsplit(url)
        require(parsed.hostname in {'127.0.0.1', 'localhost', '::1'}, 'API URL must be loopback')
        require(parsed.scheme in {'http', 'https'} and not parsed.username and not parsed.password, 'Invalid API URL')
        self.url = url.rstrip('/')
        self.token = os.getenv('CLOUDOPS_API_TOKEN', '')
        self.opener = build_opener(ProxyHandler({}))

    def request(self, method, path, body=None, stream_marker=None):
        headers = {'User-Agent': 'cloudops-kubernetes-release-smoke/1.0'}
        if self.token:
            headers['Authorization'] = 'Bearer ' + self.token
        data = None
        if body is not None:
            headers['Content-Type'] = 'application/json'
            data = json.dumps(body).encode()
        request = Request(self.url + path, data=data, headers=headers, method=method)
        try:
            with self.opener.open(request, timeout=45) as response:
                if stream_marker:
                    require(response.status == 200, 'Pod log endpoint did not succeed')
                    # The endpoint uses SSE even when follow=false. Close after our
                    # marker rather than waiting for an unrelated stream heartbeat.
                    for _ in range(100):
                        line = response.readline(65536)
                        if not line:
                            break
                        if stream_marker.encode() in line:
                            return True
                    raise SmokeFailure('Pod log stream did not contain the owned marker')
                result = json.loads(response.read(4 * 1024 * 1024))
        except HTTPError as error:
            raise SmokeFailure(f'{method} {path.split("?")[0]}: HTTP {error.code}; body withheld') from None
        except (URLError, TimeoutError, ValueError, OSError):
            raise SmokeFailure(f'{method} {path.split("?")[0]} failed; details withheld') from None
        require(result.get('code') == 0, f'{method} {path.split("?")[0]}: business code {result.get("code")}')
        return result.get('data')

    def login(self):
        if not self.token:
            password = os.getenv('BOOTSTRAP_ADMIN_PASSWORD', '')
            require(bool(password), 'Set BOOTSTRAP_ADMIN_PASSWORD or CLOUDOPS_API_TOKEN')
            result = self.request('POST', '/api/user/login', {
                'username': os.getenv('CLOUDOPS_API_USERNAME', 'admin'), 'password': password,
            })
            self.token = result['accessToken']

    def cluster_rows(self, name):
        data = self.request('GET', '/api/k8s/cluster/list?' + urlencode({'page': 1, 'size': 100, 'search': name}))
        return [row for row in (data.get('items') or []) if row.get('name') == name]


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('--url', default=os.getenv('CLOUDOPS_API_URL', 'http://127.0.0.1:17889'))
    parser.add_argument('--registration-kubeconfig', help='Private copy for container backend only; server must be host.docker.internal or cloudops-release-control-plane with verified TLS')
    parser.add_argument('--image', default='busybox:1.37.0', help='Small HTTP workload image; preload into the dedicated kind cluster for offline use')
    parser.add_argument('--execute', action='store_true', help='Run owned resource mutations; without this flag only private-cluster preflight is performed')
    parser.add_argument('--keep-resources', action='store_true', help='Retain successful owned resources for UI acceptance; failures still clean up')
    args = parser.parse_args()

    raw_path = os.getenv('KUBECONFIG', '')
    require(bool(raw_path) and os.pathsep not in raw_path, 'KUBECONFIG must identify exactly one private file')
    path = Path(raw_path).expanduser()
    require(path.is_absolute() and path.is_file() and not path.is_symlink(), 'KUBECONFIG must be an absolute regular file, not a symlink')
    require(path.resolve() != (Path.home() / '.kube/config').resolve(), 'Default user kubeconfig is forbidden')
    require(not stat.S_IMODE(path.stat().st_mode) & 0o077, 'KUBECONFIG must be private (chmod 600)')
    kubectl = ['kubectl', '--kubeconfig', str(path), '--context', CONTEXT]
    kubeconfig = json.loads(command(kubectl + ['config', 'view', '--raw', '--minify', '-o', 'json']))
    require(kubeconfig.get('current-context') == CONTEXT, 'Kubeconfig must select only kind-cloudops-release')
    require(len(kubeconfig.get('clusters', [])) == 1, 'Expected a single dedicated cluster')
    server = kubeconfig['clusters'][0]['cluster']['server']
    require(urlsplit(server).hostname in {'127.0.0.1', 'localhost', '::1'}, 'Kubernetes API server must be loopback')
    require(CLUSTER in command(['kind', 'get', 'clusters']).splitlines(), 'Dedicated kind cluster is absent')
    nodes = json.loads(command(kubectl + ['get', 'nodes', '-o', 'json']))['items']
    require(bool(nodes) and all(node['metadata']['name'].startswith(CLUSTER + '-') for node in nodes), 'Node identities do not match the dedicated cluster')
    if args.registration_kubeconfig:
        registration = Path(args.registration_kubeconfig).expanduser()
        require(registration.is_absolute() and registration.is_file() and not registration.is_symlink(), 'Registration kubeconfig must be a private regular file')
        require(not stat.S_IMODE(registration.stat().st_mode) & 0o077, 'Registration kubeconfig must have mode 600')
        copied = json.loads(command(['kubectl', '--kubeconfig', str(registration), '--context', CONTEXT, 'config', 'view', '--raw', '--minify', '-o', 'json']))
        require(copied.get('current-context') == CONTEXT and len(copied.get('clusters', [])) == 1, 'Registration copy must contain the dedicated kind context')
        cluster_config = copied['clusters'][0]['cluster']
        local_config = kubeconfig['clusters'][0]['cluster']
        server = cluster_config['server']
        require(urlsplit(server).scheme == 'https' and urlsplit(server).hostname in {'host.docker.internal', 'cloudops-release-control-plane'}, 'Registration server must be one of the two dedicated local Docker targets')
        require(not cluster_config.get('insecure-skip-tls-verify'), 'TLS verification cannot be disabled')
        require(cluster_config.get('certificate-authority-data') == local_config.get('certificate-authority-data'), 'Registration copy must retain the original CA')
        require(copied.get('users') == kubeconfig.get('users'), 'Registration copy must retain the dedicated client identity')
        if urlsplit(server).hostname == 'host.docker.internal':
            require(cluster_config.get('tls-server-name') == '127.0.0.1', 'Docker-host copy must verify the original loopback TLS identity')
        kubeconfig = copied
    print('PASS dedicated-cluster preflight; credentials withheld')
    if not args.execute:
        print('READY; rerun with --execute only after resource creation is authorized')
        return

    run_id = uuid4().hex[:12]
    name = f'cloudops-release-smoke-{run_id}'
    marker = 'cloudops-release-log-' + run_id
    api = API(args.url)
    api.login()
    require(not api.cluster_rows(name), 'Refusing to adopt an existing cluster record')
    cluster_id = None
    record_attempted = False
    namespace_attempted = False
    cleanup_errors = []
    failure = None

    def get_namespace():
        raw = command(kubectl + ['get', 'namespace', name, '--ignore-not-found', '-o', 'json'])
        return json.loads(raw) if raw.strip() else None

    def wait_replicas(count):
        deadline = time.monotonic() + 150
        while time.monotonic() < deadline:
            detail = api.request('GET', f'/api/k8s/deployment/{cluster_id}/{name}/web/detail')
            if detail.get('replicas') == count and detail.get('ready_replicas') == count:
                return
            time.sleep(2)
        raise SmokeFailure(f'Deployment failed to reach {count} ready replicas')

    try:
        record_attempted = True
        api.request('POST', '/api/k8s/cluster/create', {
            'name': name, 'env': 2, 'api_server_addr': server,
            'kube_config_content': json.dumps(kubeconfig), 'action_timeout_seconds': 30,
            'tags': [{'key': OWNER_LABEL, 'value': run_id}],
        })
        matches = api.cluster_rows(name)
        require(len(matches) == 1, 'Created cluster record could not be uniquely found')
        cluster_id = matches[0]['id']
        print(f'PASS cluster registration id={cluster_id} name={name}')
        require(get_namespace() is None, 'Refusing to adopt an existing namespace')
        namespace_attempted = True
        api.request('POST', f'/api/k8s/clusters/{cluster_id}/namespace/create', {
            'name': name, 'labels': [{'key': OWNER_LABEL, 'value': run_id}],
        })
        api.request('GET', f'/api/k8s/clusters/{cluster_id}/namespace/{name}/detail')
        print('PASS namespace create/read')

        labels = {'app': name, OWNER_LABEL: run_id}
        manifests = [
            ('configmap', {'apiVersion': 'v1', 'kind': 'ConfigMap', 'metadata': {'name': 'web-content', 'namespace': name, 'labels': labels}, 'data': {'index.html': marker}}),
            ('deployment', {'apiVersion': 'apps/v1', 'kind': 'Deployment', 'metadata': {'name': 'web', 'namespace': name, 'labels': labels}, 'spec': {
                'replicas': 1, 'selector': {'matchLabels': {'app': name}}, 'template': {'metadata': {'labels': labels}, 'spec': {'containers': [{
                    'name': 'web', 'image': args.image, 'imagePullPolicy': 'IfNotPresent',
                    'command': ['/bin/sh', '-c', 'echo ' + marker + '; exec httpd -f -p 8080 -h /www'],
                    'ports': [{'containerPort': 8080}], 'resources': {'requests': {'cpu': '20m', 'memory': '16Mi'}, 'limits': {'cpu': '100m', 'memory': '64Mi'}},
                    'readinessProbe': {'httpGet': {'path': '/', 'port': 8080}, 'initialDelaySeconds': 1, 'periodSeconds': 2},
                    'volumeMounts': [{'name': 'content', 'mountPath': '/www', 'readOnly': True}],
                }], 'volumes': [{'name': 'content', 'configMap': {'name': 'web-content'}}]}}}}),
            ('service', {'apiVersion': 'v1', 'kind': 'Service', 'metadata': {'name': 'web', 'namespace': name, 'labels': labels}, 'spec': {
                'selector': {'app': name}, 'ports': [{'port': 80, 'targetPort': 8080}],
            }}),
        ]
        for kind, manifest in manifests:
            api.request('POST', f'/api/k8s/{kind}/{cluster_id}/create/yaml', {'yaml': json.dumps(manifest)})
            api.request('GET', f'/api/k8s/{kind}/{cluster_id}/{name}/{manifest["metadata"]["name"]}/detail')
            print(f'PASS {kind} create/read')
        wait_replicas(1)
        api.request('POST', f'/api/k8s/deployment/{cluster_id}/{name}/web/scale', {'replicas': 2})
        wait_replicas(2)
        pods = api.request('GET', f'/api/k8s/deployment/{cluster_id}/{name}/web/pods')['items']
        require(len(pods) == 2, 'Deployment API did not return two pods')
        print('PASS deployment scale/readiness/pod list')
        pod = pods[0]['name']
        api.request('GET', f'/api/k8s/pod/{cluster_id}/{name}/{pod}/containers/web/logs?follow=false&tail_lines=20', stream_marker=marker)
        print('PASS pod log stream contains owned marker')
        endpoints = api.request('GET', f'/api/k8s/service/{cluster_id}/{name}/web/endpoints')
        require(bool(endpoints), 'Service endpoints are empty')
        print('PASS service endpoint discovery')
        content = command(kubectl + ['get', '--raw', f'/api/v1/namespaces/{name}/services/web:80/proxy/'])
        require(marker in content, 'Service HTTP response does not match the owned ConfigMap')
        print('PASS service HTTP response matches ConfigMap')
    except BaseException as error:
        failure = error
    finally:
        # Exact name plus unique ownership marker is required even after a timeout.
        retain = args.keep_resources and failure is None
        if namespace_attempted and not retain:
            try:
                namespace = get_namespace()
                if namespace is not None:
                    require(namespace['metadata'].get('labels', {}).get(OWNER_LABEL) == run_id, 'Namespace ownership mismatch; cleanup refused')
                    try:
                        api.request('DELETE', f'/api/k8s/clusters/{cluster_id}/namespace/{name}/delete', {'force': 2})
                    except SmokeFailure:
                        command(kubectl + ['delete', 'namespace', name, '--wait=false'])
                        cleanup_errors.append('Namespace API deletion failed; owned kubectl fallback used')
                    if get_namespace() is not None:
                        command(kubectl + ['wait', '--for=delete', f'namespace/{name}', '--timeout=90s'], timeout=100)
                print('PASS owned namespace cleanup')
            except BaseException:
                cleanup_errors.append('Owned namespace cleanup incomplete: ' + name)
        if record_attempted and not retain:
            try:
                matches = api.cluster_rows(name)
                for row in matches:
                    tags = {tag['key']: tag['value'] for tag in row.get('tags', [])}
                    require(row['name'] == name and tags.get(OWNER_LABEL) == run_id, 'Cluster record ownership mismatch')
                    api.request('DELETE', f'/api/k8s/cluster/{row["id"]}/delete')
                require(not api.cluster_rows(name), 'Owned cluster record still exists')
                print('PASS owned cluster record cleanup')
            except BaseException:
                cleanup_errors.append('Owned cluster record cleanup incomplete: ' + name)
    for error in cleanup_errors:
        print('FAIL ' + error, file=sys.stderr)
    if failure:
        if isinstance(failure, SmokeFailure):
            raise failure
        raise SmokeFailure('Integration failed; exception details withheld') from None
    require(not cleanup_errors, 'Integration cleanup did not pass')
    if args.keep_resources:
        print(f'RETAINED cluster_id={cluster_id} namespace={name} owner={run_id}; operator must delete this namespace and API cluster record after UI acceptance')
    print('PASS Kubernetes integration; dedicated kind infrastructure retained for operator cleanup')


if __name__ == '__main__':
    try:
        main()
    except SmokeFailure as error:
        print('FAIL ' + str(error), file=sys.stderr)
        sys.exit(1)
    except Exception:
        print('FAIL unexpected integration error; details withheld', file=sys.stderr)
        sys.exit(1)
