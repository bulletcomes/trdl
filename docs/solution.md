# Solution

`GET /` returns `42`. The service is trivial on purpose; the work is in running it as a highly available service on public cloud. It runs on GKE Autopilot in `europe-north1`, across three zones, at `https://136-82-211-146.sslip.io/`, and on k3d for local development. Tasks and estimates are in [discovery.md](discovery.md).

## Design choices

Guiding rule: the simplest setup that is still credibly highly available.

| Area | Choice | Reason |
|---|---|---|
| Application | Go standard library | No dependencies; the router handles methods, 404 and 405 |
| Container | Multi-stage, distroless, non-root | Small attack surface, no shell |
| Orchestrator | Kubernetes, configured with Kustomize | Portable between k3d and GKE; one base, one overlay per environment |
| Cloud | Regional GKE Autopilot | Nodes in three zones without managing node pools |
| Zone spread | 3 replicas, hard spread across zones (`maxSkew: 1`) | Losing a zone loses at most a third of capacity |
| Disruptions | PodDisruptionBudget, `maxUnavailable: 0` rollouts | Upgrades and deploys never take more than one pod away |
| Traffic | GKE Ingress, container-native load balancing, health check on `/readyz` | Load balancer sends traffic straight to ready pods in every zone |
| TLS | Google-managed certificate, HTTP disabled | No certificates to handle |
| CI | GitHub Actions | Tests, smoke test, Trivy scan, publish; lives with the code |
| Registry | Docker Hub | Works from k3d and GKE without setup |

**Autoscaling is an optional extra.** A service that returns `42` doesn't need it; I had time left and wanted to try an approach from a previous job. KEDA scales on requests per second (2–6 replicas locally, 3–9 on GKE, never below one pod per zone), with a fixed fallback if its metrics are unavailable. It is isolated: the `local` overlay runs without it, and on GKE it is two files that can be dropped. On k3d, a load test at 400 req/s scaled from 2 to 4 replicas and back, with 0 of 44,150 requests failing.

Replicas alone don't make an SLA: that also needs an agreed target, monitoring and someone on call (see [What TRDL needs to provide](#what-trdl-needs-to-provide)).

## Errors and edge cases

| Case | Behaviour | Why |
|---|---|---|
| Unknown path, including `/42` and `/healthz/` | `404` | `/` matches only the root, so a typo isn't answered with `42` |
| Wrong method on a known path | `405` with `Allow: GET, HEAD` | Tells the client what is allowed |
| `HEAD` | As `GET`, no body | Expected by HTTP; used by uptime checks |
| Query string (`/?x=1`) | Ignored, `200` | No parameters to validate |
| Slow client headers | Closed after 5 s | `ReadHeaderTimeout` (slowloris protection) |
| Pod shutdown | `preStop` waits 2 s, then in-flight requests finish within 10 s | Traffic stops before the pod closes; fits the 15 s grace period |
| Not ready, or hung | Readiness removes it from traffic; liveness restarts it | Separate probes, so a future dependency outage won't cause restart loops |

## Testing

- **Unit tests**, run in CI: every row of the HTTP contract and the cases above, through the real router and an HTTP server ([app/main_test.go](../app/main_test.go)).
- **End-to-end smoke test**, run in CI against the built container, and by hand against k3d or GKE after a deploy ([scripts/smoke-test.sh](../scripts/smoke-test.sh)).
- **Checked by hand:** pod deletion on k3d (75 of 75 requests succeeded), autoscaling under load on k3d, and HTTPS, zones and certificate on GKE.

## Delivery

CI publishes an immutable short-SHA image on `main`; `/version` reports it. In the proof of concept, GKE is deployed by hand with `kubectl apply -k deployment/overlays/gke`. In production, **Flux** would deploy from this repository: a reviewed pull request that changes the image tag is the approval, CI never holds cluster credentials, and rollback is a `git revert`.

## What TRDL needs to provide

**What the service guarantees:** the [HTTP contract](../README.md#service-contract); `/healthz` and `/readyz` are safe for probes and load balancer checks; `/version` reports the commit SHA; published images are immutable.

**To run the proof of concept:**

| TRDL provides | Used in |
|---|---|
| A GitHub repository with Actions, and Docker Hub secrets `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN` | `.github/workflows/ci.yml` |
| A Google Cloud project with billing, and a regional GKE cluster with Workload Identity | [Deploying to GKE](operations.md#deploying-to-gke) |
| Enough regional quota (notably `SSD_TOTAL_GB`) for an extra node per zone during rollouts | Autopilot adds nodes on demand; a quota limit blocked one during a rollout |
| A global static IP `trdl-ip`, and a domain pointing at it (the proof of concept uses sslip.io) | `ingress-patch.yaml`, `managedcertificate.yaml` |
| The project ID, and KEDA with `roles/monitoring.viewer` (only if autoscaling is kept) | `scaledobject.yaml` |
| The image tag to run | `newTag` in `overlays/gke/kustomization.yaml` |

**To build before production:** Flux for deployment; monitoring and alerting on 5xx rate, latency and available replicas; network policies, least-privilege IAM and image signing; a patch process for Go and base-image fixes; Artifact Registry instead of Docker Hub (rate limits).


## Limitations

1. k3d has one control plane and no zones; zone spread is shown on GKE only.
2. GKE is deployed by hand, and uses an sslip.io name instead of a real domain.
3. Autoscaling hasn't been load-tested on GKE; thresholds and resource requests are starting values, not measurements.
4. Not yet tested: request continuity during a rolling update, and a zone failure.
