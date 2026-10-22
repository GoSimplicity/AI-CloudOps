#!/usr/bin/env python3
"""Run an isolated workorder lifecycle; credentials stay in memory, owned rows are cleaned."""
import argparse
import json
import os
import time
from urllib.error import HTTPError
from urllib.parse import urlencode
from urllib.request import Request, urlopen
from uuid import uuid4


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--url', default='http://127.0.0.1:17889')
    args = parser.parse_args()
    password = os.environ['BOOTSTRAP_ADMIN_PASSWORD']
    token = None
    owned = []
    prefix = 'release-e2e-' + uuid4().hex[:12]

    def call(path, body=None, method=None):
        headers = {'User-Agent': 'cloudops-workorder-e2e/1.0'}
        if token:
            headers['Authorization'] = 'Bearer ' + token
        if body is not None:
            headers['Content-Type'] = 'application/json'
        req = Request(args.url.rstrip('/') + path, data=json.dumps(body).encode() if body is not None else None, headers=headers, method=method)
        try:
            with urlopen(req, timeout=30) as response:
                status, raw = response.status, response.read()
        except HTTPError as error:
            status, raw = error.code, error.read()
        result = json.loads(raw)
        if status != 200 or result.get('code') != 0:
            # The server message is deliberately excluded; it may contain user data.
            raise RuntimeError(f'{method or ("POST" if body is not None else "GET")} {path.split("?")[0]} failed: HTTP={status}, code={result.get("code")}')
        return result.get('data')

    def rows(path):
        data = call(path)
        return data.get('items', data.get('list', []))

    def find(kind, name, field='name'):
        matches = rows('/api/workorder/' + kind + '/list?' + urlencode({'page': 1, 'size': 100, 'search': name}))
        return next(row for row in matches if row[field] == name)

    def create(kind, data):
        call('/api/workorder/' + kind + '/create', data)
        row = find(kind, data['name'])
        owned.append((kind, row['id']))
        print('PASS create ' + kind)
        return row['id']

    login = call('/api/user/login', {'username': 'admin', 'password': password})
    token, uid = login['accessToken'], login['id']
    try:
        category = create('category', {'name': prefix, 'status': 1})
        form = create('form-design', {'name': prefix, 'status': 2, 'category_id': category, 'is_template': 1, 'schema': {'fields': [{'id': 'reason', 'name': 'reason', 'label': 'Reason', 'type': 'text', 'required': 1}]}})
        definition = {'steps': [
            {'id': 'start', 'type': 'start', 'name': 'Start', 'sort_order': 1, 'actions': ['start']},
            {'id': 'approval', 'type': 'approval', 'name': 'Approval', 'sort_order': 2, 'assignee_type': 'user', 'assignee_ids': [uid], 'actions': ['approve', 'reject']},
            {'id': 'end', 'type': 'end', 'name': 'End', 'sort_order': 3}],
            'connections': [{'from': 'start', 'to': 'approval'}, {'from': 'approval', 'to': 'end'}]}
        process = create('process', {'name': prefix, 'form_design_id': form, 'category_id': category, 'definition': definition, 'status': 2, 'is_default': 2})
        template = create('template', {'name': prefix, 'process_id': process, 'form_design_id': form, 'category_id': category, 'status': 1, 'default_values': {'reason': 'isolated integration'}})
        call(f'/api/workorder/instance/create-from-template/{template}', {'title': prefix, 'priority': 2, 'form_data': {'reason': 'isolated integration'}, 'assignee_id': uid})
        instance = find('instance', prefix, 'title')['id']
        owned.append(('instance', instance))
        assert call(f'/api/workorder/instance/detail/{instance}')['status'] == 1
        print('PASS template instance created as draft')
        call(f'/api/workorder/instance/submit/{instance}', {})
        assert call(f'/api/workorder/instance/detail/{instance}')['status'] == 2
        print('PASS submit draft to pending')
        call(f'/api/workorder/instance/approve/{instance}', {'comment': prefix})
        assert call(f'/api/workorder/instance/detail/{instance}')['status'] == 4
        print('PASS approve to completed')
        call('/api/workorder/instance/comment/create', {'instance_id': instance, 'content': prefix, 'type': 'normal', 'status': 1, 'is_system': 2})
        comments = rows(f'/api/workorder/instance/comment/list?page=1&size=100&instance_id={instance}')
        comment = next(row for row in comments if row['content'] == prefix)
        owned.append(('instance/comment', comment['id']))
        assert comment['operator_id'] == uid
        print('PASS comment persisted with authenticated author')
        flows = rows(f'/api/workorder/instance/flow/list?page=1&size=100&instance_id={instance}')
        assert len(flows) >= 3
        timeline = rows(f'/api/workorder/instance/timeline/list?page=1&size=100&instance_id={instance}')
        assert len(timeline) >= 3
        print(f'PASS flow history ({len(flows)}) and timeline ({len(timeline)})')
        for attempt in range(10):
            audits = rows('/api/audit/list?page=1&size=100')
            if any(str(row.get('target_id')) == str(instance) and '/instance/approve/' in row.get('endpoint', '') for row in audits):
                break
            time.sleep(0.2)
        else:
            raise AssertionError('approval audit record missing')
        print('PASS approval audit persisted')
    finally:
        cleanup_failed = []
        for kind, ident in reversed(owned):
            try:
                if kind == 'instance':
                    state = call(f'/api/workorder/instance/detail/{ident}')['status']
                    if state in (2, 3):
                        call(f'/api/workorder/instance/reject/{ident}', {'comment': prefix + ' cleanup'})
                if kind == 'process':
                    call(f'/api/workorder/process/update/{ident}', {'status': 3}, 'PUT')
                call(f'/api/workorder/{kind}/delete/{ident}', method='DELETE')
                print('PASS cleanup ' + kind)
            except Exception:
                cleanup_failed.append(f'{kind}:{ident}')
        call('/api/user/logout', {})
        if cleanup_failed:
            raise RuntimeError('Owned test rows need cleanup: ' + ', '.join(cleanup_failed))


if __name__ == '__main__':
    main()
