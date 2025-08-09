# golang-skeleton

A reusable starting point for Go microservices and a React frontend in one monorepo.
Includes a working demo: a users API, an event-driven worker, and a web directory.

- **Backend** - Go services with shared packages, PostgreSQL, and NATS JetStream.
- **Frontend** - React, TypeScript, and Vite; served by Nginx in production.
- **Local development** - Docker Compose with Air for Go and Vite HMR for React.
- **Containers** - a dedicated multi-stage Dockerfile for each service.
- **Deployment** - Kubernetes manifests with Kustomize dev and prod overlays.
- **CI/CD** - GitHub Actions for checks, container builds, image publishing, and opt-in deployment.

## Layout

```
├── compose.yaml              # base stack: postgres, nats, api, worker, web
├── compose.override.yaml     # dev overlay (auto-loaded): Air + bind mounts
├── compose.prod.yaml         # prod-like overlay (opt-in)
├── pkg/                      # shared code: config, logger, httpx, postgres, natsx, events
├── services/
│   ├── api/                  # HTTP service (chi) - users API, publishes events
│   │   ├── cmd/              # entrypoint
│   │   ├── internal/         # handler, store (pgx + embedded migrations)
│   │   ├── Dockerfile        # multi-stage: build → distroless release / air dev
│   │   └── .air.toml
│   ├── worker/               # background service - JetStream durable consumer
│   │   ├── cmd/, Dockerfile, .air.toml
│   └── web/                  # React team directory, Vite dev / Nginx release
│       ├── src/, e2e/        # Vitest unit tests, Playwright browser tests
│       └── Dockerfile        # node build → nginx release / vite dev
├── deploy/k8s/               # kustomize: base + overlays/{dev,prod}
├── .github/workflows/        # ci.yml (lint/test/docker/e2e), cd.yml (GHCR + deploy)
├── scripts/                  # demo.sh (API), smoke.sh (full-stack e2e)
├── Makefile
└── .env.example
```

Module path is a single `go.mod` at the root (`github.com/cookiebinary1/golang-skeleton`).
Services import shared code via `pkg/`; service-private code lives under
`services/<name>/internal/`.

## Quickstart

One-time setup, then everything runs in Docker - the fastest way to see the
whole stack (hot reload via Air/Vite included):

```bash
make up             # creates .env if missing, builds and starts postgres, nats, api, worker, web
```

Web: **http://localhost:3000**.

## Everyday commands

```bash
make demo           # exercise the API
make logs           # tail all services
make down           # stop, keep data
make nuke           # stop and delete data volumes
```

Prod-like single-host run (distroless images, JSON logs, no hot reload):

```bash
make up-prod        # = compose -f compose.yaml -f compose.prod.yaml up -d --build
```

Services on the host instead (needs Go + Node locally; postgres/nats stay in
Docker - handy for faster reloads and debugger attachment):

```bash
make tools          # installs Air and golangci-lint; ensure $(go env GOPATH)/bin is on PATH
make dev-api        # deps in Docker (postgres+nats), api local via Air
make dev-worker     # same for worker (second terminal)
make web-install    # npm ci; Node 24 LTS recommended
make dev-web        # React at http://localhost:3000 (third terminal)
make demo           # exercise the API
```

Compose files use the modern `compose.yaml` naming. `compose.override.yaml` is
merged automatically on every `docker compose` command in dev; the prod file is
opt-in via `-f`.

## Web example: team directory

Open **http://localhost:3000**. The React UI lists users from Postgres, creates a
user with a name/email form, filters by name or email, and fetches a person's
profile from the API. Duplicate emails and network errors are shown in the UI.
Entries persist across page reloads. This creates directory entries; it does not
send invitation emails.

The API currently exposes:

- `GET /api/v1/users`: list users.
- `POST /api/v1/users`: create `{ "name": "Alex Morgan", "email": "alex@example.com" }`.
- `GET /api/v1/users/{email}`: fetch one user.
- `GET /healthz` and `GET /readyz`: process/dependency checks.

The browser always calls `/api` on the web origin. Vite proxies to
`API_PROXY_TARGET` (default `http://127.0.0.1:8080`) locally, and `http://api:8080`
in Compose. Nginx proxies the same path in release images. No build-time backend
URL and no permissive CORS setup are required. The Kubernetes API Service uses
port 8080; the web Service uses port 80. For cluster access:

```bash
kubectl -n golang-skeleton-dev port-forward service/web 3000:80
```

`make web-check` runs frontend unit tests and a TypeScript/production build.
`cd services/web && npm run test:e2e` tests the running stack in Chromium
(first run `npx playwright install chromium`). CI runs the browser flow against
both Compose modes and uploads desktop/mobile screenshots and failure traces.

## Demo flow

