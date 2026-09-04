#!/usr/bin/env bash
# End-to-end smoke test: drive a parcel from creation to delivery through the
# gateway, then assert the public tracking view and the notification sink caught up.
set -euo pipefail

GATEWAY="${GATEWAY:-http://localhost:8000}"
MAILHOG="${MAILHOG:-http://localhost:8025}"
API_KEY="${GATEWAY_API_KEY:-}"

hdr=(-H "Content-Type: application/json")
[ -n "$API_KEY" ] && hdr+=(-H "x-api-key: ${API_KEY}")

say() { printf '\n\033[1m%s\033[0m\n' "$1"; }
fail() { echo "  ✗ $1" >&2; exit 1; }

say "1. create a shipment"
created=$(curl -fsS "${hdr[@]}" -X POST "${GATEWAY}/api/shipments" -d '{
  "recipient_name": "Smoke Tester",
  "recipient_email": "smoke@example.com",
  "origin": "London",
  "destination": "Paris"
}')
id=$(echo "$created" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
tn=$(echo "$created" | sed -n 's/.*"tracking_number":"\([^"]*\)".*/\1/p')
[ -n "$id" ] && [ -n "$tn" ] || fail "could not parse created shipment: $created"
echo "  ✓ ${tn} (${id})"

say "2. record scans: PICKED_UP -> OUT_FOR_DELIVERY -> DELIVERED"
for ev in PICKED_UP OUT_FOR_DELIVERY DELIVERED; do
  curl -fsS "${hdr[@]}" -X POST "${GATEWAY}/api/shipments/${id}/scans" \
    -d "{\"event_type\":\"${ev}\",\"location\":\"Somewhere\"}" > /dev/null
  echo "  ✓ ${ev}"
done

say "3. wait for the tracking projection to catch up"
for _ in $(seq 1 20); do
  track=$(curl -fsS "${GATEWAY}/api/track/${tn}" || true)
  echo "$track" | grep -q '"status":"DELIVERED"' && break
  sleep 1
done
echo "$track" | grep -q '"status":"DELIVERED"' \
  || fail "tracking projection never reached DELIVERED: $track"
echo "$track" | grep -q '"eventType":"PICKED_UP"' \
  || fail "tracking timeline missing PICKED_UP: $track"
echo "  ✓ public tracking shows DELIVERED with full timeline"

say "4. check the delivery notification reached MailHog"
for _ in $(seq 1 10); do
  count=$(curl -fsS "${MAILHOG}/api/v2/messages" | sed -n 's/.*"total":\([0-9]*\).*/\1/p')
  [ "${count:-0}" -ge 1 ] && break
  sleep 1
done
[ "${count:-0}" -ge 1 ] || fail "no messages in MailHog"
echo "  ✓ MailHog has ${count} message(s)"

say "smoke test passed ✅"
