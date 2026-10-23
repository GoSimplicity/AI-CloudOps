#!/usr/bin/env python3
"""Exercise account provisioning/revocation using only one owned release-e2e user."""
import argparse
import json
import os
import secrets
from urllib.error import HTTPError
from urllib.request import Request, urlopen
from urllib.parse import urlsplit
from uuid import uuid4


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--url', default='http://127.0.0.1:17889')
    parser.add_argument('--rbac', action='store_true')
    args = parser.parse_args()
    base = args.url.rstrip('/')
    if urlsplit(base).hostname not in {'localhost', '127.0.0.1', '::1'}:
        raise RuntimeError('Account acceptance only supports an isolated loopback gateway')

    def call(path, data=None, token=None, method=None):
        headers = {'User-Agent': 'cloudops-account-e2e/1.0'}
        if token:
            headers['Authorization'] = 'Bearer ' + token
        if data is not None:
            headers['Content-Type'] = 'application/json'
        request = Request(base + path, headers=headers, data=json.dumps(data).encode() if data is not None else None, method=method)
        try:
            with urlopen(request, timeout=30) as response:
                return response.status, json.loads(response.read())
        except HTTPError as error:
            raw = error.read()
            try:
                payload = json.loads(raw) if raw else {}
            except json.JSONDecodeError:
                payload = {}
            return error.code, payload

    def ok(path, data=None, token=None, method=None):
        status, result = call(path, data, token, method)
        assert status == 200 and result.get('code') == 0, f'{path} failed (HTTP={status}, code={result.get("code")})'
        return result.get('data')

    assert call('/api/user/signup', {})[0] == 401
    print('PASS anonymous signup rejected')
    admin = ok('/api/user/login', {'username': 'admin', 'password': os.environ['BOOTSTRAP_ADMIN_PASSWORD']})['accessToken']
    name = 'release-e2e-' + uuid4().hex[:12]
    password = secrets.token_urlsafe(24)
    mobile = '155' + ''.join(str(secrets.randbelow(10)) for _ in range(8))
    user_id = None
    role_id = None
    try:
        ok('/api/user/signup', {'username': name, 'password': password, 'mobile': mobile, 'real_name': name, 'account_type': 2, 'home_path': '/', 'email': name + '@example.invalid'}, admin)
        user = ok('/api/user/login', {'username': name, 'password': password})
        user_id = user['id']
        profile = ok('/api/user/profile', token=user['accessToken'])
        assert profile['account_type'] == 1 and profile['email'] == name + '@example.invalid'
        print('PASS admin provisioning, login and email persistence without privilege escalation')
        assert call('/api/user/signup', {}, user['accessToken'])[0] == 403
        print('PASS ordinary account cannot provision users')
        if args.rbac:
            apis = ok('/api/apis/list?page=1&size=100&search=/api/workorder/category/list', token=admin)['items']
            permission = next(item for item in apis if item['path'] == '/api/workorder/category/list' and item['method'] == 1)
            role = ok('/api/role/create', {'name': name, 'code': name, 'status': 1, 'api_ids': [permission['id']]}, admin)
            role_id = role['id']
            ok('/api/role/assign_users', {'user_id': user_id, 'role_ids': [role_id]}, admin)
            ok('/api/workorder/category/list?page=1&size=10', token=user['accessToken'])
            assert call('/api/workorder/category/create', {'name': name, 'status': 1}, user['accessToken'])[0] == 403
            print('PASS catalog-based read-only role allows GET and rejects POST')
            # AI 代理实际仍是一条通配 Gin 路由，细分授权必须在真实网关生效
            ai_catalog = ok('/api/apis/list?page=1&size=100&search=/api/v1/assistant/', token=admin)['items']
            ai_grants = [next(item['id'] for item in ai_catalog if item['path'] == path and item['method'] == 1)
                         for path in ['/api/v1/assistant/info', '/api/v1/assistant/session/:session_id']]
            ok(f'/api/role/update/{role_id}', {'name': name, 'code': name, 'status': 1, 'api_ids': ai_grants}, admin, 'PUT')
            ok('/api/v1/assistant/info', token=user['accessToken'])
            missing_session = '/api/v1/assistant/session/' + str(uuid4())
            assert call(missing_session, token=user['accessToken'])[0] == 404
            assert call(missing_session + '/extra', token=user['accessToken'])[0] == 403
            for denied in ['/api/v1/assistant/query', '/api/v1/autofix/workflow/confirm', '/api/v1/autofix/notify']:
                assert call(denied, {}, user['accessToken'])[0] == 403
            print('PASS fine-grained AI grants reach metadata/session routes and reject model, repair and notification actions')
            ok('/api/role/revoke_users', {'user_id': user_id, 'role_ids': [role_id]}, admin)
            assert call('/api/workorder/category/list?page=1&size=10', token=user['accessToken'])[0] == 403
            assert call('/api/v1/assistant/info', token=user['accessToken'])[0] == 403
            ok(f'/api/role/delete/{role_id}', token=admin, method='DELETE')
            role_id = None
            print('PASS role revocation immediately removes access and cleans owned role')
        update = {'real_name': name, 'mobile': mobile, 'account_type': 1, 'home_path': '/', 'enable': 2}
        ok(f'/api/user/profile/update/{user_id}', update, admin, 'PUT')
        assert call('/api/user/profile', token=user['accessToken'])[0] == 401
        status, refreshed = call('/api/user/refresh_token', {'refreshToken': user['refreshToken']})
        assert status == 401 or refreshed.get('code') != 0
        print('PASS administrator suspension immediately rejects existing access and refresh tokens')
        update['enable'] = 1
        ok(f'/api/user/profile/update/{user_id}', update, admin, 'PUT')
        current = ok('/api/user/login', {'username': name, 'password': password})
        ok(f'/api/user/{user_id}/delete', token=admin, method='DELETE')
        user_id = None
        assert call('/api/user/profile', token=current['accessToken'])[0] == 401
        print('PASS deletion immediately rejects existing token and cleans owned account')
    finally:
        if role_id is not None:
            if user_id is not None:
                ok('/api/role/revoke_users', {'user_id': user_id, 'role_ids': [role_id]}, admin)
            ok(f'/api/role/delete/{role_id}', token=admin, method='DELETE')
        if user_id is not None:
            ok(f'/api/user/{user_id}/delete', token=admin, method='DELETE')
            print('PASS owned account cleanup')
        ok('/api/user/logout', {}, admin)


if __name__ == '__main__':
    main()
