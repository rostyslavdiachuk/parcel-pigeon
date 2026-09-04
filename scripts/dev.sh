#!/usr/bin/env bash
# ParcelPigeon dev orchestrator — a thin wrapper over `docker compose`.
#
#   ./scripts/dev.sh up            build + start everything, migrate, seed, wait
#   ./scripts/dev.sh down          stop (keep volumes)
#   ./scripts/dev.sh reset         stop + wipe volumes + rebuild from scratch
#   ./scripts/dev.sh seed [--force] load demo shipments
#   ./scripts/dev.sh test [--e2e]  run each service's test suite (+ smoke test)
#   ./scripts/dev.sh smoke         run the end-to-end smoke test only
#   ./scripts/dev.sh lint          run every linter
#   ./scripts/dev.sh logs [svc]    tail logs
#   ./scripts/dev.sh ps            show container status
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# Prefer the `docker compose` plugin; fall back to standalone `docker-compose`.
if docker compose version >/dev/null 2>&1; then
  DC="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  DC="docker-compose"
else
  echo "error: neither 'docker compose' nor 'docker-compose' is available" >&2
  exit 1
fi

ensure_env() {
  if [ ! -f .env ]; then
    cp .env.example .env
    echo "created .env from .env.example"
  fi
}

urls() {
  cat <<EOF

  ParcelPigeon is up:
    Web UI ............ http://localhost:8080
    API gateway ...... http://localhost:8000
    Shipments API .... http://localhost:8001/docs
    Tracking API ..... http://localhost:8002/healthz
    RabbitMQ UI ...... http://localhost:15672  (guest / guest)
    MailHog UI ....... http://localhost:8025

EOF
}

cmd_up() {
  ensure_env
  $DC up -d --build
  echo "waiting for services..."
  ./scripts/wait-for.sh http://localhost:8001/healthz 90 "shipments-service"
  ./scripts/wait-for.sh http://localhost:8002/healthz 90 "tracking-service"
  ./scripts/wait-for.sh http://localhost:8000/healthz 90 "gateway"
  ./scripts/wait-for.sh http://localhost:8080/healthz 90 "web"
  cmd_seed
  urls
}

cmd_down() { $DC down; }

cmd_reset() {
  $DC down -v --remove-orphans
  $DC build --no-cache
  cmd_up
}

cmd_seed() {
  echo "seeding demo data..."
  $DC exec -T shipments-service python -m app.seed "${1:-}"
}

cmd_test() {
  local rc=0
  echo "== shipments-service (pytest) =="
  $DC run --rm --no-deps \
    -e SHIPMENTS_DATABASE_URL=sqlite+pysqlite:///:memory: \
    -e SHIPMENTS_PUBLISH_EVENTS=false \
    -e RUN_MIGRATIONS=false \
    shipments-service pytest || rc=1

  echo "== tracking-service (go test) =="
  docker run --rm -v "$ROOT/services/tracking-service":/src -w /src \
    -e GOFLAGS=-buildvcs=false golang:1.23-alpine sh -c "go test ./..." || rc=1

  echo "== gateway (vitest) =="
  docker run --rm -v "$ROOT/services/gateway":/app -w /app node:22-slim \
    sh -c "npm ci --no-audit --no-fund >/dev/null 2>&1 || npm install --no-audit --no-fund >/dev/null 2>&1; npm test" || rc=1

  echo "== web (vitest) =="
  docker run --rm -v "$ROOT/web":/app -w /app node:22-slim \
    sh -c "npm ci --no-audit --no-fund >/dev/null 2>&1 || npm install --no-audit --no-fund >/dev/null 2>&1; npm test" || rc=1

  if [ "${1:-}" = "--e2e" ]; then
    echo "== end-to-end smoke =="
    cmd_smoke || rc=1
  fi
  return $rc
}

cmd_smoke() { GATEWAY_API_KEY="$(grep -E '^GATEWAY_API_KEY=' .env 2>/dev/null | cut -d= -f2)" ./scripts/smoke.sh; }

cmd_lint() {
  local rc=0
  echo "== shipments-service (ruff) =="
  docker run --rm -v "$ROOT/services/shipments-service":/app -w /app python:3.12-slim \
    sh -c "pip install -q .[dev] >/dev/null 2>&1 && rm -rf build && ruff check ." || rc=1

  echo "== tracking-service (go vet) =="
  docker run --rm -v "$ROOT/services/tracking-service":/src -w /src \
    -e GOFLAGS=-buildvcs=false golang:1.23-alpine \
    sh -c 'unformatted=$(gofmt -l .); test -z "$unformatted" || { echo "gofmt needed: $unformatted"; exit 1; }; go vet ./...' || rc=1

  echo "== gateway (eslint + prettier) =="
  docker run --rm -v "$ROOT/services/gateway":/app -w /app node:22-slim \
    sh -c "npm install --no-audit --no-fund >/dev/null 2>&1; npm run lint" || rc=1

  echo "== web (eslint + prettier) =="
  docker run --rm -v "$ROOT/web":/app -w /app node:22-slim \
    sh -c "npm install --no-audit --no-fund >/dev/null 2>&1; npm run lint" || rc=1
  return $rc
}

cmd_logs() { $DC logs -f --tail=100 "${1:-}"; }
cmd_ps() { $DC ps; }

case "${1:-}" in
  up)    cmd_up ;;
  down)  cmd_down ;;
  reset) cmd_reset ;;
  seed)  shift; cmd_seed "${1:-}" ;;
  test)  shift; cmd_test "${1:-}" ;;
  smoke) cmd_smoke ;;
  lint)  cmd_lint ;;
  logs)  shift; cmd_logs "${1:-}" ;;
  ps)    cmd_ps ;;
  *)
    grep -E '^#( |$)' "$0" | sed 's/^# \{0,1\}//'
    exit 1
    ;;
esac
