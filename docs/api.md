# API reference

All traffic goes through the **gateway** at `http://localhost:8000`. The backend
services are also published directly (`:8001`, `:8002`) so you can inspect them
in isolation during a lecture.

When `GATEWAY_API_KEY` is set in `.env`, non-GET `/api/*` calls must send
`x-api-key: <value>`.

## Shipments (write side) — via `/api/shipments`

### `POST /api/shipments`
Create a shipment.

```json
{
  "recipient_name": "Ada Lovelace",
  "recipient_email": "ada@example.com",
  "origin": "London",
  "destination": "Paris"
}
```

`201` → the shipment, including a generated `tracking_number` (`PP-XXXXXXXX`),
`status: "CREATED"`, and an `eta` 72h out. Publishes `shipment.created`.

### `GET /api/shipments`
List shipments, newest first.

### `GET /api/shipments/{id}`
One shipment with its `scans` array.

### `POST /api/shipments/{id}/scans`
Record a scan event. Drives the state machine.

```json
{ "event_type": "OUT_FOR_DELIVERY", "location": "Paris Depot", "note": null }
```

`event_type` ∈ `PICKED_UP`, `IN_TRANSIT`, `ARRIVED_AT_FACILITY`,
`OUT_FOR_DELIVERY`, `DELIVERED`, `DELIVERY_FAILED`, `RETURNED`.

- `201` → the created scan; publishes `shipment.status_changed` (or
  `shipment.scan_added` if the status did not change).
- `409` → the transition is illegal for the current status.
- `404` → no such shipment.

### `GET /api/shipments/{id}/scans`
The scan history.

## Tracking (read side) — via `/api/track`

### `GET /api/track/{trackingNumber}`
Public tracking view, served from the Redis projection (falls back to
`shipments-service` on a cache miss, then `404`).

```json
{
  "trackingNumber": "PP-DEMO0001",
  "status": "DELIVERED",
  "previousStatus": "OUT_FOR_DELIVERY",
  "origin": "London",
  "destination": "Paris",
  "recipientName": "Ada Lovelace",
  "eta": "2026-09-04T10:00:00Z",
  "events": [
    { "status": "IN_TRANSIT", "eventType": "PICKED_UP", "location": "London Hub", "occurredAt": "..." }
  ],
  "updatedAt": "..."
}
```

### `GET /api/track/{trackingNumber}/events`
Just `{ trackingNumber, status, events }`.

### `GET /api/notifications`
The last ~50 simulated delivery emails (also visible in MailHog at `:8025`).

## Ops endpoints (every service)

| Path | Meaning |
|------|---------|
| `GET /healthz` | Liveness — always `200 {"status":"ok"}` if the process is up |
| `GET /readyz`  | Readiness — checks DB / Redis / broker; `503` if a dependency is down |
| `GET /metrics` | Prometheus exposition format |

## Domain events (RabbitMQ)

- Exchange `parcelpigeon`, type `topic`, durable.
- Routing keys: `shipment.created`, `shipment.status_changed`, `shipment.scan_added`.
- `tracking-service` binds queue `tracking.projector` to `shipment.#`.
- Payload:

```json
{
  "trackingNumber": "PP-DEMO0001",
  "status": "OUT_FOR_DELIVERY",
  "previousStatus": "IN_TRANSIT",
  "eventType": "OUT_FOR_DELIVERY",
  "location": "Paris Depot",
  "occurredAt": "2026-09-03T09:00:00+00:00",
  "recipientEmail": "ada@example.com",
  "recipientName": "Ada Lovelace",
  "origin": "London",
  "destination": "Paris",
  "eta": "2026-09-03T17:00:00+00:00"
}
```