`POST /api/v1/users` commits the user and an outbox row in one Postgres transaction.
A background relay publishes `events.user.created` to persistent NATS JetStream
and deletes the outbox row only after the publish acknowledgement. Failed
publishes are retried. The worker consumes it with a durable consumer (`worker`) and
logs it. Delivery is at least once. Message IDs suppress short-window duplicates;
real consumers must make side effects idempotent. This demo logs payloads and
retries consumption up to five times. The demo stream retains up to 64 MiB
or seven days - tune retention for your workload. Add dead-letter handling for
real workloads. `GET /readyz` on all services checks dependencies (DB ping /
NATS connection). Migrations are embedded in the api binary and applied at
startup (`services/api/internal/store/migrations/`).

## Local development modes

| Mode | Command | What runs where |
|---|---|---|
| Full Docker | `make up` | Air/Vite inside containers, source bind-mounted |
| Local Air | `make dev-api` / `make dev-worker` / `make dev-web` | Air + service on host, postgres/nats in Docker |
| Prod-like | `make up-prod` | distroless/nginx images, no mounts |

Air configs live per service (`services/*/.air.toml`) and run from the repository
root. Both service code and shared `pkg/` changes trigger rebuilds. Install tools with
`make tools` (Air, golangci-lint). Go toolchain is auto-downloaded as declared
in `go.mod`.

If a local Postgres or NATS already occupies the default ports, override the
published host ports in `.env`: `POSTGRES_HOST_PORT` (5432), `NATS_HOST_PORT`
(4222), `NATS_HTTP_PORT` (8222). These only affect the dev Compose stack;
containers always talk over the internal network.

## Kubernetes

```bash
make k8s-apply-dev    # namespace golang-skeleton-dev, 1 replica
make k8s-apply-prod   # namespace golang-skeleton, 2 replicas
```

- `deploy/k8s/base/` - postgres (Deployment+PVC+Secret), nats (JetStream, PVC),
  api, worker, web (Nginx): readiness (`/readyz`) and liveness (`/healthz`)
  probes, resource requests/limits, nonroot + read-only-root security contexts.
- `overlays/dev|prod/` - namespace, replicas, image tags, log level. Image
  overrides cover all three services (api, worker, web).
- Only the dev overlay generates demo credentials. Create the production Secret
  externally before applying the prod overlay (example below). For production DB prefer
  CloudNativePG or a managed database; for NATS a StatefulSet cluster or helm chart.

Before applying the production overlay:

```bash
kubectl create namespace golang-skeleton
# Supply values from your secret manager. Do not commit real credentials.
kubectl -n golang-skeleton create secret generic postgres-credentials \
  --from-literal=POSTGRES_USER=app \
  --from-literal=POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
  --from-literal=POSTGRES_DB=app
```

The bundled DB and NATS manifests are single-instance demo infrastructure with
PVCs, not HA production infrastructure. The dev overlay uses published `latest`
images. For local cluster builds, load your images into kind/minikube and change
the overlay image names/tags before applying. Application startup probes tolerate
dependency startup; application containers run as nonroot with a read-only root.
If GHCR packages are private, make them public or configure an image pull Secret.

## CI/CD

- `ci.yml`: lint, Go race tests, web unit tests + production build, Docker
  builds for all three services, Compose/Kustomize validation, and dev/release
  Compose smoke tests including the Playwright browser flow (screenshots and
  failure traces uploaded as artifacts). Integration tests assert user storage,
  duplicate protection, escaped-email routing, delivery to the worker, broker
  outage recovery, persistent JetStream storage, and shared-code Air reload in dev.
- `cd.yml`: after a successful push CI run on `main`, publish all service images to GHCR
  with `sha-<full commit>` and `latest` tags. Deploy uses the tested revision and
  applies manifests with the final image tags in one operation.
- Cluster deployment is disabled by default. Enable repository variable
  `DEPLOY_ENABLED=true`, create the `production` environment and configure
  `KUBE_CONFIG` (base64 kubeconfig) only when a cluster is ready. Provision the
  production database Secret first. No cluster access is needed to use this repo.

The API is intentionally unauthenticated demo functionality. Add authentication,
authorization, pagination and workload-specific limits before exposing a real app.
Compose publishes development ports on loopback only.

## Adding a new service

1. `services/<name>/cmd/main.go` - wire `config`, `logger`, `httpx.NewRouter`
   (health endpoints for free), and any pkg you need.
2. Copy a thin multi-stage `Dockerfile` from an existing service (build context
   stays the repo root, so shared `pkg/` is included).
3. Add `.air.toml` + a `dev-<name>` Makefile target, add the service to
   `compose.yaml`, kustomize base and CI/CD matrices.
4. Non-Go services (see `web/`): skip Air, give the Dockerfile a `dev` stage
   with the language's dev server, and add a CI job for its tests/build.

## Renaming the module

```bash
# replace module path in go.mod, then update all imports:
go mod edit -module github.com/<you>/<project>
grep -rl 'github.com/cookiebinary1/golang-skeleton' --include='*.go' . | xargs \
  sed -i '' 's|github.com/cookiebinary1/golang-skeleton|github.com/<you>/<project>|g'
```
