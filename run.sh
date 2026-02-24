#!/usr/bin/env bash
set -euo pipefail

ACCESS_TOKEN="$(oauth2l fetch \
  --credentials .client_secret.json \
  --scope "https://www.googleapis.com/auth/contacts.readonly,https://www.googleapis.com/auth/calendar" \
  --cache .oauth2l-cache.json \
  --refresh \
  2>/dev/null | tr -d '\r\n' || true)"

if [[ -z "$ACCESS_TOKEN" ]]; then
  echo "No cached refresh token found in .oauth2l-cache.json. Run oauth2l once manually first." >&2
  exit 1
fi

./importer -token "$ACCESS_TOKEN" "$@"
