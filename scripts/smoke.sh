#!/usr/bin/env bash
# Run against an isolated Compose stack. Fails on missing persistence or delivery.
set -euo pipefail
API_URL="${API_URL:-http://127.0.0.1:8080}"
WORKER_URL="${WORKER_URL:-http://127.0.0.1:8081}"
wait_ready() {
  curl --fail --silent --show-error --retry 90 --retry-connrefused --retry-all-errors --retry-delay 1 "$1/readyz"
}
wait_event() {
  for i in {1..45}; do
    if docker compose logs worker 2>&1 | grep -F "$1"; then return; fi
    sleep 1
  done
  echo 'Worker did not receive the event' >&2
  return 1
}
create_user() {
  curl --fail-with-body -sS -H 'Content-Type: application/json' \
    -d "{\"name\":\"Smoke\",\"email\":\"$1\"}" "$API_URL/api/v1/users"
}
wait_ready "$API_URL"
wait_ready "$WORKER_URL"
WEB_URL="${WEB_URL:-http://127.0.0.1:3000}"
curl --fail -sS --retry 90 --retry-connrefused --retry-all-errors --retry-delay 1 "$WEB_URL/" | grep -F 'id="root"'
curl --fail -sS "$WEB_URL/api/v1/users" > /dev/null
EMAIL="smoke-$(date +%s)-$RANDOM@example.com"
body="$(create_user "$EMAIL")"
[[ "$body" == *"$EMAIL"* ]]
code="$(curl -sS -o /dev/null -w '%{http_code}' -H 'Content-Type: application/json' -d "{\"name\":\"Smoke\",\"email\":\"$EMAIL\"}" "$API_URL/api/v1/users")"
[[ "$code" == 409 ]]
curl --fail -sS "$API_URL/api/v1/users/$EMAIL" | grep -F "$EMAIL"
wait_event "$EMAIL"

# A committed user must still reach the worker after a broker outage.
docker compose stop nats
OFFLINE_EMAIL="outage-$EMAIL"
create_user "$OFFLINE_EMAIL" | grep -F "$OFFLINE_EMAIL"
# Recreate rather than restart to verify JetStream uses the named volume.
docker compose up -d --force-recreate nats
wait_ready "$API_URL"
wait_ready "$WORKER_URL"
wait_event "$OFFLINE_EMAIL"

if [[ "${MODE:-}" == dev ]]; then
  api_before="$(docker compose logs api | grep -c 'starting service')"
  worker_before="$(docker compose logs worker | grep -c 'starting service')"
  cp pkg/events/events.go /tmp/golang-skeleton-events.go
  trap 'cp /tmp/golang-skeleton-events.go pkg/events/events.go' EXIT
  printf '\n// Smoke test: a shared source change must rebuild both services.\n' >> pkg/events/events.go
  reloaded=false
  for i in {1..60}; do
    if [[ "$(docker compose logs api | grep -c 'starting service')" -gt "$api_before" ]] && \
       [[ "$(docker compose logs worker | grep -c 'starting service')" -gt "$worker_before" ]]; then
      reloaded=true
      break
    fi
    sleep 1
  done
  [[ "$reloaded" == true ]]
  api_before="$(docker compose logs api | grep -c 'starting service')"
  worker_before="$(docker compose logs worker | grep -c 'starting service')"
  cp /tmp/golang-skeleton-events.go pkg/events/events.go
  trap - EXIT
  restored=false
  for i in {1..60}; do
    if [[ "$(docker compose logs api | grep -c 'starting service')" -gt "$api_before" ]] && \
       [[ "$(docker compose logs worker | grep -c 'starting service')" -gt "$worker_before" ]]; then
      restored=true
      break
    fi
    sleep 1
  done
  [[ "$restored" == true ]]
  wait_ready "$API_URL"
  wait_ready "$WORKER_URL"
fi
echo 'Smoke test passed: storage, duplicate protection, delivery, broker recovery, shared hot reload (dev).'
