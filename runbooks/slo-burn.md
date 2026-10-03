# Runbook: SLO burn-rate alerts

Alerts: `SLOFastBurn` (page), `SLOSlowBurn` (page), `SLOBudgetBurnTicket` and
`SLOBudgetLeak` (ticket). All say the same thing at different speeds: errors are
eating the monthly error budget faster than the SLO allows.

| Alert | Burn rate | Windows (long and short) | Budget gone in |
|---|---|---|---|
| SLOFastBurn | 14.4x | 1h and 5m | 2 hours at this pace |
| SLOSlowBurn | 6x | 6h and 30m | 5 hours |
| SLOBudgetBurnTicket | 3x | 1d and 2h | 10 days |
| SLOBudgetLeak | 1x | 3d and 6h | exactly the end of the window |

Both windows must exceed the threshold. The long one proves it is sustained; the
short one proves it is still happening, so the alert clears quickly after a fix.

## 1. Confirm and scope (2 minutes)

- Open the dashboard (`http://localhost:13000`, "shop-api SLO overview").
- Which status code and which route? `sum by (route, code) (rate(http_requests_total[5m]))`
- When did it start? Compare with the last deploy, config change, or dependency incident.

## 2. Find the cause

- **Errors on one route only:** look at the logs for that route
  (`{service="shop-api"} | json | route="/checkout" | code >= 500`).
- **Errors on everything, latency also up:** a shared dependency (database, cache,
  network) is the usual cause; check saturation before blaming the code.
- **Started exactly at a deploy:** roll back first, investigate second.

## 3. Mitigate

Stop the bleeding before you understand it: roll back, shift traffic, disable the
feature flag, scale out. Then keep watching the short-window ratio; the page
resolves when it drops below the threshold.

## 4. After

- Write down start, detection time, mitigation time, and cause.
- Detection time is a number worth tracking: the drill in this repo gives you the
  baseline for how fast the alert *can* be.
- If the budget is below zero, the SLO says to stop shipping features and spend
  the time on reliability until it recovers.

## Notes

- A `SLOSlowBurn` that fires together with `SLOFastBurn` is suppressed
  (Alertmanager inhibition): one page per problem.
