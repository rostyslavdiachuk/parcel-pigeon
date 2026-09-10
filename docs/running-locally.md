# Running ParcelPigeon locally (no app containers)

On this branch (`local-no-docker`) only the **datastores** run in Docker. The four
application services — `shipments-service`, `tracking-service`, `gateway`, `web` —
run **natively** on your machine. Writing the per-service Dockerfiles and the full
`docker-compose.yml` is the Containerisation lecture exercise; the finished version
lives on `main` (`git diff main` to compare).

## Prerequisites

| Tool | Version | For |
|------|---------|-----|
| Docker + Compose | any recent | datastores only (`./scripts/dev.sh up`) |
| Python | ≥ 3.12 | shipments-service |
| Node.js | ≥ 20 (22 recommended) | gateway, web |
| Go | ≥ 1.22 | tracking-service |
| `curl`, `jq` | — | the smoke test / manual walkthrough |

## Ports

| Process | Port | Notes |
|---------|------|-------|
| web (Vite dev server) | **5173** | proxies `/api` → `http://localhost:8000` |
| gateway | **8000** | edge / reverse proxy |
| shipments-service | **8001** | run uvicorn on `--port 8001` (in Docker it is `8000` inside the container, published as `8001`) |
| tracking-service | **8002** | HTTP API + RabbitMQ consumer |
| postgres | 5432 | |
| redis | 6379 | |
| rabbitmq | 5672 | management UI on 15672 (guest / guest) |
| mailhog | 1025 (SMTP) | web UI on 8025 |

---

## Step 0 — datastores

```bash
./scripts/dev.sh up
```

Starts Postgres 16, Redis 7, RabbitMQ 3.13, MailHog and waits for Postgres to accept
connections. `./scripts/dev.sh down` stops them (keeps data); `./scripts/dev.sh reset`
also wipes the volumes.

---

## Step 1 — shipments-service (Python / FastAPI, port 8001)

```bash
cd services/shipments-service

python -m venv .venv && source .venv/bin/activate
pip install -e ".[dev]"

export SHIPMENTS_DATABASE_URL="postgresql+psycopg://parcelpigeon:parcelpigeon@localhost:5432/shipments"
export SHIPMENTS_RABBITMQ_URL="amqp://guest:guest@localhost:5672/"

alembic upgrade head          # create the schema (Docker did this via docker-entrypoint.sh)
python -m app.seed            # load demo shipments PP-DEMO0001..0004  (add --force to wipe first)

uvicorn app.main:app --host 0.0.0.0 --port 8001 --reload
```

`alembic upgrade head` + `python -m app.seed` are also available as
`./scripts/dev.sh migrate` and `./scripts/dev.sh seed` (they set the same env vars).

Check: <http://localhost:8001/docs>, <http://localhost:8001/healthz>.

---

## Step 2 — tracking-service (Go, port 8002)

```bash
cd services/tracking-service

export TRACKING_REDIS_ADDR="localhost:6379"
export TRACKING_RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
export TRACKING_SHIPMENTS_URL="http://localhost:8001"
export TRACKING_SMTP_ADDR="localhost:1025"

go run ./cmd/tracking
```

Runs the HTTP API and the RabbitMQ consumer. To run the read API without RabbitMQ,
add `export TRACKING_ENABLE_CONSUMER=false`.

Check: <http://localhost:8002/healthz>, <http://localhost:8002/readyz>.

---

## Step 3 — gateway (Node / Fastify, port 8000)

```bash
cd services/gateway
npm install

export SHIPMENTS_URL="http://localhost:8001"
export TRACKING_URL="http://localhost:8002"

npm run dev          # tsx watch src/server.ts
```

Check: <http://localhost:8000/healthz>, <http://localhost:8000/readyz> (fans out to
both upstreams).

---

## Step 4 — web (React / Vite, port 5173)

```bash
cd web
npm install
npm run dev          # http://localhost:5173
```

Vite already proxies `/api` to `http://localhost:8000` (override with
`GATEWAY_URL`). Open <http://localhost:5173> and track `PP-DEMO0001`.

---

## Verify the whole flow

```bash
./scripts/dev.sh smoke
```

