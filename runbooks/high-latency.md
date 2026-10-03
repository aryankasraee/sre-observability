# Runbook: HighLatencyP95

Alert: p95 request latency above 500 ms.

1. **Is it everywhere or one route?**
   `histogram_quantile(0.95, sum by (le, route) (rate(http_request_duration_seconds_bucket[5m])))`
2. **Is it only latency, or errors too?** If errors are up as well, work
   [slo-burn](slo-burn.md) first: it is the page.
3. **Look for saturation**, not code: CPU throttling, connection pool exhaustion,
   a slow downstream dependency, GC pauses.
4. **Mitigate:** scale out, shed load, or roll back the last change.
5. Latency alone is a ticket, not a page, until it starts causing errors.

Reproduce it safely: `curl 'localhost:18080/admin/chaos?latency_ms=800'`, then
`...?latency_ms=0` to undo.
