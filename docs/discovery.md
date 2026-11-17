# Discovery

## 1. Understanding of the task

- The service must respond with `42` on `GET /`.
- Show how this would run as a highly available production service in a public cloud.
- Constraints: at most 10 hours, deliverables reviewed from the repository.
- Simplicity first: the core (service, container, Kubernetes manifests, CI) is deliberately minimal. KEDA autoscaling (task 5b) is an extra a service that returns `42` doesn't need; I had time left and wanted to try an approach from a previous job. It is isolated, so it can be dropped without touching the core.

## 2. Requirements

| ID | From the directions | How I will show it's met |
|---|---|---|
| R1 | The service contract: `GET /` returns `42` | Go service; the HTTP contract in the README |
| R2 | "Start by making a discovery … and then the time you think this will take for each task" | This document |
| R3 | "Provide reasoning and justifications why you have planned the system the way you have" | [solution.md](solution.md#design-choices): design choices with reasons |
| R4 | "It should run on public cloud infrastructure … in a container orchestrator" | Kubernetes on GKE Autopilot, three zones, HTTPS load balancer; k3d for local development |
| R5 | "Ensure that the developer understands what are the additional requirements that TRDL needs to implement to deploy and run the PoC" | [What TRDL needs to provide](solution.md#what-trdl-needs-to-provide) |
| R6 | "When in doubt, err on the side of simplicity and doing less work" | A minimal core; KEDA kept optional and isolated (section 1) |
| R7 | "Put some thought into error and edge cases in the implementation" | 404, 405 with `Allow`, timeouts, graceful shutdown; [errors and edge cases](solution.md#errors-and-edge-cases) |
| R8 | "How automated testing of some kind should be done, either unit testing … or end-to-end testing" | Unit tests and an end-to-end smoke test, both run in CI |

## 3. Assumptions

- A GKE Autopilot cluster in my own Google Cloud project is acceptable for the proof of concept, deployed by hand and torn down afterwards.
- The availability target isn't specified, so I don't commit to an SLA; see solution.md.
- A public registry (Docker Hub) is acceptable for the proof of concept.
- The reviewer has Docker, kubectl and k3d to run it locally.
- Traffic volume is unknown, so resource requests and autoscaling thresholds are starting values.

## 4. Open questions

| Question | Why it matters | Blocking? | Assumed answer |
|---|---|---|---|
| Which cloud provider? | Decides the overlay, load balancer and IAM | No | GKE |
| What availability target, or SLA? | Decides the number of zones/regions and the replica count | No | Multi-zone, one region |
| Is a real cloud deployment expected? | Decides the scope and the cost | No | Yes: GKE Autopilot |
| Expected traffic and latency? | Sizing, autoscaling | No | Low |
| Any security or compliance requirements? | Scanning, signing, network policy | No | Baseline only |

## 5. Tasks and estimate

| # | Task | Covers | Estimate | Notes |
|---|---|---|---:|---|
| 1 | Read the directions, write this discovery | R2 | 0.75 h | |
| 2 | Go service + unit tests (`/`, probes, 404/405, edge cases) | R1, R7, R8 | 0.5 h | |
| 3 | Dockerfile, non-root, distroless | R4 | 0.5 h | |
| 4 | Base manifests + `local` overlay, k3d setup, Ingress | R4 | 1.0 h | Includes the 404 host-rule troubleshooting |
| 5 | `gke` overlay (zone spread, PDB, LB, TLS) and deploy to GKE Autopilot | R4 | 1.0 h | Deploying found two errors in the KEDA trigger |
| 5b | KEDA autoscaling: local demo (Prometheus, ScaledObject, k6) + GKE ScaledObject | R4, R6 | 1.25 h | Optional extra, done with spare time; cut first if time runs short |
| 6 | CI: tests, end-to-end smoke test, manifest render, build, Trivy, publish | R8, R4 | 1.25 h | Includes the Go stdlib CVE fix and the Docker Hub switch |
| 7 | Short-SHA image tags + `/version` endpoint | R4 | 0.25 h | Shows which build is running |
| 8 | Failure and rollout checks (pod kill, rolling update, smoke test) | R8 | 1.0 h | Done: pod kill on k3d, smoke checks on GKE. Not done: rolling update under load |
| 9 | Docs: README, solution, operations | R3, R5 | 1.0 h | |
| | **Total** | | **8.5 h** | |
| | Contingency | | 1.5 h | |

## 6. Out of scope

- Automated deployment: GKE is deployed by hand. In production, Flux (GitOps) would deploy from this repository, with pull-request review as the approval (described in solution.md).
- A real domain: the certificate is for an sslip.io name derived from the static IP.
- Release-based publishing: the proof of concept publishes an image on every push to `main`. In production, pull requests would publish test images and a GitHub release would publish the version that gets deployed.

## 7. Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Cloud costs while the cluster runs | Unexpected bill | Autopilot bills per pod; delete the cluster and static IP after review (commands in operations.md) |
| Single-server k3d isn't really HA | Local demo overstates resilience | Stated explicitly in solution.md's limitations |
| Docker Hub pull rate limits | Image pulls fail | Move to Artifact Registry for production |
| Running over 10 h | Assignment constraint | Track time per task; cut 5b first |

## 8. Retrospective (after the work)

<!-- Short: where the estimate was right or wrong, and why. Reviewers value this. -->

- [Biggest difference between estimate and actual, and why.]
- [What I would estimate differently next time.]
