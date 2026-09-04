# shipments-service

Write-side service. FastAPI + SQLAlchemy 2 + Alembic on PostgreSQL. Owns the
shipment state machine and publishes domain events to RabbitMQ.

- Business rules: `app/shipment_status.py` (pure, no I/O)
- Use cases: `app/service.py` (shared by routers and the seed script)
- HTTP: `app/routers/` — `shipments.py`, `health.py`; `/metrics` in `app/main.py`
- Events: `app/broker.py` (pika, lazy connection, publish-with-retry)
- Migrations: `alembic/` (applied by `docker-entrypoint.sh` on start)

## Local

```bash
pip install -e ".[dev]"
export SHIPMENTS_DATABASE_URL=sqlite+pysqlite:///:memory: SHIPMENTS_PUBLISH_EVENTS=false
pytest
ruff check .
```

Config: `SHIPMENTS_DATABASE_URL`, `SHIPMENTS_RABBITMQ_URL`, `SHIPMENTS_EXCHANGE`,
`SHIPMENTS_PUBLISH_EVENTS`, `SHIPMENTS_ETA_HOURS`, `SHIPMENTS_LOG_LEVEL`,
`RUN_MIGRATIONS`.
