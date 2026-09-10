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
| `web`              | React + TypeScript + Vite   | 5173  | Track page + operations dashboard (Vite dev server; Nginx on :8080 on `main`) |
| `gateway`          | Node + TypeScript + Fastify | 8000  | Edge: routing, stub `x-api-key` auth, `/metrics` |
| `shipments-service`| Python + FastAPI + SQLAlchemy| 8001 | Write side: shipments, scans, state machine, event publishing |
| `tracking-service` | Go + chi                    | 8002  | Read side: RabbitMQ consumer → Redis projection, delivery emails |
| `postgres`         | PostgreSQL 16               | 5432  | Shipment store |
| `redis`            | Redis 7                     | 6379  | Tracking read model + notification feed |
| `rabbitmq`         | RabbitMQ 3.13 (+ mgmt UI)   | 5672 / 15672 | `parcelpigeon` topic exchange |
| `mailhog`          | MailHog                     | 1025 / 8025  | Captures delivery notification emails |

Every service also exposes `/healthz`, `/readyz`, and Prometheus `/metrics`
(there is no Prometheus container yet — that arrives in a later phase).

> **This branch (`local-no-docker`)** ships no app Dockerfiles. Only the
> datastores run in Docker; the four services run natively on the host. Writing
> the per-service Dockerfiles and the full app `docker-compose.yml` is the
> Containerisation lecture exercise — `git diff main` shows the reference
> implementation. The web app is served by Vite on **:5173** here (not Nginx on
> :8080), and `shipments-service` runs on **:8001** directly.

## Requirements

- Docker + Docker Compose — for the datastores only
- Python ≥ 3.12, Node.js ≥ 20 (22 recommended), Go ≥ 1.22 — for the services
- Bash, `curl`, `jq` (for the helper scripts)

## Quick start

```bash
./scripts/dev.sh up        # start postgres / redis / rabbitmq / mailhog
```

then start each service natively — full step-by-step in
**[docs/running-locally.md](docs/running-locally.md)**:

| Service | Command (from its directory) | URL |
|---------|------------------------------|-----|
| shipments-service | `alembic upgrade head && python -m app.seed && uvicorn app.main:app --port 8001 --reload` | http://localhost:8001/docs |
| tracking-service | `go run ./cmd/tracking` | http://localhost:8002/healthz |
| gateway | `npm run dev` | http://localhost:8000/healthz |
| web | `npm run dev` | **http://localhost:5173** (try tracking `PP-DEMO0001`) |

Datastore UIs: http://localhost:15672 (RabbitMQ, guest / guest),
http://localhost:8025 (MailHog).

### `scripts/dev.sh` commands

| Command | What it does |
|---------|--------------|
| `up` | `docker compose up -d` (datastores), wait for Postgres |
| `down` | Stop datastores (keep volumes) |
| `reset` | Stop datastores, delete volumes |
| `migrate` | `alembic upgrade head` for shipments-service |
| `seed [--force]` | (Re)load the demo shipments |
| `smoke` | End-to-end check: create → scan → track → assert MailHog got the email |
| `logs [svc]` | Tail datastore logs |
| `ps` | Datastore container status |

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
  dev.sh              datastore orchestrator (+ migrate / seed / smoke)
  wait-for.sh         HTTP readiness poller
  smoke.sh            end-to-end test
docs/
  running-locally.md  step-by-step: start every service natively
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
