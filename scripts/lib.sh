#!/usr/bin/env bash
# shellcheck disable=SC2034  # the URLs below are used by the scripts that source this file
set -euo pipefail

PROM=http://localhost:19090
AM=http://localhost:19093
LOKI=http://localhost:13100
GRAFANA=http://localhost:13000
APP=http://localhost:18080

# firing alert names, one per line
firing() {
  curl -fsS "$PROM/api/v1/alerts" | jq -r '.data.alerts[] | select(.state=="firing") | .labels.alertname' | sort -u
}

wait_for() { # description, timeout-seconds, command...
  local desc=$1 timeout=$2; shift 2
  local start=$SECONDS
  until "$@" >/dev/null 2>&1; do
    if (( SECONDS - start > timeout )); then echo "timed out waiting for: $desc" >&2; return 1; fi
    sleep 2
  done
}
