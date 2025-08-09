#!/usr/bin/env bash
# Demo flow against the running stack: create a user, list users, fetch one.
set -euo pipefail

API_URL="${API_URL:-http://127.0.0.1:8080}"
EMAIL="demo-$(date +%s)@example.com"

echo "== POST /api/v1/users =="
curl --fail-with-body -sS -X POST "$API_URL/api/v1/users" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"Martin\",\"email\":\"$EMAIL\"}"
echo

echo "== GET /api/v1/users =="
curl --fail-with-body -sS "$API_URL/api/v1/users"
echo

echo "== GET /api/v1/users/{email} =="
curl --fail-with-body -sS "$API_URL/api/v1/users/$EMAIL"
echo

echo "== watch worker logs for the user.created event =="
echo "docker compose logs -f worker"
