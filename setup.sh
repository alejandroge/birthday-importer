#!/usr/bin/env bash
set -euo pipefail

oauth2l fetch \
  --credentials .client_secret.json \
  --scope "https://www.googleapis.com/auth/contacts.readonly,https://www.googleapis.com/auth/calendar" \
  --cache .oauth2l-cache.json
