#!/usr/bin/env bash
# Break the service on purpose and check the whole signal path:
#   errors -> metrics -> burn-rate alert -> Alertmanager routing, and logs in Loki.
# Then undo the fault and check the alert clears.
set -euo pipefail
cd "$(dirname "$0")/.."
source scripts/lib.sh
mkdir -p results

echo "waiting for the SLI to exist"
wait_for "SLI recorded" 120 bash -c "curl -fsS '$PROM/api/v1/query?query=slo:http_error_ratio:rate1h' | jq -e '.data.result | length > 0'"

echo "baseline: 30s of healthy traffic must not alert"
sleep 30
base=$(firing || true)
[[ -z "$base" ]] || { echo "alerts firing on healthy traffic: $base" >&2; exit 1; }

echo "injecting 30% errors (about 60x the error budget burn rate)"
curl -fsS "$APP/admin/chaos?error_rate=0.3" >/dev/null
t0=$SECONDS
wait_for "SLOFastBurn firing" 240 bash -c "curl -fsS '$PROM/api/v1/alerts' | jq -e '.data.alerts[] | select(.labels.alertname==\"SLOFastBurn\" and .state==\"firing\")'"
fire_s=$((SECONDS - t0))
echo "SLOFastBurn firing after ${fire_s}s"

echo "alert reached Alertmanager and was routed to the pager receiver"
wait_for "alert in Alertmanager" 60 bash -c "curl -fsS '$AM/api/v2/alerts' | jq -e '.[] | select(.labels.alertname==\"SLOFastBurn\" and ((.receivers|map(.name))|index(\"pager\")))'"

echo "the failing requests are searchable in Loki"
wait_for "error logs in Loki" 90 bash -c "curl -fsSG '$LOKI/loki/api/v1/query_range' --data-urlencode 'query={service=\"shop-api\"} | json | code >= 500' --data-urlencode 'since=5m' | jq -e '.data.result | length > 0'"

echo "the dashboard is provisioned"
curl -fsS "$GRAFANA/api/dashboards/uid/slo-overview" | jq -e '.dashboard.title' >/dev/null

echo "removing the fault"
curl -fsS "$APP/admin/chaos?error_rate=0" >/dev/null
t1=$SECONDS
wait_for "SLOFastBurn resolved" 300 bash -c "! curl -fsS '$PROM/api/v1/alerts' | jq -e '.data.alerts[] | select(.labels.alertname==\"SLOFastBurn\" and .state==\"firing\")'"
resolve_s=$((SECONDS - t1))
echo "SLOFastBurn resolved after ${resolve_s}s"

jq -n --argjson fire "$fire_s" --argjson resolve "$resolve_s" \
  '{profile:"lab (windows scaled by ~1/30)", injected_error_rate:0.3, seconds_to_fire:$fire, seconds_to_resolve:$resolve, routed_to:"pager", logs_in_loki:true, dashboard_provisioned:true}' \
  | tee results/drill.json
