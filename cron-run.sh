#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

{
  echo "[$(date -Is)] START run"
  if ./run.sh; then
    echo "[$(date -Is)] END ok"
  else
    status=$?
    echo "[$(date -Is)] END failed (exit $status)"
    exit "$status"
  fi
} >> cron.log 2>&1
