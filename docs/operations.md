# Operations and production-readiness notes

## Health checks

- `/healthz` is a liveness endpoint: it confirms that the HTTP process responds.
- `/readyz` is a readiness endpoint: it confirms that this instance can receive traffic.

There are no downstream dependencies today, so both return success while the process is running. If dependencies are added later, readiness should reflect whether the instance can serve requests; liveness should not fail merely because a remote dependency is temporarily unavailable.

## Rollout and recovery

- Watch `kubectl rollout status deployment/trdl -n trdl`.
- Check ready replicas and Service endpoints.
- Keep immutable image tags and the previous known-good version available.
- Roll back with `kubectl rollout undo deployment/trdl -n trdl` if a rollout is unhealthy.
- Test pod deletion and rolling updates before production.

## Example commands

```bash
kubectl -n trdl get deploy,pods,svc
kubectl -n trdl get endpoints trdl
kubectl -n trdl logs deployment/trdl
kubectl -n trdl rollout history deployment/trdl
kubectl -n trdl rollout undo deployment/trdl
```

## SLO discussion

Agree a target only after requirements and measurement are defined. For reference, approximate annual downtime budgets are:

| Availability | Downtime per 365-day year |
|---|---:|
| 99% | 3 days 15 hours |
| 99.9% | 8 hours 46 minutes |
| 99.99% | 52 minutes 34 seconds |

These figures are mathematical examples, not a proposed SLA. Define the measurement window, exclusions, request population, and responsibility boundaries before committing to a target.
