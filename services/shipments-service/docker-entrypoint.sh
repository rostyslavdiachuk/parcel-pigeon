#!/bin/sh
set -e

# Wait for Postgres, then apply migrations before starting the API.
if [ "${RUN_MIGRATIONS:-true}" = "true" ]; then
  echo "shipments-service: running alembic upgrade head"
  alembic upgrade head
fi

exec "$@"
