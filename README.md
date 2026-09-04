# ParcelPigeon 📦

A small, fully working polyglot microservice system built as a **sandbox for
teaching DevOps and Internal Developer Platform (IDP) practices** — Docker,
CI/CD, static analysis, metrics, Kubernetes, Helm, GitOps.

This is **Phase 1**: a system you can run locally with one script. Later phases
(CI pipelines, Prometheus/Grafana, Kubernetes, Helm, Argo CD, Backstage) layer
on top without changing the application code — see [docs/lecture-map.md](docs/lecture-map.md).

## The domain

A parcel courier. Operators create **shipments** and record **scan events**
("picked up", "out for delivery", "delivered"). Each scan drives a state
machine and publishes a domain event. A separate read-side service folds those
events into a fast public **tracking** view and emails the recipient when the
parcel is delivered.

## Architecture

```
Browser ──▶ web (React/TS, Nginx)
              │  /api/*
              ▼
           gateway (Node/TS, Fastify)   ── reverse proxy + stub auth + metrics
              │                    │
   /api/shipments/*         /api/track/*, /api/notifications
              ▼                    ▼
   shipments-service        tracking-service (Go)
   (Python, FastAPI)          │        ▲
     │        │               │ Redis  │ consume shipment.*
     │     Postgres           ▼        │
     │                     read model  │
     └── publish shipment.* ──▶ RabbitMQ ┘
                                    │ on DELIVERED
                                    ▼
                                 MailHog (fake email)
```

| Component          | Language / stack            | Port  | Role |
|--------------------|-----------------------------|-------|------|
| `web`              | React + TypeScript + Vite   | 8080  | Track page + operations dashboard |
| `gateway`          | Node + TypeScript + Fastify | 8000  | Edge: routing, stub `x-api-key` auth, `/metrics` |
| `shipments-service`| Python + FastAPI + SQLAlchemy| 8001 | Write side: shipments, scans, state machine, event publishing |
| `tracking-service` | Go + chi                    | 8002  | Read side: RabbitMQ consumer → Redis projection, delivery emails |
| `postgres`         | PostgreSQL 16               | 5432  | Shipment store |
| `redis`            | Redis 7                     | 6379  | Tracking read model + notification feed |
| `rabbitmq`         | RabbitMQ 3.13 (+ mgmt UI)   | 5672 / 15672 | `parcelpigeon` topic exchange |
| `mailhog`          | MailHog                     | 1025 / 8025  | Captures delivery notification emails |

Every service also exposes `/healthz`, `/readyz`, and Prometheus `/metrics`
(there is no Prometheus container yet — that arrives in a later phase).

## Requirements

- Docker + Docker Compose (`docker compose` plugin **or** standalone `docker-compose`)
- Bash, `curl` (for the helper scripts)

Nothing else — every language toolchain runs inside a container.

## Quick start

```bash
./scripts/dev.sh up        # build images, start everything, migrate, seed, wait
```

Then open:

- **http://localhost:8080** — the web app (try tracking `PP-DEMO0001`)
- http://localhost:8000/healthz — gateway
- http://localhost:8001/docs — shipments-service OpenAPI
- http://localhost:15672 — RabbitMQ (guest / guest)
- http://localhost:8025 — MailHog (you should see a "Parcel PP-DEMO0001 delivered" email)

### `scripts/dev.sh` commands

| Command | What it does |
|---------|--------------|
| `up` | `docker compose up -d --build`, run migrations, seed demo data, wait for readiness |
| `down` | Stop containers (keep volumes) |
| `reset` | Stop, delete volumes, rebuild with `--no-cache`, start again |
| `seed [--force]` | (Re)load the demo shipments |
| `test [--e2e]` | Run every service's test suite in a container; `--e2e` also runs the smoke test |
| `smoke` | End-to-end check: create → scan → track → assert MailHog got the email |
| `lint` | Run ruff / go vet+gofmt / eslint+prettier for all services |
| `logs [svc]` | Tail logs |
| `ps` | Container status |

## Try the flow by hand

```bash
# create a shipment through the gateway
curl -s -XPOST localhost:8000/api/shipments -H 'content-type: application/json' -d '{
  "recipient_name":"Jane Doe","recipient_email":"jane@example.com",
  "origin":"Kyiv","destination":"Lviv"}' | tee /tmp/s.json

ID=$(jq -r .id /tmp/s.json); TN=$(jq -r .tracking_number /tmp/s.json)

# record scans (state machine: CREATED → IN_TRANSIT → OUT_FOR_DELIVERY → DELIVERED)
for E in PICKED_UP OUT_FOR_DELIVERY DELIVERED; do
  curl -s -XPOST localhost:8000/api/shipments/$ID/scans \
    -H 'content-type: application/json' -d "{\"event_type\":\"$E\",\"location\":\"Hub\"}" >/dev/null
done

# public tracking view (served from the Redis projection)
curl -s localhost:8000/api/track/$TN | jq
```

An illegal transition (e.g. `DELIVERED` straight from `CREATED`) returns `409`.

## Repository layout

```
services/
  gateway/            Node + TypeScript (Fastify) reverse proxy
  shipments-service/  Python + FastAPI, Alembic migrations, pytest
  tracking-service/   Go, RabbitMQ consumer + HTTP read API, go test
web/                  React + TypeScript + Vite, Vitest
scripts/
  dev.sh              orchestrator
  wait-for.sh         HTTP readiness poller
  smoke.sh            end-to-end test
docs/
  architecture.md     how the pieces fit and why
  api.md              endpoint reference
  lecture-map.md      which DevOps topic to demo where (fills in over phases)
```

## Design notes for the lectures

- **Testable core**: the shipment state machine
  (`services/shipments-service/app/shipment_status.py`) and the tracking reducer
  (`services/tracking-service/internal/projection/projection.go`) are pure
  functions with no I/O — the obvious place to talk about unit testing.
- **CQRS-lite**: `shipments-service` owns the write model in Postgres;
  `tracking-service` maintains an independent read model in Redis, fed only by
  events. The reducer is idempotent, so redelivered messages are safe.
- **12-factor config**: every service is configured entirely by environment
  variables (`docker-compose.yml` is the single wiring point).
- **Ops endpoints everywhere**: `/healthz` (liveness), `/readyz` (checks
  dependencies), `/metrics` (Prometheus) — ready for k8s probes and scraping
  in later phases.