Creates a shipment through the gateway, records `PICKED_UP → OUT_FOR_DELIVERY →
DELIVERED`, asserts the public tracking view caught up, and checks MailHog received
the delivery email. Or drive it by hand with the `curl` walkthrough in the root
`README.md` ("Try the flow by hand").

---

## Environment variables

Every service is configured entirely by environment variables. In-code defaults
target the Docker Compose service names (`postgres`, `redis`, `rabbitmq`,
`shipments-service`); running natively means overriding the host/URL to `localhost`.

### shipments-service (`SHIPMENTS_` prefix, also reads `.env` in its cwd)

| Var | Docker default | Local value |
|-----|----------------|-------------|
| `SHIPMENTS_DATABASE_URL` | `postgresql+psycopg://parcelpigeon:parcelpigeon@postgres:5432/shipments` | `…@localhost:5432/shipments` |
| `SHIPMENTS_RABBITMQ_URL` | `amqp://guest:guest@rabbitmq:5672/` | `amqp://guest:guest@localhost:5672/` |
| `SHIPMENTS_RABBITMQ_EXCHANGE` | `parcelpigeon` | same |
| `SHIPMENTS_PUBLISH_EVENTS` | `true` | `true` (`false` disables publishing) |
| `SHIPMENTS_ETA_HOURS` | `72` | same |
| `SHIPMENTS_LOG_LEVEL` | `INFO` | same |
| `RUN_MIGRATIONS` | `true` (used only by the Docker entrypoint) | n/a — run `alembic upgrade head` yourself |

### tracking-service (`TRACKING_` prefix)

| Var | Docker default | Local value |
|-----|----------------|-------------|
| `TRACKING_HTTP_ADDR` | `:8002` | same |
| `TRACKING_REDIS_ADDR` | `redis:6379` | `localhost:6379` |
| `TRACKING_RABBITMQ_URL` | `amqp://guest:guest@rabbitmq:5672/` | `amqp://guest:guest@localhost:5672/` |
| `TRACKING_EXCHANGE` | `parcelpigeon` | same |
| `TRACKING_QUEUE` | `tracking.projector` | same |
| `TRACKING_SHIPMENTS_URL` | `http://shipments-service:8000` | `http://localhost:8001` |
| `TRACKING_SMTP_ADDR` | `mailhog:1025` | `localhost:1025` |
| `TRACKING_MAIL_FROM` | `no-reply@parcelpigeon.test` | same |
| `TRACKING_ENABLE_CONSUMER` | `true` | `true` (`false` to skip RabbitMQ) |

### gateway (plain names, zod-validated)

| Var | Docker default | Local value |
|-----|----------------|-------------|
| `PORT` | `8000` | `8000` |
| `HOST` | `0.0.0.0` | same |
| `SHIPMENTS_URL` | `http://shipments-service:8000` | `http://localhost:8001` |
| `TRACKING_URL` | `http://tracking-service:8002` | `http://localhost:8002` |
| `GATEWAY_API_KEY` | *(empty — stub auth off)* | set to require `x-api-key` on non-GET `/api/*` |
| `LOG_LEVEL` | `info` | same |

### web

| Var | Default | Purpose |
|-----|---------|---------|
| `GATEWAY_URL` | `http://localhost:8000` | where the Vite dev server proxies `/api` |
| `VITE_API_KEY` | *(unset)* | sent as `x-api-key` on non-GET requests (only if the gateway sets `GATEWAY_API_KEY`) |

---

## Troubleshooting

- **`shipments-service` on 8001 vs 8000** — the app itself listens on whatever
  `--port` you give `uvicorn`. Everything downstream (`gateway`, `tracking-service`,
  the smoke test, the docs) expects **8001**, so use `--port 8001`.
- **`/readyz` returns 503** — a dependency is down. `gateway` `/readyz` needs both
  backends; `tracking-service` `/readyz` pings Redis; `shipments-service` `/readyz`
  runs `SELECT 1` and checks the broker.
- **Want just the read side?** Start Redis + `tracking-service` with
  `TRACKING_ENABLE_CONSUMER=false` — it serves `/track/*` from Redis (or falls back
  to `shipments-service` on a cache miss).
- **Port already in use** — 5173/8000/8001/8002 must be free; stop any previous run.
- **RabbitMQ / MailHog UIs** — <http://localhost:15672> (guest/guest),
  <http://localhost:8025>.
