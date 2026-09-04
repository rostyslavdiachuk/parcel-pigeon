#!/usr/bin/env bash
# Poll an HTTP endpoint until it returns 2xx/3xx or we time out.
# Usage: wait-for.sh <url> [timeout_seconds] [label]
set -euo pipefail

url="${1:?usage: wait-for.sh <url> [timeout] [label]}"
timeout="${2:-60}"
label="${3:-$url}"

deadline=$(( $(date +%s) + timeout ))
until curl -fsS -o /dev/null --max-time 3 "$url"; do
  if [ "$(date +%s)" -ge "$deadline" ]; then
    echo "  ✗ ${label} did not become ready within ${timeout}s" >&2
    exit 1
  fi
  sleep 2
done
echo "  ✓ ${label} ready"
