# Architecture

## Goals

ParcelPigeon exists to demonstrate DevOps/IDP practice on something that behaves
like a real system: multiple services, multiple languages, a database, a cache,
a message broker, and asynchronous workflows — while staying small enough to
read in an afternoon.

## Services and responsibilities

### web (React + TypeScript, served by Nginx)
Two screens:
- **Track** — public: enter a tracking number, see status + a timeline.
- **Operations** — create shipments, record scan events, watch statuses update
  (polls every 5s).

In dev, Vite proxies `/api` to the gateway. In the container, an Nginx `location
/api/` block does the same. The SPA never talks to a backend service directly.

### gateway (Node + TypeScript, Fastify)
The single ingress. Responsibilities:
- **Routing**: `/api/shipments/*` → `shipments-service`, `/api/track/*` and
  `/api/notifications` → `tracking-service` (prefix rewritten, `@fastify/http-proxy`).
- **Stub auth**: if `GATEWAY_API_KEY` is set, every non-GET `/api/*` request must
  carry a matching `x-api-key`. This is intentionally a placeholder for "verify a
  JWT / call the IdP here".
- **Cross-cutting**: CORS, a propagated `x-request-id`, RED metrics, `/healthz`,
  `/readyz` (fans out to both upstreams' health).

### shipments-service (Python + FastAPI, the write side)
Owns the authoritative data in PostgreSQL:
- `shipment` and `scan_event` tables (Alembic migration `0001`).
- The **state machine** lives in `app/shipment_status.py` — a pure module with no
  imports from the framework or the DB. `next_status(current, event_type)` either
  returns the new status or raises `InvalidTransition` (→ HTTP 409).
- After a successful write it publishes a JSON event to the `parcelpigeon` topic
  exchange: `shipment.created`, `shipment.status_changed`, or `shipment.scan_added`.
- Publishing failures are logged, counted (`shipment_events_published_total`), and
  retried once — they do not fail the API call.

### tracking-service (Go, the read side)
One binary, two jobs:
- **Consumer** (`internal/consumer`): binds a durable queue `tracking.projector`
  to `shipment.#`, folds each event through the pure reducer
  (`internal/projection.Reduce`) and stores the result in Redis under
  `track:<trackingNumber>`. The reducer is **idempotent** (keyed on
  `occurredAt` + `eventType`) so redelivery is safe, and **order-independent**
  (a late older scan is recorded without rewinding the headline status).
- **HTTP API** (`internal/httpapi`): `GET /track/{tn}` serves from Redis
  (`tracking_cache_hits_total`); on a miss it falls back to
  `shipments-service`'s `/internal/track/{tn}`, warms the cache, and returns
  (`tracking_cache_misses_total`).
- On the first transition into `DELIVERED` it sends a plain-text email through
  MailHog and appends to the `track:notifications` Redis list.

## Data flow

```
create shipment / add scan
  → shipments-service: validate transition, write Postgres
  → publish shipment.status_changed  ──▶  RabbitMQ (parcelpigeon, topic)
                                            │
                              tracking.projector queue (shipment.#)
                                            │
  tracking-service: Get(redis) → Reduce(state, event) → Set(redis)
                                            │
                              if now DELIVERED and was not:
                                 send email → MailHog
                                 LPUSH track:notifications
```

Read path:

```
GET /api/track/PP-XXXX
  → gateway → tracking-service
      → Redis hit?  yes → return
                    no  → GET shipments-service/internal/track/PP-XXXX
                          → shape into read model → Set(redis) → return
```

## Why these boundaries

- **CQRS-lite**: writes and reads scale and fail independently. Killing
  `tracking-service` does not stop parcels being created; killing `redis` only
  slows tracking (it falls back to the source of truth).
- **Event-carried state transfer**: the tracking read model is rebuildable purely
  from the event stream — a good hook for talking about replay, DLQs, and
  schema evolution later.
- **Pure cores**: the two functions that encode all the business rules have zero
  infrastructure dependencies and exhaustive table tests.

## Configuration

Everything is environment variables, prefixed per service (`SHIPMENTS_*`,
`TRACKING_*`, and plain names for the gateway). `docker-compose.yml` is the only
place they are wired together. Defaults in the code point at the compose service
names, so a service started with no env still works inside the network.

## What is deliberately missing in Phase 1

No CI, no Prometheus/Grafana/Alertmanager, no tracing, no Kubernetes/Helm, no
Argo CD, no Backstage. The application is structured so each of those attaches
without touching service code — see [lecture-map.md](lecture-map.md).
