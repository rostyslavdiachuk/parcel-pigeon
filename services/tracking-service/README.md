# tracking-service

Read-side service in Go. A single binary that runs an HTTP API **and** a RabbitMQ
consumer that folds `shipment.*` events into a Redis projection.

- Pure reducer: `internal/projection/projection.go` (`Reduce(state, msg) -> state, changed`)
- Consumer: `internal/consumer/` (durable queue `tracking.projector`, manual ack, idempotent)
- HTTP: `internal/httpapi/` — `/track/{tn}`, `/track/{tn}/events`, `/notifications`, `/healthz`, `/readyz`, `/metrics`
- Cache-miss fallback to shipments-service: `internal/upstream/`
- Delivery emails to MailHog: `internal/notify/`

## Local

```bash
go test ./...
go vet ./... && gofmt -l .
go run ./cmd/tracking      # needs redis + rabbitmq reachable, or set TRACKING_ENABLE_CONSUMER=false
```

Config: `TRACKING_HTTP_ADDR`, `TRACKING_REDIS_ADDR`, `TRACKING_RABBITMQ_URL`,
`TRACKING_EXCHANGE`, `TRACKING_QUEUE`, `TRACKING_SHIPMENTS_URL`,
`TRACKING_SMTP_ADDR`, `TRACKING_MAIL_FROM`, `TRACKING_ENABLE_CONSUMER`.
