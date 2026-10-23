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
go run main.go
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
| `JWT_KEY1`, `JWT_KEY2`, `JWT_ISSUER`, `JWT_EXPIRATION` | JWT signing keys and settings. `JWT_EXPIRATION` is in minutes. |
| `TREE_PASSWORD_ENCRYPTION_KEY` | Key used to encrypt stored host passwords. Keep it safe: changing it breaks decryption of existing credentials. |
| `BOOTSTRAP_ADMIN_PASSWORD` | Creates the `admin` account on the first start with an empty database (min. 12 characters). Remove it after the first successful start. Existing users are never reset. |
| `SSH_KNOWN_HOSTS` | Path to an operator-verified OpenSSH `known_hosts` file used for web terminal and SSH connections. |
| `AIOPS_BASE_URL` | URL of the AIOps service, e.g. `http://aiops:8080`. Enables the `/api/v1/*` gateway. |
| `AIOPS_API_KEY` | Shared service key sent to the AIOps service as `X-API-Key`. Must match the Python service. Never put it in frontend config. |
| `MOCK_ENABLED` | Seed mock data for development. Defaults to `true`, or `false` when `ENV=production`; enabling it in production is rejected at startup. |
| `MOCK_ADMIN_PASSWORD` | Password for the mock `admin` account (min. 12 characters), only used when mock is enabled and the user table is empty. If unset, no mock admin is created. There is no built-in password. |
| `LLM_API_KEY`, `LLM_BASE_URL`, `ALIYUN_ACCESS_KEY_ID`, `ALIYUN_ACCESS_KEY_SECRET`, `TAVILY_API_KEY` | External service credentials (environment variables only). |
| `NOTIFICATION_EMAIL_*`, `NOTIFICATION_FEISHU_*` | Email (SMTP) and Feishu notification channels. |
| `WEBHOOK_*` | Settings for the alert webhook service in `cmd/webhook`. |

### Production security requirements

With `ENV=production` the backend refuses to start unless:

- `JWT_KEY1` and `JWT_KEY2` are distinct, at least 32 characters, and not the repository sample values.
- `TREE_PASSWORD_ENCRYPTION_KEY` is exactly 32 bytes and not the sample value.
- Mock is disabled (the default when `ENV=production`; an explicit `MOCK_ENABLED=true` is rejected).
- `AIOPS_API_KEY` is at least 32 characters whenever `AIOPS_BASE_URL` is set.

Other security behaviour to be aware of:

- In production, SSH connections require `SSH_KNOWN_HOSTS`. Connections are refused if the file is missing or the host key does not match.
- Browser origins are checked against `SERVER_ALLOWED_ORIGINS` (exact match) or same-origin, for both HTTP and WebSocket.
- Every protected request and token refresh re-checks the current account. Frozen, deleted or renamed accounts are rejected immediately, and authentication fails closed if the database is unavailable.
- `POST /api/user/signup` requires an `admin` session. New users are normal accounts by default.
- Non-admin users need explicit role permissions for each `/api/v1/` path they call. Service accounts no longer bypass role checks.

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
