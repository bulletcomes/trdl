# Technical solution and proof-of-concept report

## 1. Objective and scope

The objective is to demonstrate a small HTTP service and a credible path to operating it as a highly available production service in a public cloud. The application itself is deliberately simple: `GET /` returns the text `42`.

The proof of concept focuses on delivery and runtime concerns rather than adding application complexity. It includes an executable service, tests, a container image, Kubernetes resources, local deployment instructions, and an automated CI pipeline.

The implementation is intended to be completed and reviewed within the assignment's ten-hour constraint. Cloud infrastructure provisioning and a real cloud deployment are explicitly outside the local proof-of-concept boundary because cloud account access is not assumed.

## 2. Overall technical approach

The target production architecture is:

1. A client resolves a public DNS name and connects over HTTPS.
2. A cloud load balancer receives traffic and forwards it to the cluster ingress/gateway.
3. A Kubernetes Service distributes requests to ready application pods.
4. A Deployment maintains the desired number of stateless replicas and performs rolling updates.
5. Container images are built and tested by CI, scanned, and published to a registry.
6. A controlled deployment process updates the workload to a new immutable image version.

For the local demonstration, k3d provides a Kubernetes cluster running in Docker. This demonstrates container packaging, Kubernetes scheduling, service discovery, probes, and rolling-update mechanics. It does not reproduce cloud-zone or managed-control-plane availability.

## 3. Technology choices

| Area | Choice | Reason |
|---|---|---|
| Application | Go standard library | Small dependency footprint, fast startup, simple HTTP implementation |
| Container | Multi-stage build, distroless runtime | Small runtime surface and no shell/package manager in the final image |
| Orchestration | Kubernetes | Widely supported public-cloud workload platform |
| Local cluster | k3d | Reproducible Kubernetes environment without cloud access |
| Configuration | Kustomize | Native Kubernetes manifest customization without introducing another packaging layer |
| CI | GitHub Actions | Integrated with source control and straightforward for a small project |
| Image registry | GHCR | Can be used without assuming a cloud account |
| Cloud target | Regional GKE | Managed Kubernetes option; regional control plane and multi-zone node placement can support resilience |

The app is deliberately stateless. There is no database, cache, queue, or persistent volume because none is required by the service contract.

## 4. Availability and resilience

The Kubernetes Deployment starts with two replicas. A Service sends traffic only to ready endpoints. Readiness probes prevent a pod from receiving traffic before it is ready; liveness probes allow Kubernetes to restart an unhealthy process.

A rolling update uses `maxUnavailable: 0` and `maxSurge: 1`, subject to available cluster capacity. The process handles SIGTERM and allows a short graceful-shutdown window. The container runs as a non-root user.

For production, I would use a regional managed cluster and distribute replicas across independent zones. The included GKE overlay adds a zone topology-spread preference, but this is not a substitute for provisioning multiple zones, checking node capacity, and validating scheduling behaviour. A PodDisruptionBudget should be added once replica count and maintenance policy are finalized.

Two replicas do not by themselves establish an availability SLA. Availability also depends on the load balancer, DNS, network, cluster control plane, node pools, image registry, deployment process, and operational response. An SLA needs a defined measurement window and error budget.

## 5. Developer experience and delivery

A developer should be able to clone the repository, run the tests, build the image, and deploy locally using the README. The application has no external runtime dependencies.

The GitHub Actions workflow runs on pull requests and pushes to `main`. It tests the Go code, renders the Kustomize overlays, builds the image, and scans it for high/critical vulnerabilities. A push to `main` publishes an immutable commit-SHA image tag to GHCR.

The workflow does not deploy to a cluster. A hosted CI runner cannot reach a laptop's local k3d cluster. For a real GKE deployment, I would add a separate protected deployment job using Workload Identity Federation, with environment approvals and a smoke test after rollout. I would avoid long-lived service-account keys.

## 6. Testing strategy

Automated unit tests cover:
- The successful root response and content type.
- Unknown paths (`404`).
- Unsupported methods (`405` and `Allow: GET`).
- Health and readiness endpoints.

The next integration checks in a real cluster should verify:
- All replicas become ready.
- The Service has ready endpoints.
- Requests continue during a rolling update.
- Deleting a pod causes the Deployment to replace it.
- A process that fails a liveness probe is restarted.
- A pod is removed from Service endpoints before termination.
- The externally exposed endpoint returns `42`.

A minimal smoke test after deployment should issue `GET /` and compare the response body with `42`.

## 7. Monitoring and operations

For production I would define a small set of service-level indicators:
- Request success ratio (successful requests divided by valid requests).
- Request rate.
- HTTP 5xx rate.
- Request latency (at least p50, p95, and p99).
- Pod readiness and restart count.
- CPU and memory utilization.

The service should emit structured request logs to stdout/stderr. In GKE these can be collected by the cloud logging agent. Metrics and traces should be integrated with the chosen platform; OpenTelemetry is a reasonable instrumentation standard if tracing becomes useful.

Alerts should focus on user-visible symptoms (elevated 5xx rate, latency, or unavailable replicas), not merely CPU usage. A tiny constant-response service is unlikely to benefit from CPU-based autoscaling. If traffic-driven scaling becomes a requirement, evaluate request-rate or concurrency metrics. KEDA could be introduced for event/metric-driven scaling, but it is not necessary for this proof of concept and would add components to operate.

## 8. Security

The initial controls are:
- Non-root runtime user.
- Distroless runtime image.
- No application secrets or credentials.
- Kubernetes pod security context and seccomp RuntimeDefault.
- Resource requests and limits.
- Automated image vulnerability scanning.
- Immutable image tags based on commit SHA.

Before production I would also configure image provenance/signing, dependency update automation, network policies, least-privilege IAM, secret management, admission policy, and a documented patch process.

## 9. Discovery phase and indicative estimate

The following is an estimate for a small engineering team, assuming access to a cloud account and an existing GitHub repository. It is not a claim that the cloud work has been performed.

| Activity | Estimate |
|---|---:|
| Clarify availability, traffic, latency, security, and operational requirements | 0.75 h |
| Confirm cloud landing-zone, networking, IAM, and registry assumptions | 0.75 h |
| Application contract and implementation | 1.0 h |
| Containerization and local Kubernetes deployment | 1.5 h |
| CI pipeline, validation, and image publishing | 1.5 h |
| Operational design, monitoring, and security review | 1.0 h |
| Failure/rollout tests and documentation | 1.5 h |
| Review, handover, and contingency | 1.0 h |
| **Total** | **9.0 h** |

This is a proof-of-concept estimate, not a production delivery estimate. A production implementation would require separate estimates for cloud networking, cluster provisioning, DNS/TLS, IAM, monitoring, security controls, load testing, disaster recovery, and operational readiness.

## 10. Known limitations and next steps

1. The local k3d cluster has one control-plane server; it is not a highly available control plane.
2. Local Docker networking and a local ingress do not validate a public cloud load balancer or DNS/TLS.
3. The GKE overlay contains a placeholder image repository and example hostname.
4. The CI workflow publishes to GHCR but does not perform a cloud deployment.
5. Resource requests and limits are starting values and must be validated with measurements.
6. The service is so lightweight that autoscaling is not meaningful without a realistic traffic profile.

Recommended next steps are to provision a regional GKE environment, configure secure CI identity, deploy the GKE overlay with a real registry image, expose it through HTTPS, run failure and load tests, and agree measurable SLOs before making availability commitments.
