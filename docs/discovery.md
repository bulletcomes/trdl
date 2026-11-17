# Discovery

## 1. Understanding of the task

- The service must respond with `42` on `GET /`.
- Show how this would run as a highly available production service in a public cloud.
- Constraints: at most 10 hours, no cloud account assumed, deliverables reviewed from the repository.

## 2. Requirements

<!-- Number every requirement from the directions so the tasks in section 5 can point to them. Quote the directions where you can. -->

| ID | Requirement (from the directions) | How I will show it's met |
|---|---|---|
| R1 | HTTP service responding `42` | Go service + unit tests |
| R2 | Runs in a container | Multi-stage Dockerfile |
| R3 | Deployable to Kubernetes / locally | k3d + `overlays/local` |
| R4 | Highly available in a public cloud | `overlays/gke`: 3 zones, PDB, KEDA autoscaling; documented, not deployed |
| R5 | CI/CD | GitHub Actions: test, validate, build, scan, publish |
| R6 | Documentation / developer experience | README, solution.md, operations.md |
| R7 | Discovery + time estimate | This document |

## 3. Assumptions

- No cloud account available, so the cloud deployment is designed and validated by rendering it, not by running it.
- The availability target isn't specified, so I don't commit to an SLA; see operations.md.
- A public registry (Docker Hub) is acceptable for the proof of concept.
- The reviewer has Docker, kubectl and k3d to run it locally.
- Traffic volume is unknown, so resource requests and autoscaling thresholds are starting values.

## 4. Open questions

| Question | Why it matters | Blocking? | Assumed answer |
|---|---|---|---|
| Which cloud provider? | Decides the overlay, load balancer and IAM | No | GKE |
| What availability target, or SLA? | Decides the number of zones/regions and the replica count | No | Multi-zone, one region |
| Is a real cloud deployment expected? | Decides the scope and the cost | No | No |
| Expected traffic and latency? | Sizing, autoscaling | No | Low |
| Any security or compliance requirements? | Scanning, signing, network policy | No | Baseline only |

## 5. Tasks and estimate

| # | Task | Covers | Estimate | Notes |
|---|---|---|---:|---|
| 1 | Read the directions, write this discovery | R7 | 0.75 h | |
| 2 | Go service + unit tests (`/`, probes, 404/405) | R1 | 0.5 h | |
| 3 | Dockerfile, non-root, distroless | R2 | 0.5 h | |
| 4 | Base manifests + `local` overlay, k3d setup, Ingress | R3 | 1.0 h | Includes the 404 host-rule troubleshooting |
| 5 | `gke` reference overlay (zone spread, PDB, LB, TLS) | R4 | 1.0 h | |
| 5b | KEDA autoscaling: local demo (Prometheus, ScaledObject, k6) + GKE ScaledObject | R4 | 1.25 h | Optional; cut first if time runs short |
| 6 | CI: tests, manifest render, build, Trivy, publish | R5 | 1.25 h | Includes the Go stdlib CVE fix and the Docker Hub switch |
| 7 | Short-SHA image tags + `/version` endpoint | R5 | 0.25 h | Shows which build is running |
| 8 | Failure and rollout checks (pod kill, rolling update, smoke test) | R4 | 1.0 h | The k6 Job from 5b can be reused for the rolling-update check |
| 9 | Docs: README, solution, operations | R6 | 1.0 h | |
| | **Total** | | **8.5 h** | |
| | Contingency | | 1.5 h | |

## 6. Out of scope

- A real GKE deployment: no cloud account, and it costs money.
- CI deploying to a cluster: a hosted runner can't reach k3d; the design is described in solution.md.
- DNS, real TLS, IAM / Workload Identity Federation.
- Release-based publishing. The proof of concept publishes a short-SHA image on every push to `main`. The intended flow is that pull requests build and push short-SHA images for testing a change before merge, and creating a GitHub release publishes the release-tagged image that gets deployed:

  ```yaml
  on:
    pull_request:
    release:
      types: [published]

  jobs:
    publish:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4

        - name: Choose image tag
          id: tag
          run: |
            if [ "${{ github.event_name }}" = "release" ]; then
              echo "tag=${GITHUB_REF_NAME#v}" >> "$GITHUB_OUTPUT"   # v1.2.0 -> 1.2.0
            else
              sha="${{ github.event.pull_request.head.sha }}"
              echo "tag=${sha::7}" >> "$GITHUB_OUTPUT"              # PR head commit
            fi

        - uses: docker/login-action@v3
          with:
            username: ${{ secrets.DOCKERHUB_USERNAME }}
            password: ${{ secrets.DOCKERHUB_TOKEN }}

        - uses: docker/build-push-action@v6
          with:
            context: app
            build-args: VERSION=${{ steps.tag.outputs.tag }}
            push: true
            tags: docker.io/bulletcomes/trdl:${{ steps.tag.outputs.tag }}
  ```

  Pull requests from forks don't receive repository secrets, so they can build but not push.

## 7. Risks

| Risk | Impact | Mitigation |
|---|---|---|
| GKE overlay never applied to a real cluster | May contain mistakes only GKE would catch | Rendered in CI; uses only documented GKE resources; listed as a limitation |
| Single-server k3d isn't really HA | Local demo overstates resilience | Stated explicitly in solution.md's limitations |
| Docker Hub pull rate limits | Image pulls fail | Move to Artifact Registry for production |
| Running over 10 h | Assignment constraint | Track time per task; cut 5b first |

## 8. Retrospective (after the work)

<!-- Short: where the estimate was right or wrong, and why. Reviewers value this. -->

- [Biggest difference between estimate and actual, and why.]
- [What I would estimate differently next time.]
