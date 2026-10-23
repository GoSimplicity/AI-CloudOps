# AI-CloudOps

**English** | [简体中文](README.md)

AI-CloudOps is an open-source, AI-assisted cloud-native operations platform. It brings Kubernetes management, a service tree / CMDB, Prometheus monitoring and alerting, ticketing (work orders), scheduled jobs and RBAC into a single console, and adds an AIOps service for root cause analysis, load prediction, an operations assistant, and human-approved Kubernetes auto-remediation.

This repository is the **Go backend** and the API gateway for the whole platform.

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3.5-4FC08D?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![Python](https://img.shields.io/badge/Python-3.11-3776AB?logo=python&logoColor=white)](https://www.python.org/)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Stars](https://img.shields.io/github/stars/GoSimplicity/AI-CloudOps)](https://github.com/GoSimplicity/AI-CloudOps/stargazers)

## Project Components

AI-CloudOps is made up of three repositories that are deployed together:

| Repository | Role | Stack |
| --- | --- | --- |
| **[AI-CloudOps](https://github.com/GoSimplicity/AI-CloudOps)** (this repo) | Backend API, authentication/RBAC, and gateway to the AI service | Go, Gin, GORM, MySQL, Redis |
| **[AI-CloudOps-web](https://github.com/GoSimplicity/AI-CloudOps-web)** | Web console | Vue 3, TypeScript, Vben Admin, Ant Design Vue |
| **[AI-CloudOps-aiops](https://github.com/GoSimplicity/AI-CloudOps-aiops)** | AIOps service: RCA, prediction, assistant (RAG + MCP), auto-remediation | Python, FastAPI, LangGraph, scikit-learn |

```text
Browser ──> Web console (static files)
   │
   └── /api/*      ──> Go backend (JWT + per-request RBAC) ──> MySQL / Redis / Kubernetes / cloud APIs / SSH
       /api/v1/*   ──> Go backend (JWT + RBAC) ──X-API-Key──> AIOps service ──> Prometheus / Kubernetes / LLM / MCP server
```

The browser only ever holds a user session token. The internal `AIOPS_API_KEY` is injected by the Go gateway and is never sent to the frontend.

## Features

The list below reflects what is implemented in this repository today.

### Kubernetes management
- Register clusters by uploading a kubeconfig. Kubeconfigs are parsed under a strict security boundary: `exec`/`auth-provider` plugins, server-side file paths, `proxy-url`, plain HTTP and skipped TLS verification are rejected (see [docs/kubeconfig-security.md](docs/kubeconfig-security.md)).
- View and manage Namespaces, Nodes, Pods, Deployments, StatefulSets, DaemonSets, Services, Ingresses, ConfigMaps, Secrets, PVs/PVCs, ServiceAccounts, Roles/ClusterRoles and their bindings, and Events.
- YAML templates and YAML apply tasks.
- Optional per-namespace `ResourceQuota` management from the cluster's CPU/memory settings ([docs/kubernetes-resource-policy.md](docs/kubernetes-resource-policy.md)).

### Service tree and CMDB
- Hierarchical service tree with node membership and resource binding.
- Local hosts with encrypted credentials and a browser-based SSH terminal over WebSocket. In production, host keys are verified against an operator-provided `known_hosts` file.
- Cloud accounts and read-only instance sync. The backend has SDK implementations for **Alibaba Cloud ECS, AWS EC2 and Tencent Cloud CVM** (credential check, live region listing, instance sync). The sync only reads cloud APIs; it never creates, modifies or deletes cloud instances. See [docs/multicloud-compute.md](docs/multicloud-compute.md). *Note: the current web console only enables Alibaba Cloud in its account form.*

### Monitoring and alerting (Prometheus)
- Manage scrape pools and scrape jobs, alert rules, recording rules and Alertmanager configuration.
- On-duty groups, alert send groups, alert event history.
- A separate alert webhook service (`cmd/webhook`) that receives Alertmanager notifications and delivers them to Feishu (Lark).

### Ticketing (work orders)
- Categories, form designer, process (workflow) designer, templates.
- Ticket instances with flow actions, comments and timeline.
- Notifications by email and Feishu to creators, assignees, specified users or role members (deduplicated). Delivery failures are reported as errors, never as false successes.

### Scheduled jobs
- Cron jobs of type system, command, HTTP, script and SSH remote execution, queued through Redis (asynq).

### Users, RBAC and audit
- Users, roles, departments and an API permission catalog. The catalog is built from the registered Gin routes at every start-up; permissions are checked on every request.
- Accounts are provisioned by the administrator (no anonymous self sign-up).
- Audit log of operations (metadata only; credentials in request/response bodies are not stored).

### AI gateway
- `/api/v1/*` is reverse-proxied to the AIOps service after JWT and role checks. This exposes RCA, prediction, the operations assistant (including streaming), inspection and auto-remediation to the web console. See [AI-CloudOps-aiops](https://github.com/GoSimplicity/AI-CloudOps-aiops) for details.

## Tech Stack

- **Language / runtime:** Go 1.26.10 (`GOTOOLCHAIN=auto` will download the toolchain declared in `go.mod`)
- **Web framework:** Gin; dependency injection with Google Wire
- **Storage:** MySQL 8 (GORM), Redis (cache, sessions, asynq task queue)
- **Integrations:** Kubernetes client-go 0.35 and OpenKruise API, Prometheus / Alertmanager libraries, Alibaba Cloud / AWS / Tencent Cloud SDKs, SSH, WebSocket, SSE
- **Auth:** JWT (golang-jwt v5) with refresh tokens; per-request account and role checks

## Repository Layout

```text
AI-CloudOps/
├── main.go                 # Main API server entry point
├── cmd/webhook/            # Alertmanager -> Feishu webhook service
├── config/                 # config.development.yaml / config.production.yaml / webhook.yaml
├── internal/
│   ├── app/                # App wiring, AI gateway, admin bootstrap, permission catalog
│   ├── config/             # Config loading and production validation
│   ├── middleware/         # Auth, login, audit, logging
│   ├── k8s/                # Kubernetes management
│   ├── tree/               # Service tree, local hosts, cloud accounts/resources, SSH
│   ├── prometheus/         # Monitoring config, alerting, webhook
│   ├── workorder/          # Ticketing and notifications
│   ├── cron/               # Scheduled jobs
│   ├── system/             # Users, roles, departments, APIs, audit
│   └── model/              # GORM models
├── pkg/                    # Shared packages (jwt, ssh, terminal, websocket, sse, di, ...)
├── deploy/                 # MySQL init, Prometheus, nginx, release Compose bundle
├── scripts/                # Release helpers and integration scripts
└── docs/                   # Design notes, security baseline, release docs
```

## Quick Start (development)

### Requirements

- Go 1.26.10+ (or any recent Go with `GOTOOLCHAIN=auto`)
- Docker with Docker Compose v2
- Node.js 22 and pnpm 9 (for the [web console](https://github.com/GoSimplicity/AI-CloudOps-web))
- Python 3.11 (optional, for the [AIOps service](https://github.com/GoSimplicity/AI-CloudOps-aiops))

### 1. Start MySQL, Redis and Prometheus

The root `docker-compose.yaml` starts the development dependencies (the application services in it are commented out):

```bash
git clone https://github.com/GoSimplicity/AI-CloudOps.git
cd AI-CloudOps
docker compose up -d
```

This exposes MySQL on `3306`, Redis on `36379` and Prometheus on `9090`. The credentials in this file are development defaults only.

### 2. Configure and run the backend

```bash
cp env.example .env
# Edit .env. With the dev Compose file, Redis listens on localhost:36379:
#   REDIS_ADDR=localhost:36379
# Set BOOTSTRAP_ADMIN_PASSWORD (>= 12 chars) to create the `admin` user on first start.
# Replace JWT_KEY1 and JWT_KEY2 with two freshly generated keys (see below).
go run main.go
```

JWT signing keys are validated in **every** environment, including development; there is no switch to relax this. `JWT_KEY1` and `JWT_KEY2` must each be at least 32 characters with at least 10 distinct characters, must not have leading/trailing whitespace or contain control characters (nothing is trimmed silently), must differ from each other (compared case-insensitively after trimming), and must not be or contain a repository sample or historical default value (the placeholders in `env.example` are rejected, in any letter case). Error messages name the variable and the reason only; they never print the key.

**Required before the first start:** generate two independent keys and put them in `.env`, otherwise the backend refuses to start:

```bash
openssl rand -base64 48   # use the output as JWT_KEY1
openssl rand -base64 48   # run again for JWT_KEY2
```

The API listens on `http://localhost:8889` (`SERVER_PORT`). `GET /health` is a liveness probe only; it does not mean MySQL, Redis or external services are ready.

Configuration precedence is **environment variables > config file > defaults**. `ENV=production` selects `config/config.production.yaml`; anything else uses `config/config.development.yaml`. You can also pass `--config <file>`.

### 3. Run the web console

```bash
git clone https://github.com/GoSimplicity/AI-CloudOps-web.git
cd AI-CloudOps-web
pnpm install --frozen-lockfile
pnpm dev            # http://localhost:5666, proxies /api to http://localhost:8889
```

### 4. Run the AIOps service (optional)

Follow the [AI-CloudOps-aiops README](https://github.com/GoSimplicity/AI-CloudOps-aiops). Then set `AIOPS_BASE_URL` (for example `http://localhost:8080`) and the same `AIOPS_API_KEY` in the backend `.env`. If `AIOPS_BASE_URL` is empty, the `/api/v1/*` gateway is not registered.

## Deployment (all three services)

`deploy/release/` contains a Docker Compose bundle that builds and runs the complete platform behind an nginx gateway: MySQL 8.4, Redis 7.4, Prometheus, the Go backend, the AIOps API, the MCP server and the web console.

Place the three repositories side by side. The directory names of the frontend and AIOps repos must match what the Compose file expects:

```text
<workspace>/
├── AI-CloudOps/          # this repo (any name works)
├── AI-CloudOps-web/
└── Ai-CloudOps-aiops/    # note the "Ai-" spelling expected by deploy/release/compose.yaml
```

From the backend repository:

```bash
bash scripts/release-start.sh
```

The script checks the sibling repos and Docker, generates a private `deploy/release/.env` (mode `0600`, git-ignored, never overwritten) with random secrets, then builds and waits for all services to become healthy. The console is served at `http://localhost:18080`, bound to the loopback interface. The initial account is `admin`; its password is `BOOTSTRAP_ADMIN_PASSWORD` in `deploy/release/.env`.

Step by step:

```bash
python3 scripts/release-env.py
docker compose -f deploy/release/compose.yaml config --quiet
docker compose -f deploy/release/compose.yaml up -d --build --wait --wait-timeout 300
python3 scripts/release-smoke.py
```

For public access, put a TLS reverse proxy in front and set `PUBLIC_ORIGIN` to the real HTTPS origin. Do not expose MySQL, Redis or the AIOps service directly. Backups, upgrades, rollback and release gates are described in [docs/release/README.md](docs/release/README.md) (Chinese).

Tagged releases (`v*`) are built by GitHub Actions after `go test`, a gateway smoke test, `go vet` and `govulncheck`. The source archive and a Linux amd64 binary are published with `SHA256SUMS`.

## Configuration

All settings can be provided as environment variables (see [`env.example`](env.example)). Key variables:

| Variable | Description |
| --- | --- |
| `ENV` | `development` (default) or `production`. Production turns on the strict checks below. |
| `SERVER_PORT` | HTTP port, default `8889`. |
| `SERVER_ALLOWED_ORIGINS` | Comma-separated list of exact browser origins allowed for CORS and WebSocket. Empty means same-origin only. |
| `MYSQL_ADDR`, `REDIS_ADDR`, `REDIS_PASSWORD` | Database and Redis connection. |
| `JWT_KEY1`, `JWT_KEY2`, `JWT_EXPIRATION` | JWT signing keys and settings. `JWT_KEY1` and `JWT_KEY2` are required in every environment: at least 32 characters each, distinct, and not a repository sample value (generate with `openssl rand -base64 48`). `JWT_EXPIRATION` is the access-token lifetime in **minutes** (not seconds); the default and the release configuration are `30`. A value such as `3600` means 60 hours. Expired access tokens are renewed automatically by the frontend with the refresh token (valid for 7 days). |
| `CRON_ALLOW_LOCAL_EXEC` | `cron.allow_local_exec`, default `false`. Allows command (type 2) and script (type 4) jobs to run on the backend host. Read at startup; restart the backend after changing it. See the notes below. |
| `JWT_ISSUER` | Optional. The token issuer is configured as `jwt.issuer` in `config/*.yaml`; set this variable only to override it. The value must not be empty (the backend fails to start otherwise), and changing it invalidates all tokens already issued. After upgrading to the release that requires an issuer (`iss`), all older refresh tokens without `iss` are invalid and every user has to log in again once. |
| `TREE_PASSWORD_ENCRYPTION_KEY` | Key used to encrypt stored host passwords. Keep it safe: changing it breaks decryption of existing credentials. |
| `BOOTSTRAP_ADMIN_PASSWORD` | Creates the `admin` account on the first start with an empty database (min. 12 characters). Remove it after the first successful start. Existing users are never reset. |
| `SSH_KNOWN_HOSTS` | Path to an operator-verified OpenSSH `known_hosts` file used for web terminal and SSH connections. |
| `AIOPS_BASE_URL` | URL of the AIOps service, e.g. `http://aiops:8080`. Enables the `/api/v1/*` gateway. |
| `AIOPS_API_KEY` | Shared service key sent to the AIOps service as `X-API-Key`. Must match the Python service. Never put it in frontend config. |
| `MOCK_ENABLED` | Seed mock data for development. Must be `false` in production. |
| `LLM_API_KEY`, `LLM_BASE_URL`, `ALIYUN_ACCESS_KEY_ID`, `ALIYUN_ACCESS_KEY_SECRET`, `TAVILY_API_KEY` | External service credentials (environment variables only). |
| `NOTIFICATION_EMAIL_*`, `NOTIFICATION_FEISHU_*` | Email (SMTP) and Feishu notification channels. |
| `WEBHOOK_*` | Settings for the alert webhook service in `cmd/webhook`. `WEBHOOK_PORT` defaults to `8889` (previously `8888`), matching `config/webhook.yaml`. |

### Production security requirements

In every environment (development included) the backend refuses to start unless `JWT_KEY1` and `JWT_KEY2` are distinct (case-insensitive), at least 32 characters with at least 10 distinct characters, free of leading/trailing whitespace and control characters, and not (or containing) the repository sample values.

With `ENV=production` the backend additionally refuses to start unless:

- `TREE_PASSWORD_ENCRYPTION_KEY` is exactly 32 bytes and not the sample value.
- `MOCK_ENABLED=false`.
- `AIOPS_API_KEY` is at least 32 characters whenever `AIOPS_BASE_URL` is set.

Other security behaviour to be aware of:

- In production, SSH connections require `SSH_KNOWN_HOSTS`. Connections are refused if the file is missing or the host key does not match.
- Browser origins are checked against `SERVER_ALLOWED_ORIGINS` (exact match) or same-origin, for both HTTP and WebSocket.
- Every protected request and token refresh re-checks the current account. Frozen, deleted or renamed accounts are rejected immediately, and authentication fails closed if the database is unavailable.
- `POST /api/user/signup` requires an `admin` session. New users are normal accounts by default.
- Non-admin users need explicit role permissions for each `/api/v1/` path they call. Service accounts no longer bypass role checks.
- Turning `CRON_ALLOW_LOCAL_EXEC` off only blocks creating, enabling, scheduling and running command/script jobs from then on. It does **not** stop command or script processes that are already running (or background processes they spawned); check the backend host and terminate leftovers manually.
- Custom environment variables (the `environment` field) apply only to command jobs (type 2), and only command jobs are validated on save and before execution. HTTP, SSH and script jobs do not use this field and are never blocked by its content (SSH jobs use the separate `ssh_environment`). For command jobs:
  - rows whose name and value are both empty (blank rows added in the UI) are dropped on save and skipped at execution;
  - a row with an empty name but a non-empty value, or a name containing `=` or NUL, is rejected;
  - these names are rejected (case-insensitive): prefixes `LD_*`, `DYLD_*`, `BASH_FUNC_*`, `MALLOC_*`, and `BASH_ENV`, `ENV`, `IFS`, `SHELLOPTS`, `BASHOPTS`, `PS4`, `GCONV_PATH`, `GLIBC_TUNABLES`, `LOCPATH`;
  - errors name the variable only, never its value. Existing non-compliant command jobs are rejected at execution until they are edited and saved again.

See [docs/security-baseline.md](docs/security-baseline.md) for the vulnerability scan baseline and the list of hardened boundaries.

## Development

```bash
make fmt          # go fmt
make lint         # go vet
go test ./...
go test -tags=smoke ./internal/app
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
```

CI runs tests, `govulncheck`, the gateway smoke test and a build on every push to `main` and on pull requests. Real-HTTP acceptance scripts for account permissions, the full ticket workflow, Kubernetes and quotas live in `scripts/integration/`. They read credentials only from environment variables.

## Online Demo

The public online demo is **temporarily unavailable**. Please use the Docker Compose deployment above to try the platform locally.

## Core Contributors

- **[GoSimplicity](https://github.com/GoSimplicity)**: project founder and core maintainer
- **[Penge666](https://github.com/Penge666)**: senior developer
- **[shixiaocaia](https://github.com/shixiaocaia)**: core contributor
- **[daihao4371](https://github.com/daihao4371)**: feature developer

[![Contributors](https://contrib.rocks/image?repo=GoSimplicity/AI-CloudOps)](https://github.com/GoSimplicity/AI-CloudOps/graphs/contributors)

## Contributing

Contributions are welcome.

1. Fork the repository and create a feature branch.
2. Follow the existing code style, run `make fmt`, `make lint` and `go test ./...`.
3. Use [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `ci:` ...).
4. Open a pull request describing the change and how it was tested.

## License

This repository is licensed under the [MIT License](LICENSE).

## Contact

- **Email:** <bamboocloudops@gmail.com>
- **WeChat:** GoSimplicity (please mention "AI-CloudOps")
- **Issues:** <https://github.com/GoSimplicity/AI-CloudOps/issues>

## Acknowledgements

Thanks to everyone who has contributed to AI-CloudOps, and to the open-source projects it builds on, including [Go](https://go.dev/), [Gin](https://gin-gonic.com/), [Kubernetes](https://kubernetes.io/), [Prometheus](https://prometheus.io/), [Vue.js](https://vuejs.org/), [Vben Admin](https://github.com/vbenjs/vue-vben-admin), [Ant Design Vue](https://antdv.com/) and [FastAPI](https://fastapi.tiangolo.com/).

---

If you find this project useful, please give it a ⭐.

[![Star History Chart](https://api.star-history.com/svg?repos=GoSimplicity/AI-CloudOps&type=Date)](https://star-history.com/#GoSimplicity/AI-CloudOps&Date)
